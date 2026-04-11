package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"

	e "github.com/ChatDetectiveORG/shared/errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultOutgoingExchange   = "chatdetective.output.send"
	defaultSendResultExchange = "chatdetective.send.result"
)

// sendResultQueueName returns a dedicated queue per pod so SendResult is not load-balanced
// across unrelated handler processes that share the same routing key pattern.
func sendResultQueueName(podID string, shardID int) string {
	seg := strings.TrimSpace(podID)
	if seg == "" {
		seg = "unknown"
	}
	seg = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, seg)
	return fmt.Sprintf("chatdetective.send.result.%s.q%02d", seg, shardID)
}

// publishEnvelope — задание на публикацию: тело = JSON tele.Message.
type publishEnvelope struct {
	routingKey    string
	body          []byte
	correlationID string
}

// StartOutgoing поднимает publish loop один раз и consumer очереди результатов для каждого shardID.
// Повторные вызовы безопасны.
func (r *Router) StartOutgoing(wg *sync.WaitGroup, podID string, shardID int, ctx context.Context) *e.ErrorInfo {
	if r == nil {
		return e.NewError("router is nil", "StartOutgoing").WithSeverity(e.Critical)
	}
	if r.RabbitmqChannel == nil {
		return e.NewError("RabbitmqChannel is nil", "call StartOutgoing after AMQP is ready").WithSeverity(e.Critical)
	}

	r.outgoingMu.Lock()
	defer r.outgoingMu.Unlock()

	if !r.outgoingStarted {
		outEx := r.OutgoingExchange
		if outEx == "" {
			outEx = defaultOutgoingExchange
		}
		inEx := r.SendResultExchange
		if inEx == "" {
			inEx = defaultSendResultExchange
		}

		r.sendWaiters = &sync.Map{}
		r.outgoingExchange = outEx
		r.sendResultExchange = inEx

		for i := range r.Endpoints {
			ep := &r.Endpoints[i]
			ep.jobs = make(chan *publishEnvelope, 256)
			ep.outExchange = outEx
			ep.rabbitmqChannel = r.RabbitmqChannel
			r.startPublishLoop(wg, ep, ctx)
		}

		r.sendResultConsumers = map[int]bool{}
		r.outgoingStarted = true
	}

	if r.sendResultConsumers == nil {
		r.sendResultConsumers = map[int]bool{}
	}
	if r.sendResultConsumers[shardID] {
		return e.Nil()
	}

	effectivePodID := podID
	if effectivePodID == "" {
		effectivePodID = "unknown"
	}
	qName := sendResultQueueName(effectivePodID, shardID)
	q, uerr := r.RabbitmqChannel.QueueDeclare(
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
	uerr = r.RabbitmqChannel.QueueBind(
		q.Name,
		fmt.Sprintf("%s.q%02d", effectivePodID, shardID),
		r.sendResultExchange,
		false,
		amqp.Table{},
	)
	if uerr != nil {
		return e.FromError(uerr, "bind send result queue").WithSeverity(e.Critical)
	}
	if err := r.startSendResultConsumer(wg, qName, ctx); !err.IsNil() {
		return err
	}
	r.sendResultConsumers[shardID] = true
	return e.Nil()
}

func (r *Router) startPublishLoop(wg *sync.WaitGroup, ep *Endpoint, ctx context.Context) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for job := range ep.jobs {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if job == nil {
				continue
			}
			pubCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			pid := r.PodID
			if pid == "" {
				pid = "unknown"
			}
			// Must match QueueBind in StartOutgoing (result always to shard 0 key).
			resultRoutingKey := fmt.Sprintf("%s.q%02d", pid, 0)
			log.Printf("trace=%s handlers.publish outgoing_exchange=%s outgoing_rk=%s result_rk=%s", job.correlationID, ep.outExchange, job.routingKey, resultRoutingKey)
			publishErr := ep.rabbitmqChannel.PublishWithContext(
				pubCtx,
				ep.outExchange,
				job.routingKey,
				false,
				false,
				amqp.Publishing{
					ContentType:   "application/json",
					CorrelationId: job.correlationID,
					Body:          job.body,
					Headers: amqp.Table{
						"correlation_id":     job.correlationID,
						"result_routing_key": resultRoutingKey,
					},
				},
			)
			cancel()
			if publishErr != nil && r.ErrorChannel != nil {
				r.ErrorChannel <- e.FromError(publishErr, "publish tele.Message").WithSeverity(e.Critical).PushStack()
			}
			if publishErr != nil {
				log.Printf("trace=%s handlers.publish_error exchange=%s rk=%s err=%v", job.correlationID, ep.outExchange, job.routingKey, publishErr)
			} else {
				log.Printf("trace=%s handlers.publish_ok outgoing_exchange=%s outgoing_rk=%s", job.correlationID, ep.outExchange, job.routingKey)
			}
		}
	}()
}

func (r *Router) startSendResultConsumer(wg *sync.WaitGroup, queueName string, ctx context.Context) *e.ErrorInfo {
	consumer, uerr := r.RabbitmqChannel.Consume(
		queueName,
		fmt.Sprintf("handlers-send-result-%s", queueName),
		false,
		false,
		false,
		false,
		amqp.Table{},
	)
	if uerr != nil {
		return e.FromError(uerr, "consume send results").WithSeverity(e.Critical)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for delivery := range consumer {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var sr SendResult
			if uerr := json.Unmarshal(delivery.Body, &sr); uerr != nil {
				log.Printf("trace=%s handlers.result_unmarshal_error queue=%s rk=%s err=%v", delivery.CorrelationId, queueName, delivery.RoutingKey, uerr)
				if r.ErrorChannel != nil {
					r.ErrorChannel <- e.FromError(uerr, "unmarshal SendResult").WithSeverity(e.Critical).PushStack()
				}
				// Poison message: retry won't change payload, so avoid infinite requeue loop.
				_ = delivery.Nack(false, false)
				continue
			}
			corr := sr.CorrelationID
			if corr == "" {
				corr = delivery.CorrelationId
			}
			log.Printf("trace=%s handlers.result_received queue=%s rk=%s success=%t", corr, queueName, delivery.RoutingKey, sr.IsSuccess)
			if corr == "" {
				log.Printf("trace=missing handlers.result_missing_correlation queue=%s rk=%s", queueName, delivery.RoutingKey)
				_ = delivery.Ack(false)
				continue
			}
			if v, ok := r.sendWaiters.LoadAndDelete(corr); ok {
				if ch, ok := v.(chan *SendResult); ok {
					log.Printf("trace=%s handlers.result_waiter_found queue=%s", corr, queueName)
					select {
					case ch <- &sr:
						log.Printf("trace=%s handlers.result_delivered_to_waiter queue=%s", corr, queueName)
					default:
						log.Printf("trace=%s handlers.result_waiter_channel_full queue=%s", corr, queueName)
					}
				}
			} else {
				log.Printf("trace=%s handlers.result_waiter_missing queue=%s", corr, queueName)
			}
			_ = delivery.Ack(false)
		}
	}()

	return e.Nil()
}
