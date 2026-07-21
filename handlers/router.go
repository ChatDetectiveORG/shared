package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sync"

	e "github.com/ChatDetectiveORG/shared/errors"

	amqp "github.com/rabbitmq/amqp091-go"
	tele "gopkg.in/telebot.v4"
)

// Router маршрутизирует tele.Update по endpoint-ам (фильтры) и опционально по шардам.
type Router struct {
	Endpoints       []Endpoint
	ErrorChannel    chan *e.ErrorInfo
	RabbitmqChannel     *amqp.Channel
	OpenRabbitmqChannel func() (*amqp.Channel, error)
	ReplicaCount        int
	PodID           string

	// OutgoingExchange / SendResultExchange пустые строки → дефолты из outgoing.go
	OutgoingExchange   string
	SendResultExchange string

	wg       *sync.WaitGroup
	ctx      context.Context
	mu       sync.Mutex
	replicas map[string]chan updateEnvelope

	outgoingMu sync.Mutex
	publisher  *OutgoingPublisher
}

type updateEnvelope struct {
	update   tele.Update
	mirrorID string
}

func (r *Router) shardRoutingKey(sessionID string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(sessionID))
	shard := int(h.Sum32() % uint32(r.ReplicaCount))
	return fmt.Sprintf("endpoint-%02d", shard)
}

// HandleUpdate читает session_id из заголовков, кладёт update в шард-канал (после InitSharding).
func (r *Router) HandleUpdate(delivery amqp.Delivery) *e.ErrorInfo {
	sid, _ := delivery.Headers["session_id"].(string)
	if sid == "" {
		return e.NewError("missing session_id header", "HandleUpdate").WithSeverity(e.Critical)
	}
	shard := r.shardRoutingKey(sid)
	mirrorID, _ := delivery.Headers["mirror_id"].(string)

	var update tele.Update
	if err := json.Unmarshal(delivery.Body, &update); err != nil {
		return e.FromError(err, "unmarshal update").WithSeverity(e.Critical)
	}

	ch, ok := r.replicas[shard]
	if !ok || ch == nil {
		return e.NewError("router sharding not initialized", "call InitSharding").WithSeverity(e.Critical)
	}

	select {
	case ch <- updateEnvelope{update: update, mirrorID: mirrorID}:
	default:
		return e.NewError("shard channel full", shard).WithSeverity(e.Critical)
	}
	return e.Nil()
}

// InitSharding поднимает ReplicaCount воркеров с каналами; каждый вызывает Dispatch внутри себя.
func (r *Router) InitSharding(podID string, wg *sync.WaitGroup, ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wg = wg
	r.ctx = ctx
	r.PodID = podID
	if r.ReplicaCount <= 0 {
		return
	}
	r.replicas = make(map[string]chan updateEnvelope, r.ReplicaCount)
	for i := 0; i < r.ReplicaCount; i++ {
		key := fmt.Sprintf("endpoint-%02d", i)
		r.replicas[key] = r.listenShard()
	}
}

// ConnectRabbitmqSession wires the live AMQP channel into the router on first connect and after reconnect.
func (r *Router) ConnectRabbitmqSession(ch *amqp.Channel, open func() (*amqp.Channel, error), wg *sync.WaitGroup, podID string, ctx context.Context) *e.ErrorInfo {
	if r == nil || ch == nil {
		return e.Nil()
	}

	r.RabbitmqChannel = ch
	r.OpenRabbitmqChannel = open

	r.outgoingMu.Lock()
	hasPublisher := r.publisher != nil
	replicas := r.ReplicaCount
	r.outgoingMu.Unlock()

	if replicas <= 0 {
		return e.Nil()
	}

	if !hasPublisher {
		effectivePodID := podID
		if effectivePodID == "" {
			effectivePodID = r.PodID
		}
		for i := 0; i < replicas; i++ {
			if err := r.StartOutgoing(wg, effectivePodID, i, ctx); !err.IsNil() {
				return err
			}
		}
		return e.Nil()
	}

	return r.RefreshRabbitmqSession(ch, wg, ctx)
}

func (r *Router) listenShard() chan updateEnvelope {
	updates := make(chan updateEnvelope, 1000)
	if r.wg == nil {
		return updates
	}
	r.wg.Add(1)
	ctx := r.ctx
	go func() {
		defer r.wg.Done()
		if ctx == nil {
			for u := range updates {
				r.Dispatch(u.update, u.mirrorID)
			}
			return
		}
		for {
			select {
			case u, ok := <-updates:
				if !ok {
					return
				}
				r.Dispatch(u.update, u.mirrorID)
			case <-ctx.Done():
				return
			}
		}
	}()
	return updates
}

// RefreshRabbitmqSession updates the router channel and rebinds SendResult consumers after reconnect.
func (r *Router) RefreshRabbitmqSession(ch *amqp.Channel, wg *sync.WaitGroup, ctx context.Context) *e.ErrorInfo {
	if r == nil || ch == nil {
		return e.Nil()
	}

	r.RabbitmqChannel = ch

	r.outgoingMu.Lock()
	pub := r.publisher
	replicas := r.ReplicaCount
	r.outgoingMu.Unlock()

	if pub == nil || replicas <= 0 {
		return e.Nil()
	}
	return pub.RefreshSendResultConsumers(ch, wg, ctx, replicas)
}

// Dispatch прогоняет update через все endpoint-ы с подходящим фильтром.
func (r *Router) Dispatch(update tele.Update, mirrorID string) {
	var wg *sync.WaitGroup
	r.mu.Lock()
	wg = r.wg
	r.mu.Unlock()

	for i := range r.Endpoints {
		ep := &r.Endpoints[i]
		if err := ep.runChain(update, r, wg, mirrorID); r.ErrorChannel != nil && err != nil && !err.IsNil() && err.Severity != e.Ingnored {
			r.ErrorChannel <- err.PushStack()
		}
	}
}
