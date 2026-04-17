package handlers

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/ChatDetectiveORG/shared/telegram"
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

func (hch *HandlerChainHashe) marshalOutgoing(request *telegram.OutgoingRequest) ([]byte, *e.ErrorInfo) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, e.FromError(err, "marshal outgoing request").WithSeverity(e.Critical)
	}
	return body, e.Nil()
}

func (hch *HandlerChainHashe) enqueue(routingKey string, body []byte, correlationID string, action string) *e.ErrorInfo {
	if hch.jobs == nil {
		return e.NewError("outgoing not configured", "call Router.StartOutgoing before Emit").
			WithSeverity(e.Warning)
	}
	log.Printf("trace=%s %s rk=%s", correlationID, action, routingKey)
	select {
	case hch.jobs <- &publishEnvelope{
		routingKey:    routingKey,
		body:          body,
		correlationID: correlationID,
	}:
		return e.Nil()
	default:
		return e.NewError("outgoing queue is full", action).WithSeverity(e.Critical)
	}
}

func (hch *HandlerChainHashe) waitResult(ctx context.Context, routingKey string, body []byte, action string) (*SendResult, *e.ErrorInfo) {
	if hch.jobs == nil || hch.waiters == nil {
		return nil, e.NewError("outgoing not configured", "call Router.StartOutgoing before "+action).
			WithSeverity(e.Warning)
	}

	corr := uuid.New().String()
	replyCh := make(chan *SendResult, 1)
	hch.waiters.Store(corr, replyCh)
	log.Printf("trace=%s %s_store_waiter rk=%s run_id=%s", corr, action, routingKey, hch.runID)
	defer func() {
		hch.waiters.Delete(corr)
		log.Printf("trace=%s %s_cleanup_waiter run_id=%s", corr, action, hch.runID)
	}()

	if err := hch.enqueue(routingKey, body, corr, action); e.IsNonNil(err) {
		return nil, err
	}
	log.Printf("trace=%s %s_enqueued rk=%s", corr, action, routingKey)

	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	select {
	case sr := <-replyCh:
		log.Printf("trace=%s %s_got_result success=%t", corr, action, sr != nil && sr.IsSuccess)
		if sr == nil {
			return nil, e.NewError("empty send result", action).WithSeverity(e.Warning)
		}
		if !sr.IsSuccess {
			if sr.Error != nil && !sr.Error.IsNil() {
				return sr, sr.Error.PushStack()
			}
			return sr, e.NewError("send failed", action).WithSeverity(e.Notice)
		}
		return sr, e.Nil()
	case <-waitCtx.Done():
		log.Printf("trace=%s %s_timeout", corr, action)
		return nil, e.FromError(waitCtx.Err(), "wait send result").WithSeverity(e.Warning)
	}
}

// Emit публикует в OutgoingExchange JSON envelope c tele.Message.
func (hch *HandlerChainHashe) Emit(routingKey string, msg *tele.Message) *e.ErrorInfo {
	if msg == nil {
		return e.NewError("message is nil", "Emit").WithSeverity(e.Warning)
	}
	body, err := hch.marshalOutgoing(telegram.NewOutgoingMessageRequest(msg))
	if e.IsNonNil(err) {
		return err
	}
	return hch.enqueue(routingKey, body, uuid.New().String(), "handlers.emit")
}

// EmitAlbum публикует в OutgoingExchange JSON envelope c album payload.
func (hch *HandlerChainHashe) EmitAlbum(routingKey string, album *telegram.MediaGroup) *e.ErrorInfo {
	if album == nil {
		return e.NewError("album is nil", "EmitAlbum").WithSeverity(e.Warning)
	}
	body, err := hch.marshalOutgoing(telegram.NewOutgoingAlbumRequest(album))
	if e.IsNonNil(err) {
		return err
	}
	return hch.enqueue(routingKey, body, uuid.New().String(), "handlers.emit_album")
}

// EmitWait ждёт SendResult от message-sender с тем же correlation_id.
func (hch *HandlerChainHashe) EmitWait(ctx context.Context, routingKey string, msg *tele.Message) (*tele.Message, *e.ErrorInfo) {
	if msg == nil {
		return nil, e.NewError("message is nil", "EmitWait").WithSeverity(e.Warning)
	}
	body, err := hch.marshalOutgoing(telegram.NewOutgoingMessageRequest(msg))
	if e.IsNonNil(err) {
		return nil, err
	}
	sr, waitErr := hch.waitResult(ctx, routingKey, body, "handlers.emit_wait")
	if e.IsNonNil(waitErr) {
		if sr != nil {
			return sr.SentMessage, waitErr
		}
		return nil, waitErr
	}
	return sr.SentMessage, e.Nil()
}

// EmitAlbumWait ждёт SendResult от message-sender для отправленного альбома.
func (hch *HandlerChainHashe) EmitAlbumWait(ctx context.Context, routingKey string, album *telegram.MediaGroup) ([]*tele.Message, *e.ErrorInfo) {
	if album == nil {
		return nil, e.NewError("album is nil", "EmitAlbumWait").WithSeverity(e.Warning)
	}
	body, err := hch.marshalOutgoing(telegram.NewOutgoingAlbumRequest(album))
	if e.IsNonNil(err) {
		return nil, err
	}
	sr, waitErr := hch.waitResult(ctx, routingKey, body, "handlers.emit_album_wait")
	if e.IsNonNil(waitErr) {
		if sr != nil && len(sr.SentAlbum) > 0 {
			return sr.SentAlbum, waitErr
		}
		if sr != nil && sr.SentMessage != nil {
			return []*tele.Message{sr.SentMessage}, waitErr
		}
		return nil, waitErr
	}
	if len(sr.SentAlbum) > 0 {
		return sr.SentAlbum, e.Nil()
	}
	if sr.SentMessage != nil {
		return []*tele.Message{sr.SentMessage}, e.Nil()
	}
	return nil, e.NewError("empty sent album", "EmitAlbumWait").WithSeverity(e.Warning)
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
