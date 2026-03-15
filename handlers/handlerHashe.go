package handlers

type handlerChainHashe struct {
	args map[string]any
	responses []handlerResponse
}

func (hch handlerChainHashe) Init() *handlerChainHashe {
	return &handlerChainHashe{
		args: make(map[string]any),
	}
}

func (hch *handlerChainHashe) Add(name string, value interface{}) *handlerChainHashe {
	hch.args[name] = value

	return  hch
}

func (hch *handlerChainHashe) Get(name string) (interface{}, bool) {
	v, exists := hch.args[name]

	if !exists {
		return nil, false
	}

	return v, true
}

func (hch *handlerChainHashe) AddResponse(response handlerResponse) *handlerChainHashe {
	hch.responses = append(hch.responses, response)

	return hch
}

func (hch *handlerChainHashe) Trunc() *handlerChainHashe {
	clear(hch.args)

	return hch
}
