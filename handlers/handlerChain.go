package handlers

import (
	"context"
	e "github.com/ChatDetectiveORG/shared/errors"
	"time"

	tele "gopkg.in/telebot.v4"
)

type chainHandlerFunc func(u tele.Update, hashe *handlerChainHashe) *e.ErrorInfo
type executionType string
const (
	endOnError executionType = "endOnError"
	skipOnError executionType = "skipOnError"
)

type chainHandler struct {
	function chainHandlerFunc
	middlewares []chainHandler
	exectype executionType
}

func InitChainHandler(function chainHandlerFunc, exectype executionType, middlewares ...chainHandler) chainHandler {
	return chainHandler{
		function: function,
		exectype: exectype,
		middlewares: middlewares,
	}
}

func (ch chainHandler) Exec(u tele.Update, hashe *handlerChainHashe) *e.ErrorInfo {
	if ch.middlewares != nil {
		for _, middleware := range ch.middlewares {
			errInfo := middleware.Exec(u, hashe)

			if !errInfo.IsNil() && middleware.exectype == endOnError {
				return errInfo.PushStack()
			}
		}
	}

	errInfo := ch.function(u, hashe)
	if !errInfo.IsNil() && ch.exectype == endOnError {
		return errInfo
	}

	return e.Nil()
}

type HandlerChain struct {
	Handlers      []chainHandler
	Hashe         *handlerChainHashe
	ErrorInfo     *e.ErrorInfo
	timeout       time.Duration
}

func (hc HandlerChain) Init(timeout time.Duration, handlers ...chainHandler) *HandlerChain {
	new := HandlerChain{
		Handlers: handlers,
		timeout:  timeout,
		Hashe:    handlerChainHashe{}.Init(),
	}

	return &new
}

func (hc *HandlerChain) Run(u tele.Update) ([]handlerResponse, *e.ErrorInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	done := make(chan *e.ErrorInfo, 1)

	go func() {
		for _, handler := range hc.Handlers {
			select {
			case <-ctx.Done():
				hc.ErrorInfo = e.FromError(ctx.Err(), "Context cancelled")
				done <- hc.ErrorInfo
				return
			default:
			}

			errInfo := handler.Exec(u, hc.Hashe)
			if !errInfo.IsNil() {
				hc.ErrorInfo = errInfo
				done <- hc.ErrorInfo
				return
			}
		}

		done <- e.Nil()
	}()

	select {
	case err := <-done:
		if err.IsNil() || err.Severity == e.Ingnored {
			return hc.Hashe.responses, nil
		}
		return hc.Hashe.responses, err.PushStack()
	case <-ctx.Done():
		hc.ErrorInfo = e.FromError(ctx.Err(), "Context timeout")
		return hc.Hashe.responses, hc.ErrorInfo
	}
}