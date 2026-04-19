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
