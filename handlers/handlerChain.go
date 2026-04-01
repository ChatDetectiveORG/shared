package handlers

import (
	"context"
	"sync"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/google/uuid"

	tele "gopkg.in/telebot.v4"
)

type chainHandlerFunc func(u tele.Update, hashe *HandlerChainHashe) *e.ErrorInfo
type executionType string

const (
	EndOnError  executionType = "endOnError"
	SkipOnError executionType = "skipOnError"
)

type chainHandler struct {
	function    chainHandlerFunc
	middlewares []chainHandler
	exectype    executionType
}

func InitChainHandler(function chainHandlerFunc, exectype executionType, middlewares ...chainHandler) chainHandler {
	return chainHandler{
		function:    function,
		exectype:    exectype,
		middlewares: middlewares,
	}
}

func (ch chainHandler) Exec(u tele.Update, hashe *HandlerChainHashe) *e.ErrorInfo {
	if ch.middlewares != nil {
		for _, middleware := range ch.middlewares {
			errInfo := middleware.Exec(u, hashe)
			if !errInfo.IsNil() && middleware.exectype == EndOnError {
				return errInfo.PushStack()
			}
		}
	}

	errInfo := ch.function(u, hashe)
	if !errInfo.IsNil() && ch.exectype == EndOnError {
		return errInfo
	}

	return e.Nil()
}

type HandlerChain struct {
	Handlers []chainHandler
	Hashe    *HandlerChainHashe
	timeout  time.Duration
	ChainID  string

	wg *sync.WaitGroup
}

func (hc HandlerChain) Init(timeout time.Duration, handlers ...chainHandler) *HandlerChain {
	n := HandlerChain{
		Handlers: handlers,
		timeout:  timeout,
		Hashe:    nil,
		ChainID:  uuid.New().String(),
	}
	return &n
}

func (hc *HandlerChain) WithWaitGroup(wg *sync.WaitGroup) *HandlerChain {
	hc.wg = wg
	return hc
}

func (hc *HandlerChain) Run(u tele.Update, jobs chan *publishEnvelope, waiters *sync.Map) *e.ErrorInfo {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	runID := uuid.New().String()
	hc.Hashe = HandlerChainHashe{}.Init(jobs, waiters, runID)

	done := make(chan *e.ErrorInfo, 1)

	runHandlers := func() {
		for _, handler := range hc.Handlers {
			select {
			case <-ctx.Done():
				done <- e.FromError(ctx.Err(), "context cancelled")
				return
			default:
			}

			err := handler.Exec(u, hc.Hashe)
			if !err.IsNil() {
				done <- err
				return
			}
		}
		done <- e.Nil()
	}

	hc.wg.Add(1)
	go func() {
		defer hc.wg.Done()
		runHandlers()
	}()

	select {
	case err := <-done:
		if err.IsNil() || err.Severity == e.Ingnored {
			return e.Nil()
		}
		return err.PushStack()
	case <-ctx.Done():
		return e.FromError(ctx.Err(), "handler chain timeout")
	}
}
