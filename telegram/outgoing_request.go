package telegram

import (
	tele "gopkg.in/telebot.v4"
)

type OutgoingRequestKind string

const (
	OutgoingRequestKindMessage  OutgoingRequestKind = "message"
	OutgoingRequestKindAlbum    OutgoingRequestKind = "album"
	OutgoingRequestKindEdit     OutgoingRequestKind = "edit_message"
	OutgoingRequestKindDelete   OutgoingRequestKind = "delete_message"
	OutgoingRequestKindCallback OutgoingRequestKind = "callback_query"
)

type OutgoingRequest struct {
	Kind             OutgoingRequestKind    `json:"kind"`
	MirrorID         string                 `json:"mirror_id,omitempty"`
	Message          *tele.Message          `json:"message,omitempty"`
	Album            *MediaGroup            `json:"album,omitempty"`
	Callback         *tele.Callback         `json:"callback,omitempty"`
	CallbackResponse *tele.CallbackResponse `json:"callback_response,omitempty"`
	ParseModeEnabled bool                   `json:"parse_mode_enabled,omitempty"`
}

func NewOutgoingMessageRequest(msg *tele.Message, parseModeEnabled bool) *OutgoingRequest {
	return &OutgoingRequest{
		Kind:             OutgoingRequestKindMessage,
		Message:          msg,
		ParseModeEnabled: parseModeEnabled,
	}
}

func NewOutgoingEditMessageRequest(msg *tele.Message, parseModeEnabled bool) *OutgoingRequest {
	return &OutgoingRequest{
		Kind:             OutgoingRequestKindEdit,
		Message:          msg,
		ParseModeEnabled: parseModeEnabled,
	}
}

func NewOutgoingDeleteMessageRequest(msg *tele.Message, parseModeEnabled bool) *OutgoingRequest {
	return &OutgoingRequest{
		Kind:             OutgoingRequestKindDelete,
		Message:          msg,
		ParseModeEnabled: parseModeEnabled,
	}
}

func NewOutgoingAlbumRequest(album *MediaGroup, parseModeEnabled bool) *OutgoingRequest {
	return &OutgoingRequest{
		Kind:             OutgoingRequestKindAlbum,
		Album:            album,
		ParseModeEnabled: parseModeEnabled,
	}
}

// NewOutgoingCallbackRequest enqueues answerCallbackQuery (alerts, optional URL, etc.).
func NewOutgoingCallbackRequest(cb *tele.Callback, resp *tele.CallbackResponse) *OutgoingRequest {
	return &OutgoingRequest{
		Kind:             OutgoingRequestKindCallback,
		Callback:         cb,
		CallbackResponse: resp,
		ParseModeEnabled: false,
	}
}

// func ParseOutgoingRequest(data []byte) (*OutgoingRequest, error) {
// 	data = bytes.TrimSpace(data)
// 	if len(data) == 0 {
// 		return nil, errors.New("outgoing request payload is empty")
// 	}

// 	var envelope struct {
// 		Kind OutgoingRequestKind `json:"kind"`
// 	}
// 	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Kind != "" {
// 		var request OutgoingRequest
// 		if err := json.Unmarshal(data, &request); err != nil {
// 			return nil, fmt.Errorf("unmarshal outgoing request: %w", err)
// 		}

// 		switch request.Kind {
// 		case OutgoingRequestKindMessage:
// 			if request.Message == nil {
// 				return nil, errors.New("outgoing message payload is nil")
// 			}
// 		case OutgoingRequestKindAlbum:
// 			if request.Album == nil {
// 				return nil, errors.New("outgoing album payload is nil")
// 			}
// 		default:
// 			return nil, fmt.Errorf("unsupported outgoing request kind: %s", request.Kind)
// 		}

// 		return &request, nil
// 	}

// 	var message tele.Message
// 	if err := json.Unmarshal(data, &message); err != nil {
// 		return nil, fmt.Errorf("unmarshal legacy outgoing message: %w", err)
// 	}

// 	return NewOutgoingMessageRequest(&message), nil
// }
