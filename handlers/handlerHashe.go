package handlers

import "github.com/google/uuid"

type HandlerChainHashe struct {
	args      map[string]any
	SendChannel chan *HandlerResponse
	SendResultChannel chan *SendResult
	ChainID string
}

func (hch HandlerChainHashe) Init() *HandlerChainHashe {
	return &HandlerChainHashe{
		args: make(map[string]any),
		SendChannel: make(chan *HandlerResponse),
		SendResultChannel: make(chan *SendResult),
		ChainID: uuid.New().String(),
	}
}

func (hch *HandlerChainHashe) SendToChannel(handlerResponse *HandlerResponse) {
	hch.SendChannel <- handlerResponse.WithChainID(hch.ChainID)
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
	clear(hch.args)

	return hch
}
