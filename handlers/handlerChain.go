package handlers

import (
	"context"
	"sync"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"

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
	Handlers  []chainHandler
	Hashe     *HandlerChainHashe
	timeout   time.Duration

	wg *sync.WaitGroup
}

func (hc HandlerChain) Init(timeout time.Duration, handlers ...chainHandler) *HandlerChain {	
	new := HandlerChain{
		Handlers: handlers,
		timeout:  timeout,
		Hashe:    HandlerChainHashe{}.Init(),
	}

	return &new
}

func (hc *HandlerChain) WithWaitGroup(wg *sync.WaitGroup) *HandlerChain {
	hc.wg = wg
	return hc
}

func (hc *HandlerChain) Run(u tele.Update) *e.ErrorInfo {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	done := make(chan *e.ErrorInfo, 1)

	hc.wg.Add(1)
	go func() {
		defer hc.wg.Done()

		for _, handler := range hc.Handlers {
			select {
			case <-ctx.Done():
				err := e.FromError(ctx.Err(), "Context cancelled")
				done <- err
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
	}()

	select {
	case err := <-done:
		if err.IsNil() || err.Severity == e.Ingnored {
			return nil
		}
		return err.PushStack()
	case <-ctx.Done():
		return e.FromError(ctx.Err(), "Context timeout")
	}
}
