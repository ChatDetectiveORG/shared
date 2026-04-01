package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/google/uuid"

	tele "gopkg.in/telebot.v4"
)

// HandlerChainHashe — контекст одного прогона цепочки: отправка в exchange и произвольные аргументы между шагами.
type HandlerChainHashe struct {
	args    map[string]any
	jobs    chan *publishEnvelope
	waiters *sync.Map
	runID   string
}

func (hch HandlerChainHashe) Init(jobs chan *publishEnvelope, waiters *sync.Map, runID string) *HandlerChainHashe {
	return &HandlerChainHashe{
		args:    make(map[string]any),
		jobs:    jobs,
		waiters: waiters,
		runID:   runID,
	}
}

// RunID идентификатор конкретного запуска цепочки (логи / трассировка).
func (hch *HandlerChainHashe) RunID() string {
	return hch.runID
}

// Emit публикует в OutgoingExchange тело = JSON tele.Message. Без Router.StartOutgoing — ошибка.
func (hch *HandlerChainHashe) Emit(routingKey string, msg *tele.Message) *e.ErrorInfo {
	if hch.jobs == nil {
		return e.NewError("outgoing not configured", "call Router.StartOutgoing before Emit").
			WithSeverity(e.Warning)
	}
	if msg == nil {
		return e.NewError("message is nil", "Emit").WithSeverity(e.Warning)
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return e.FromError(err, "marshal tele.Message").WithSeverity(e.Critical)
	}
	corr := uuid.New().String()
	select {
	case hch.jobs <- &publishEnvelope{
		routingKey:    routingKey,
		body:          body,
		correlationID: corr,
	}:
		return e.Nil()
	default:
		return e.NewError("outgoing queue is full", "Emit").WithSeverity(e.Critical)
	}
}

// EmitWait ждёт SendResult от message-sender с тем же correlation_id (в теле или в CorrelationId delivery).
func (hch *HandlerChainHashe) EmitWait(ctx context.Context, routingKey string, msg *tele.Message) (*tele.Message, *e.ErrorInfo) {
	if hch.jobs == nil || hch.waiters == nil {
		return nil, e.NewError("outgoing not configured", "call Router.StartOutgoing before EmitWait").
			WithSeverity(e.Warning)
	}
	if msg == nil {
		return nil, e.NewError("message is nil", "EmitWait").WithSeverity(e.Warning)
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, e.FromError(err, "marshal tele.Message").WithSeverity(e.Critical)
	}

	corr := uuid.New().String()
	replyCh := make(chan *SendResult, 1)
	hch.waiters.Store(corr, replyCh)
	defer func() {
		hch.waiters.Delete(corr)
	}()

	select {
	case hch.jobs <- &publishEnvelope{
		routingKey:    routingKey,
		body:          body,
		correlationID: corr,
	}:
	case <-ctx.Done():
		return nil, e.FromError(ctx.Err(), "enqueue publish").WithSeverity(e.Warning)
	default:
		return nil, e.NewError("outgoing queue is full", "EmitWait").WithSeverity(e.Critical)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	select {
	case sr := <-replyCh:
		if sr == nil {
			return nil, e.NewError("empty send result", "EmitWait").WithSeverity(e.Warning)
		}
		if !sr.IsSuccess {
			if sr.Error != nil && !sr.Error.IsNil() {
				return sr.SentMessage, sr.Error.PushStack()
			}
			return sr.SentMessage, e.NewError("send failed", "EmitWait").WithSeverity(e.Notice)
		}
		return sr.SentMessage, e.Nil()
	case <-waitCtx.Done():
		return nil, e.FromError(waitCtx.Err(), "wait send result").WithSeverity(e.Warning)
	}
}

func (hch *HandlerChainHashe) Add(name string, value interface{}) *HandlerChainHashe {
	hch.args[name] = value
	return hch
}

func (hch *HandlerChainHashe) Get(name string) (interface{}, bool) {
	v, exists := hch.args[name]
	if !exists {
		return nil, false
	}
	return v, true
}

func (hch *HandlerChainHashe) Trunc() *HandlerChainHashe {
	hch.args = make(map[string]any)
	return hch
}
