package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	tele "gopkg.in/telebot.v4"
)

type OutgoingRequestKind string

const (
	OutgoingRequestKindMessage OutgoingRequestKind = "message"
	OutgoingRequestKindAlbum   OutgoingRequestKind = "album"
)

type OutgoingRequest struct {
	Kind    OutgoingRequestKind `json:"kind"`
	Message *tele.Message       `json:"message,omitempty"`
	Album   *MediaGroup         `json:"album,omitempty"`
}

func NewOutgoingMessageRequest(msg *tele.Message) *OutgoingRequest {
	return &OutgoingRequest{
		Kind:    OutgoingRequestKindMessage,
		Message: msg,
	}
}

func NewOutgoingAlbumRequest(album *MediaGroup) *OutgoingRequest {
	return &OutgoingRequest{
		Kind:  OutgoingRequestKindAlbum,
		Album: album,
	}
}

func ParseOutgoingRequest(data []byte) (*OutgoingRequest, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, errors.New("outgoing request payload is empty")
	}

	var envelope struct {
		Kind OutgoingRequestKind `json:"kind"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Kind != "" {
		var request OutgoingRequest
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, fmt.Errorf("unmarshal outgoing request: %w", err)
		}

		switch request.Kind {
		case OutgoingRequestKindMessage:
			if request.Message == nil {
				return nil, errors.New("outgoing message payload is nil")
			}
		case OutgoingRequestKindAlbum:
			if request.Album == nil {
				return nil, errors.New("outgoing album payload is nil")
			}
		default:
			return nil, fmt.Errorf("unsupported outgoing request kind: %s", request.Kind)
		}

		return &request, nil
	}

	var message tele.Message
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, fmt.Errorf("unmarshal legacy outgoing message: %w", err)
	}

	return NewOutgoingMessageRequest(&message), nil
}
