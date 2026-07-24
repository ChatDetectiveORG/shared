package handlers

import (
	"context"
	"fmt"
	"sync"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/ChatDetectiveORG/shared/amqputil"
	"github.com/google/uuid"

	amqp "github.com/rabbitmq/amqp091-go"
)

const defaultJobsBuffer = 256

// OutgoingConfig configures AMQP publishing and SendResult consumption for HandlerChainHashe.
type OutgoingConfig struct {
	Channel            *amqp.Channel
	OpenChannel        func() (*amqp.Channel, error)
	PodID              string
	OutgoingExchange   string
	SendResultExchange string
	JobsBuffer         int
	ErrorChannel       chan *e.ErrorInfo
}

// OutgoingPublisher publishes telegram.OutgoingRequest envelopes and optionally waits for SendResult.
// Use it from background workers that do not run Router/Endpoint/tele.Update flows.
type OutgoingPublisher struct {
	channel            *amqp.Channel
	publish            *amqputil.PublishChannel
	podID              string
	outgoingExchange   string
	sendResultExchange string
	jobs               chan *PublishEnvelope
	waiters            *sync.Map
	errorChannel       chan *e.ErrorInfo

	mu                  sync.Mutex
	started             bool
	sendResultConsumers map[int]bool
}

// NewOutgoingPublisher validates config and returns a publisher that must be started before Emit.
func NewOutgoingPublisher(cfg OutgoingConfig) (*OutgoingPublisher, *e.ErrorInfo) {
	if cfg.Channel == nil {
		return nil, e.NewError("RabbitmqChannel is nil", "NewOutgoingPublisher").WithSeverity(e.Critical)
	}

	outEx := cfg.OutgoingExchange
	if outEx == "" {
		outEx = defaultOutgoingExchange
	}
	inEx := cfg.SendResultExchange
	if inEx == "" {
		inEx = defaultSendResultExchange
	}

	podID := cfg.PodID
	if podID == "" {
		podID = "unknown"
	}

	buf := cfg.JobsBuffer
	if buf <= 0 {
		buf = defaultJobsBuffer
	}

	return &OutgoingPublisher{
		channel:            cfg.Channel,
		publish:            amqputil.NewPublishChannel(cfg.Channel, cfg.OpenChannel),
		podID:              podID,
		outgoingExchange:   outEx,
		sendResultExchange: inEx,
		jobs:               make(chan *PublishEnvelope, buf),
		waiters:            &sync.Map{},
		errorChannel:       cfg.ErrorChannel,
		sendResultConsumers: map[int]bool{},
	}, e.Nil()
}

// Start starts the publish loop and the SendResult consumer for shard 0.
func (p *OutgoingPublisher) Start(wg *sync.WaitGroup, ctx context.Context) *e.ErrorInfo {
	if err := p.ensurePublishLoop(wg, ctx); !err.IsNil() {
		return err
	}
	return p.EnsureSendResultConsumer(wg, 0, ctx)
}

func (p *OutgoingPublisher) ensurePublishLoop(wg *sync.WaitGroup, ctx context.Context) *e.ErrorInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return e.Nil()
	}

	p.startPublishLoop(wg, ctx)
	p.started = true
	return e.Nil()
}

// EnsureSendResultConsumer declares and binds a per-pod SendResult queue for the given shard.
// Repeated calls for the same shard are safe.
func (p *OutgoingPublisher) EnsureSendResultConsumer(wg *sync.WaitGroup, shardID int, ctx context.Context) *e.ErrorInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.sendResultConsumers == nil {
		p.sendResultConsumers = map[int]bool{}
	}
	if p.sendResultConsumers[shardID] {
		return e.Nil()
	}

	qName := sendResultQueueName(p.podID, shardID)
	q, uerr := p.channel.QueueDeclare(
		qName,
		true,
		false,
		false,
		false,
		amqp.Table{},
	)
	if uerr != nil {
		return e.FromError(uerr, "declare send result queue").WithSeverity(e.Critical)
	}
	uerr = p.channel.QueueBind(
		q.Name,
		fmt.Sprintf("%s.q%02d", p.podID, shardID),
		p.sendResultExchange,
		false,
		amqp.Table{},
	)
	if uerr != nil {
		return e.FromError(uerr, "bind send result queue").WithSeverity(e.Critical)
	}
	if err := p.startSendResultConsumer(wg, qName, ctx); !err.IsNil() {
		return err
	}
	p.sendResultConsumers[shardID] = true
	return e.Nil()
}

// RefreshSendResultConsumers rebinds SendResult queues after the main AMQP channel was recreated.
func (p *OutgoingPublisher) RefreshSendResultConsumers(ch *amqp.Channel, wg *sync.WaitGroup, ctx context.Context, replicaCount int) *e.ErrorInfo {
	if p == nil || ch == nil {
		return e.Nil()
	}

	p.mu.Lock()
	p.channel = ch
	p.sendResultConsumers = map[int]bool{}
	p.mu.Unlock()

	for i := 0; i < replicaCount; i++ {
		if err := p.EnsureSendResultConsumer(wg, i, ctx); !err.IsNil() {
			return err
		}
	}
	return e.Nil()
}

// NewHashe returns a HandlerChainHashe wired to this publisher (same Emit API as in handler chains).
func (p *OutgoingPublisher) NewHashe(mirrorID ...string) *HandlerChainHashe {
	return HandlerChainHashe{}.Init(p.jobs, p.waiters, uuid.New().String(), mirrorID...)
}

// Jobs exposes the internal publish queue (Router endpoints share the same channel).
func (p *OutgoingPublisher) Jobs() chan *PublishEnvelope {
	return p.jobs
}

// Waiters exposes correlation waiters used by EmitWait and related methods.
func (p *OutgoingPublisher) Waiters() *sync.Map {
	return p.waiters
}

func (p *OutgoingPublisher) startPublishLoop(wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for job := range p.jobs {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if job == nil {
				continue
			}
			publishOutgoingJob(p.publish, p.outgoingExchange, p.podID, p.errorChannel, job)
		}
	}()
}

func (p *OutgoingPublisher) startSendResultConsumer(wg *sync.WaitGroup, queueName string, ctx context.Context) *e.ErrorInfo {
	return startSendResultConsumer(wg, p.channel, queueName, p.waiters, p.errorChannel, ctx)
}
