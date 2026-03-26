package telegram

type HandlerResponse struct {
	Method    string         `json:"method"`
	SendData  map[string]any `json:"send_data"`
	SenderBot string         `json:"sender_bot"` // Для механики зеркал
}