package handlers

import (
	e "github.com/ChatDetectiveORG/shared/errors"
	tele "gopkg.in/telebot.v4"
)

type HandlerResponse struct {
	ToSend *tele.Message     `json:"to_send"`
	SenderBot string         `json:"sender_bot"`
	FromPodID string         `json:"from_pod_id"`
	ChainID   string         `json:"chain_id"`
	// Если false, не будут возвращаться объекты отправленных сообщений
	//
	// Ошибки отправки возвращаются ВСЕГДА
	ReturnSendResult bool `json:"return_send_result"`
}

func (hr *HandlerResponse) WithChainID(chainID string) *HandlerResponse {
	hr.ChainID = chainID
	return hr
}

type SendResult struct {
	SentMessage *tele.Message `json:"message"`
	Error   *e.ErrorInfo      `json:"error"`
	IsSuccess bool            `json:"is_success"`
	ChainID   string         `json:"chain_id"`
}
