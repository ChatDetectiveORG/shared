package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultOutgoingExchange   = "chatdetective.output.send"
	defaultSendResultExchange = "chatdetective.send.result"
)

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

	qName := fmt.Sprintf("chatdetective.send.result.q%02d", shardID)
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
		fmt.Sprintf("%s.q%02d", podID, shardID),
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
						"correlation_id": job.correlationID,
					},
				},
			)
			cancel()
			if publishErr != nil && r.ErrorChannel != nil {
				r.ErrorChannel <- e.FromError(publishErr, "publish tele.Message").WithSeverity(e.Critical).PushStack()
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
				if r.ErrorChannel != nil {
					r.ErrorChannel <- e.FromError(uerr, "unmarshal SendResult").WithSeverity(e.Critical).PushStack()
				}
				_ = delivery.Nack(false, true)
				continue
			}
			corr := sr.CorrelationID
			if corr == "" {
				corr = delivery.CorrelationId
			}
			if corr == "" {
				_ = delivery.Ack(false)
				continue
			}
			if v, ok := r.sendWaiters.LoadAndDelete(corr); ok {
				if ch, ok := v.(chan *SendResult); ok {
					select {
					case ch <- &sr:
					default:
					}
				}
			}
			_ = delivery.Ack(false)
		}
	}()

	return e.Nil()
}
