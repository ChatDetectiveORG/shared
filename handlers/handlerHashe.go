package handlers

type HandlerChainHashe struct {
	args      map[string]any
	responses []handlerResponse
}

func (hch HandlerChainHashe) Init() *HandlerChainHashe {
	return &HandlerChainHashe{
		args: make(map[string]any),
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

func (hch *HandlerChainHashe) AddResponse(response handlerResponse) *HandlerChainHashe {
	hch.responses = append(hch.responses, response)

	return hch
}

func (hch *HandlerChainHashe) Trunc() *HandlerChainHashe {
	clear(hch.args)

	return hch
}
