package amqputil

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultConsumerReconnectDelay = 2 * time.Second
	maxConsumerReconnectDelay     = 30 * time.Second
)

// ConsumerSession is one AMQP consume session (merged or single queue).
type ConsumerSession struct {
	Deliveries <-chan amqp.Delivery
	Channel    *amqp.Channel
	Cleanup    func()
}

// ConsumerConfig configures a resilient delivery loop with reconnect.
type ConsumerConfig struct {
	Dial       func() (*ConsumerSession, error)
	OnConnect  func(*ConsumerSession)
	OnDelivery func(amqp.Delivery)
}

// RunConsumerLoop dials, consumes, and reconnects until ctx is cancelled.
func RunConsumerLoop(ctx context.Context, cfg ConsumerConfig) {
	delay := defaultConsumerReconnectDelay

	for {
		if ctx.Err() != nil {
			return
		}

		session, err := cfg.Dial()
		if err != nil {
			log.Printf("amqputil: consumer connect failed: %v; retry in %s", err, delay)
			if waitFor(ctx, delay) {
				return
			}
			delay = nextConsumerDelay(delay)
			continue
		}

		if cfg.OnConnect != nil {
			cfg.OnConnect(session)
		}

		disconnected := consumeDeliveries(ctx, session.Deliveries, cfg.OnDelivery)
		if session.Cleanup != nil {
			session.Cleanup()
		}

		if ctx.Err() != nil {
			return
		}
		if !disconnected {
			return
		}

		log.Printf("amqputil: consumer disconnected; reconnecting in %s", delay)
		if waitFor(ctx, delay) {
			return
		}
		delay = nextConsumerDelay(delay)
	}
}

func consumeDeliveries(ctx context.Context, deliveries <-chan amqp.Delivery, onDelivery func(amqp.Delivery)) bool {
	if onDelivery == nil {
		onDelivery = func(amqp.Delivery) {}
	}

	for {
		select {
		case <-ctx.Done():
			return false
		case delivery, ok := <-deliveries:
			if !ok {
				return true
			}
			onDelivery(delivery)
		}
	}
}

func waitFor(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return true
	case <-timer.C:
		return false
	}
}

func nextConsumerDelay(delay time.Duration) time.Duration {
	delay *= 2
	if delay > maxConsumerReconnectDelay {
		return maxConsumerReconnectDelay
	}
	return delay
}
