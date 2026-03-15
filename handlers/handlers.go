package handlers

import (
	"context"
	"encoding/json"
	e "github.com/ChatDetectiveORG/shared/errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	tele "gopkg.in/telebot.v4"
)

type Endpoint struct {
	handlerChain HandlerChain
	filter  filter
	Name    string
}

func (e *Endpoint) Init(name string, handlerChain HandlerChain, filter filter) {
	e.Name = name
	e.handlerChain = handlerChain
	e.filter = filter
}

func (ep *Endpoint) ExecuteIfFilterPasses(update tele.Update, rabbitmqChannel *amqp.Channel) *e.ErrorInfo {
	if !ep.filter.Filter(update) {
		return nil
	}

	if rabbitmqChannel == nil {
		return e.NewError("rabbitmq channel is nil", "rabbitmq channel is not initialized").WithSeverity(e.Critical)
	}

	resp, err := ep.handlerChain.Run(update)
	if !err.IsNil() {
		return err
	}

	jsonData, unwrappedError := json.Marshal(resp)
	if unwrappedError != nil {
		return e.FromError(unwrappedError, "failed to marshal send data").WithSeverity(e.Critical)
	}

	publishContext, publishContextCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer publishContextCancel()

	var sendErrors []*e.ErrorInfo

	for _, r := range resp {
		unwrappedError = rabbitmqChannel.PublishWithContext(
			publishContext,
			"chatdetective.output.send",
			r.SenderBot,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        jsonData,
			},
		)
		if unwrappedError != nil {
			sendErrors = append(sendErrors, e.FromError(unwrappedError, "failed to publish send data").WithSeverity(e.Critical))
		}	
	}

	if len(sendErrors) != 0 {
		globalError := e.NewError("Error While publishing handler chain responses", "")
		globalError.WithData(map[string]any{"errors": sendErrors})

		return globalError.PushStack()
	}

	return e.Nil()
}

type Router struct {
	Endpoints []Endpoint
	ErrorChannel chan *e.ErrorInfo
	RabbitmqChannel *amqp.Channel
}

func (r *Router) Dispatch(update tele.Update) *e.ErrorInfo {
	for _, endpoint := range r.Endpoints {
		err := endpoint.ExecuteIfFilterPasses(update, r.RabbitmqChannel)
		if !err.IsNil() {
			r.ErrorChannel <- err.PushStack().WithData(map[string]any{"endpoint name": endpoint.Name})
		}
	}

	return nil
}
