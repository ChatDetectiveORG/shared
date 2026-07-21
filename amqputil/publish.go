package amqputil

import (
	"context"
	"log"
	"strings"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// IsRecoverableAMQPError reports client/broker errors where reopening the channel may help.
func IsRecoverableAMQPError(err error) bool {
	if err == nil {
		return false
	}
	if err == amqp.ErrClosed {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "channel/connection is not open") ||
		strings.Contains(msg, "Exception (504)") ||
		strings.Contains(msg, "Exception (320)")
}

// PublishChannel holds one AMQP channel for publishing and transparently reopens it on failure.
type PublishChannel struct {
	mu   sync.Mutex
	ch   *amqp.Channel
	open func() (*amqp.Channel, error)
}

// NewPublishChannel creates a publisher. open must return a fresh channel when the current one is dead.
func NewPublishChannel(initial *amqp.Channel, open func() (*amqp.Channel, error)) *PublishChannel {
	return &PublishChannel{ch: initial, open: open}
}

func (p *PublishChannel) invalidate() {
	if p.ch != nil {
		_ = p.ch.Close()
		p.ch = nil
	}
}

// Close closes the underlying AMQP channel.
func (p *PublishChannel) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.invalidate()
}

func (p *PublishChannel) ensure() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch != nil {
		return nil
	}
	if p.open == nil {
		return amqp.ErrClosed
	}
	ch, err := p.open()
	if err != nil {
		return err
	}
	p.ch = ch
	return nil
}

// Publish publishes with one reconnect attempt on recoverable AMQP errors.
func (p *PublishChannel) Publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	pubErr := p.publishOnce(ctx, exchange, routingKey, msg)
	if pubErr == nil {
		return nil
	}
	if !IsRecoverableAMQPError(pubErr) {
		return pubErr
	}

	log.Printf("amqputil: publish failed (%v); reopening channel", pubErr)
	p.mu.Lock()
	p.invalidate()
	p.mu.Unlock()

	if ensureErr := p.ensure(); ensureErr != nil {
		return pubErr
	}
	return p.publishOnce(ctx, exchange, routingKey, msg)
}

func (p *PublishChannel) publishOnce(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	p.mu.Lock()
	ch := p.ch
	p.mu.Unlock()

	if ch == nil {
		return amqp.ErrClosed
	}
	return ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
}
