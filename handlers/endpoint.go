package handlers

import (
	"sync"

	e "github.com/ChatDetectiveORG/shared/errors"
	tele "gopkg.in/telebot.v4"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Endpoint — фильтр + цепочка хендлеров на один тип сценария.
type Endpoint struct {
	Name         string
	HandlerChain HandlerChain
	Filter       UpdateFilter

	jobs              chan *publishEnvelope
	rabbitmqChannel   *amqp.Channel
	outExchange       string
}

// Init задаёт имя, цепочку и фильтр. Исходящий AMQP подключается через Router.StartOutgoing.
func (ep *Endpoint) Init(name string, chain HandlerChain, f UpdateFilter) *Endpoint {
	ep.Name = name
	ep.HandlerChain = chain
	ep.Filter = f
	return ep
}

func (ep *Endpoint) runChain(update tele.Update, router *Router, wg *sync.WaitGroup) *e.ErrorInfo {
	if ep.Filter != nil && !ep.Filter.Filter(update) {
		return e.Nil()
	}
	return ep.HandlerChain.WithWaitGroup(wg).Run(update, ep.jobs, router.sendWaiters)
}
