package handlers

import (
	e "github.com/ChatDetectiveORG/shared/errors"
	tele "gopkg.in/telebot.v4"
)

// SendResult — ответ message-sender (или аналога) после попытки отправки в Telegram.
// CorrelationID должен совпадать с тем, что ушло в AMQP CorrelationId при Publish.
type SendResult struct {
	CorrelationID string          `json:"correlation_id"`
	SentMessage   *tele.Message   `json:"message"`
	SentAlbum     []*tele.Message `json:"album,omitempty"`
	Error         *e.ErrorInfo    `json:"error"`
	IsSuccess     bool            `json:"is_success"`
}

func (sr *SendResult) FirstSentMessage() *tele.Message {
	if sr == nil {
		return nil
	}
	if sr.SentMessage != nil {
		return sr.SentMessage
	}
	if len(sr.SentAlbum) == 0 {
		return nil
	}
	return sr.SentAlbum[0]
}
