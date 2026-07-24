package handlers

import (
	"sync"

	e "github.com/ChatDetectiveORG/shared/errors"
	tele "gopkg.in/telebot.v4"
)

// Endpoint — фильтр + цепочка хендлеров на один тип сценария.
type Endpoint struct {
	Name         string
	HandlerChain HandlerChain
	Filter       UpdateFilter

	jobs chan *PublishEnvelope
}

// Init задаёт имя, цепочку и фильтр. Исходящий AMQP подключается через Router.StartOutgoing или OutgoingPublisher.
func (ep *Endpoint) Init(name string, chain HandlerChain, f UpdateFilter) *Endpoint {
	ep.Name = name
	ep.HandlerChain = chain
	ep.Filter = f
	return ep
}

func (ep *Endpoint) runChain(update tele.Update, router *Router, wg *sync.WaitGroup, mirrorID string) *e.ErrorInfo {
	if ep.Filter != nil && !ep.Filter.Filter(update) {
		return e.Nil()
	}

	jobs := ep.jobs
	var waiters *sync.Map
	if router != nil && router.publisher != nil {
		if jobs == nil {
			jobs = router.publisher.jobs
		}
		waiters = router.publisher.waiters
	}

	return ep.HandlerChain.WithWaitGroup(wg).Run(update, jobs, waiters, mirrorID)
}

// RunForTest synchronously executes the endpoint filter and handler chain.
// Use with injected jobs/waiters channels from chain tests.
func (ep *Endpoint) RunForTest(update tele.Update, jobs chan *PublishEnvelope, waiters *sync.Map, mirrorID string, wg *sync.WaitGroup) *e.ErrorInfo {
	if ep.Filter != nil && !ep.Filter.Filter(update) {
		return e.Nil()
	}
	if wg == nil {
		var local sync.WaitGroup
		return ep.HandlerChain.WithWaitGroup(&local).Run(update, jobs, waiters, mirrorID)
	}
	return ep.HandlerChain.WithWaitGroup(wg).Run(update, jobs, waiters, mirrorID)
}
