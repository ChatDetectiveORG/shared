package amqputil

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRunConsumerLoopReconnects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deliveries := make(chan amqp.Delivery, 1)
	dialCount := 0

	go RunConsumerLoop(ctx, ConsumerConfig{
		Dial: func() (*ConsumerSession, error) {
			dialCount++
			if dialCount == 1 {
				deliveries <- amqp.Delivery{CorrelationId: "first"}
				close(deliveries)
				return &ConsumerSession{
					Deliveries: deliveries,
					Cleanup:    func() {},
				}, nil
			}
			cancel()
			return &ConsumerSession{
				Deliveries: make(chan amqp.Delivery),
				Cleanup:    func() {},
			}, nil
		},
		OnDelivery: func(delivery amqp.Delivery) {
			if delivery.CorrelationId != "first" {
				t.Fatalf("unexpected delivery: %#v", delivery)
			}
		},
	})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if dialCount >= 2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected at least 2 dial attempts, got %d", dialCount)
}
