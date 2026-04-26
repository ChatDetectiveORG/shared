package telegram

import (
	"encoding/json"
	"testing"

	tele "gopkg.in/telebot.v4"
)

func TestOutgoingRequestEnvelopeMessageJSONRoundTrip(t *testing.T) {
	payload, err := json.Marshal(NewOutgoingMessageRequest(&tele.Message{
		ID:   42,
		Text: "hello",
		Chat: &tele.Chat{ID: 1001},
	}, false))
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var request OutgoingRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("unmarshal outgoing request: %v", err)
	}
	if request.Kind != OutgoingRequestKindMessage {
		t.Fatalf("expected kind %q, got %q", OutgoingRequestKindMessage, request.Kind)
	}
	if request.Message == nil {
		t.Fatal("expected message payload")
	}
	if request.Message.Text != "hello" {
		t.Fatalf("expected text %q, got %q", "hello", request.Message.Text)
	}
	if request.ParseModeEnabled {
		t.Fatal("expected parse_mode_enabled false")
	}
}

func TestOutgoingRequestEnvelopeAlbumJSONRoundTrip(t *testing.T) {
	payload, err := json.Marshal(NewOutgoingAlbumRequest(&MediaGroup{
		Chat: &tele.Chat{ID: 1001},
		Messages: []*tele.Message{
			{Photo: &tele.Photo{}},
		},
		Silent: true,
	}, false))
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var request OutgoingRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("unmarshal outgoing request: %v", err)
	}
	if request.Kind != OutgoingRequestKindAlbum {
		t.Fatalf("expected kind %q, got %q", OutgoingRequestKindAlbum, request.Kind)
	}
	if request.Album == nil {
		t.Fatal("expected album payload")
	}
	if !request.Album.Silent {
		t.Fatal("expected silent album flag to survive parsing")
	}
	if len(request.Album.Messages) != 1 {
		t.Fatalf("expected 1 album message, got %d", len(request.Album.Messages))
	}
}

func TestOutgoingRequestEnvelopeEditMessageJSONRoundTrip(t *testing.T) {
	payload, err := json.Marshal(NewOutgoingEditMessageRequest(&tele.Message{
		ID:   42,
		Text: "edited",
		Chat: &tele.Chat{ID: 1001},
	}, true))
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var request OutgoingRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("unmarshal outgoing request: %v", err)
	}
	if request.Kind != OutgoingRequestKindEdit {
		t.Fatalf("expected kind %q, got %q", OutgoingRequestKindEdit, request.Kind)
	}
	if request.Message == nil || request.Message.Text != "edited" {
		t.Fatalf("expected edited message payload, got %#v", request.Message)
	}
	if !request.ParseModeEnabled {
		t.Fatal("expected parse_mode_enabled true")
	}
}

func TestOutgoingRequestEnvelopeDeleteMessageJSONRoundTrip(t *testing.T) {
	payload, err := json.Marshal(NewOutgoingDeleteMessageRequest(&tele.Message{
		ID:   42,
		Chat: &tele.Chat{ID: 1001},
	}, false))
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var request OutgoingRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("unmarshal outgoing request: %v", err)
	}
	if request.Kind != OutgoingRequestKindDelete {
		t.Fatalf("expected kind %q, got %q", OutgoingRequestKindDelete, request.Kind)
	}
	if request.Message == nil || request.Message.ID != 42 {
		t.Fatalf("expected delete message payload, got %#v", request.Message)
	}
}

func TestOutgoingRequestEnvelopeCallbackJSONRoundTrip(t *testing.T) {
	cb := &tele.Callback{ID: "cb-1", Data: "x"}
	resp := &tele.CallbackResponse{Text: "ok", ShowAlert: true}
	payload, err := json.Marshal(NewOutgoingCallbackRequest(cb, resp))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var request OutgoingRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if request.Kind != OutgoingRequestKindCallback {
		t.Fatalf("expected kind %q, got %q", OutgoingRequestKindCallback, request.Kind)
	}
	if request.Callback == nil || request.Callback.ID != "cb-1" {
		t.Fatalf("expected callback, got %#v", request.Callback)
	}
	if request.CallbackResponse == nil || request.CallbackResponse.Text != "ok" || !request.CallbackResponse.ShowAlert {
		t.Fatalf("expected callback response, got %#v", request.CallbackResponse)
	}
}

func TestOutgoingRequestParseModeEnabledJSONRoundTrip(t *testing.T) {
	payload, err := json.Marshal(NewOutgoingMessageRequest(&tele.Message{
		Text: "x",
		Chat: &tele.Chat{ID: 1},
	}, true))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var request OutgoingRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !request.ParseModeEnabled {
		t.Fatal("expected parse_mode_enabled true after round-trip")
	}
}
