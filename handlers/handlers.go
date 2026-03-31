package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"

	amqp "github.com/rabbitmq/amqp091-go"
	tele "gopkg.in/telebot.v4"
)

type Endpoint struct {
	handlerChain HandlerChain
	filter  filter
	Name    string
	wg *sync.WaitGroup
	rabbitmqChannel *amqp.Channel

	outcomingExchangeName string
	incomingQueueName string
}

func (e *Endpoint) Init(name string, handlerChain HandlerChain, filter filter, wg *sync.WaitGroup) {
	e.Name = name
	e.handlerChain = handlerChain
	e.filter = filter
	e.wg = wg

	e.outcomingExchangeName = "chatdetective.output.send"
	e.incomingQueueName = "chatdetective.sendresult.queue"
}

func (ep *Endpoint) WithOutcomingExchangeName(outcomingExchangeName string) *Endpoint {
	ep.outcomingExchangeName = outcomingExchangeName
	return ep
}

func (ep *Endpoint) WithIncomingQueueName(incomingQueueName string) *Endpoint {
	ep.incomingQueueName = incomingQueueName
	return ep
}

func (ep *Endpoint) WithRabbitMQChannel(rabbitmqChannel *amqp.Channel) *Endpoint {
	ep.rabbitmqChannel = rabbitmqChannel
	return ep
}

func (ep *Endpoint) send(messages chan *HandlerResponse, ctx context.Context) chan *e.ErrorInfo {
	errors := make(chan *e.ErrorInfo, 10)

	ep.wg.Add(1)
	go func () {
		defer ep.wg.Done()

		for {
			select {
			case message, ok := <-messages:
				if !ok {
					return // channel closed
				}
	
				jsonData, unwrappedError := json.Marshal(message)
				if unwrappedError != nil {
					errors <- e.FromError(unwrappedError, "failed to marshal send data").WithSeverity(e.Critical)
					continue
				}
	
				publishContext, publishContextCancel := context.WithTimeout(context.Background(), 10*time.Second)
				
				unwrappedError = ep.rabbitmqChannel.PublishWithContext(
					publishContext,
					ep.outcomingExchangeName,
					message.SenderBot,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        jsonData,
					},
				)
				publishContextCancel()
	
				if unwrappedError != nil {
					errors <- e.FromError(unwrappedError, "failed to publish send data").WithSeverity(e.Critical)
				}

			case <-ctx.Done():
				return // context cancelled
			}
		}
	}()

	return errors
}

func (ep *Endpoint) receiveSendResults(sendResults chan *SendResult, ctx context.Context) chan *e.ErrorInfo {
	errors := make(chan *e.ErrorInfo, 10)

	ep.wg.Add(1)
	go func () {
		defer ep.wg.Done()

		consumer, unwrappedError := ep.rabbitmqChannel.Consume(
			ep.incomingQueueName,
			"sendresult-consumer",
			false,
			false,
			false,
			false,
			amqp.Table{},
		)

		defer func () {
			unwrappedError =ep.rabbitmqChannel.Cancel("sendresult-consumer", false)

			if unwrappedError != nil {
				errors <- e.FromError(unwrappedError, "failed to cancel consumer").WithSeverity(e.Critical)
			}
		}()

		if unwrappedError != nil {
			errors <- e.FromError(unwrappedError, "failed to consume send results").WithSeverity(e.Critical)
			return
		}

		for {
			select {
			case delivery, ok := <-consumer:
				if !ok {
					return // channel closed
				}

				if delivery.CorrelationId != ep.handlerChain.Hashe.ChainID {
					continue // Обновление долетело от другой цепочки
				}
	
				var sendResult *SendResult = &SendResult{}
				unwrappedError := json.Unmarshal(delivery.Body, sendResult)
				if unwrappedError != nil {
					errors <- e.FromError(unwrappedError, "failed to unmarshal send result").WithSeverity(e.Critical)
					continue
				}
	
				sendResults <- sendResult
				delivery.Ack(false)

			case <-ctx.Done():
				return // context cancelled
			}
		}

	}()

	return errors
}

func (ep *Endpoint) ExecuteIfFilterPasses(update tele.Update) *e.ErrorInfo {
	if !ep.filter.Filter(update) {
		return nil
	}

	if ep.rabbitmqChannel == nil {
		return e.NewError("rabbitmq channel is nil", "rabbitmq channel is not initialized").WithSeverity(e.Critical)
	}

	messageTransportContext, messageTransportContextCancel := context.WithCancel(context.Background())

	sendErrors := ep.send(ep.handlerChain.Hashe.SendChannel, messageTransportContext)
	receiveErrors := ep.receiveSendResults(ep.handlerChain.Hashe.SendResultChannel, messageTransportContext)

	defer close(sendErrors)
	defer close(receiveErrors)
	defer messageTransportContextCancel() // Завершаем все горутины, которые могли бы писать в каналы, которые закрываются выше

	// Оповещение обработчиков об ошибках, возникших во время передачи сообщений через брокера
	// Помогает избежать ситуаций, когда он зависает до таймаута ожидая результат отправленного сообщения
	// В данном случае логирования конкретно в этой функции не надо
	// Ошибка попадёт в логи когда всплывёт в HandlerChain
	ep.wg.Add(1)
	go func () {
		defer ep.wg.Done()

		for {
			select {
			case err, ok := <-sendErrors:
				if !ok {
					continue // channel closed
				}

				if err.Severity == e.Ingnored {
					continue
				}

				ep.handlerChain.Hashe.SendResultChannel <- &SendResult{
					SentMessage: nil,
					Error: err,
					IsSuccess: false,
				}

			case err, ok := <-receiveErrors:
				if !ok {
					continue // channel closed
				}

				if err.Severity == e.Ingnored {
					continue
				}

				ep.handlerChain.Hashe.SendResultChannel <- &SendResult{
					SentMessage: nil,
					Error: err,
					IsSuccess: false,
				}
			}
		}
	}()

	err := ep.handlerChain.WithWaitGroup(ep.wg).Run(update)

	return err
}

type Router struct {
	Endpoints []Endpoint
	ErrorChannel chan *e.ErrorInfo
	RabbitmqChannel *amqp.Channel
}

func (r *Router) Dispatch(update tele.Update) *e.ErrorInfo {
	for _, endpoint := range r.Endpoints {
		err := endpoint.WithRabbitMQChannel(r.RabbitmqChannel).ExecuteIfFilterPasses(update)
		if !err.IsNil() {
			r.ErrorChannel <- err.PushStack().WithData(map[string]any{"endpoint name": endpoint.Name})
		}
	}

	return nil
}
