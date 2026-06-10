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

// publishEnvelope — задание на публикацию: тело = JSON telegram.OutgoingRequest.
type publishEnvelope struct {
	routingKey    string
	body          []byte
	correlationID string
}

func publishOutgoingJob(ch *amqp.Channel, outExchange, podID string, errorChannel chan *e.ErrorInfo, job *publishEnvelope) {
	pubCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	effectivePodID := podID
	if effectivePodID == "" {
		effectivePodID = "unknown"
	}
	// Must match QueueBind in EnsureSendResultConsumer (result always to shard 0 key).
	resultRoutingKey := fmt.Sprintf("%s.q%02d", effectivePodID, 0)
	log.Printf("trace=%s handlers.publish outgoing_exchange=%s outgoing_rk=%s result_rk=%s", job.correlationID, outExchange, job.routingKey, resultRoutingKey)
	publishErr := ch.PublishWithContext(
		pubCtx,
		outExchange,
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
	if publishErr != nil && errorChannel != nil {
		errorChannel <- e.FromError(publishErr, "publish tele.Message").WithSeverity(e.Critical).PushStack()
	}
	if publishErr != nil {
		log.Printf("trace=%s handlers.publish_error exchange=%s rk=%s err=%v", job.correlationID, outExchange, job.routingKey, publishErr)
	} else {
		log.Printf("trace=%s handlers.publish_ok outgoing_exchange=%s outgoing_rk=%s", job.correlationID, outExchange, job.routingKey)
	}
}

func startSendResultConsumer(wg *sync.WaitGroup, ch *amqp.Channel, queueName string, waiters *sync.Map, errorChannel chan *e.ErrorInfo, ctx context.Context) *e.ErrorInfo {
	consumer, uerr := ch.Consume(
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
				if errorChannel != nil {
					errorChannel <- e.FromError(uerr, "unmarshal SendResult").WithSeverity(e.Critical).PushStack()
				}
				_ = delivery.Nack(false, false)
				continue
			}
			corr := sr.CorrelationID
			if corr == "" {
				corr = delivery.CorrelationId
			}
			if e.IsNonNil(sr.Error) {
				log.Printf("trace=%s handlers.result_received queue=%s rk=%s success=%t err=%v", corr, queueName, delivery.RoutingKey, sr.IsSuccess, sr.Error)
			}
			log.Printf("trace=%s handlers.result_received queue=%s rk=%s success=%t", corr, queueName, delivery.RoutingKey, sr.IsSuccess)
			if corr == "" {
				log.Printf("trace=missing handlers.result_missing_correlation queue=%s rk=%s", queueName, delivery.RoutingKey)
				_ = delivery.Ack(false)
				continue
			}
			if v, ok := waiters.LoadAndDelete(corr); ok {
				if replyCh, ok := v.(chan *SendResult); ok {
					log.Printf("trace=%s handlers.result_waiter_found queue=%s", corr, queueName)
					select {
					case replyCh <- &sr:
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

// StartOutgoing поднимает shared OutgoingPublisher и consumer SendResult для shardID.
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

	if r.publisher == nil {
		effectivePodID := podID
		if effectivePodID == "" {
			effectivePodID = r.PodID
		}
		pub, err := NewOutgoingPublisher(OutgoingConfig{
			Channel:            r.RabbitmqChannel,
			PodID:              effectivePodID,
			OutgoingExchange:   r.OutgoingExchange,
			SendResultExchange: r.SendResultExchange,
			ErrorChannel:       r.ErrorChannel,
		})
		if e.IsNonNil(err) {
			return err
		}
		r.publisher = pub
	}

	if err := r.publisher.ensurePublishLoop(wg, ctx); e.IsNonNil(err) {
		return err
	}

	for i := range r.Endpoints {
		r.Endpoints[i].jobs = r.publisher.jobs
	}

	return r.publisher.EnsureSendResultConsumer(wg, shardID, ctx)
}
