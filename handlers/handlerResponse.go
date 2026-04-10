package handlers

import (
	e "github.com/ChatDetectiveORG/shared/errors"
	tele "gopkg.in/telebot.v4"
)

// SendResult — ответ message-sender (или аналога) после попытки отправки в Telegram.
// CorrelationID должен совпадать с тем, что ушло в AMQP CorrelationId при Publish.
type SendResult struct {
	CorrelationID string        `json:"correlation_id"`
	SentMessage   *tele.Message `json:"message"`
	Error         e.ErrorInfo   `json:"error"`
	IsSuccess     bool          `json:"is_success"`
}
