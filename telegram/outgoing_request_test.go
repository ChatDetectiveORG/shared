package telegram

import (
	"encoding/json"
	"testing"

	tele "gopkg.in/telebot.v4"
)

func TestParseOutgoingRequestEnvelopeMessage(t *testing.T) {
	payload, err := json.Marshal(NewOutgoingMessageRequest(&tele.Message{
		ID:   42,
		Text: "hello",
		Chat: &tele.Chat{ID: 1001},
	}))
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request, err := ParseOutgoingRequest(payload)
	if err != nil {
		t.Fatalf("parse outgoing request: %v", err)
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
}

func TestParseOutgoingRequestEnvelopeAlbum(t *testing.T) {
	payload, err := json.Marshal(NewOutgoingAlbumRequest(&MediaGroup{
		Chat: &tele.Chat{ID: 1001},
		Messages: []*tele.Message{
			{Photo: &tele.Photo{}},
		},
		Silent: true,
	}))
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request, err := ParseOutgoingRequest(payload)
	if err != nil {
		t.Fatalf("parse outgoing request: %v", err)
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

func TestParseOutgoingRequestLegacyMessagePayload(t *testing.T) {
	payload, err := json.Marshal(&tele.Message{
		ID:   7,
		Text: "legacy",
		Chat: &tele.Chat{ID: 99},
	})
	if err != nil {
		t.Fatalf("marshal legacy payload: %v", err)
	}

	request, err := ParseOutgoingRequest(payload)
	if err != nil {
		t.Fatalf("parse legacy outgoing request: %v", err)
	}
	if request.Kind != OutgoingRequestKindMessage {
		t.Fatalf("expected kind %q, got %q", OutgoingRequestKindMessage, request.Kind)
	}
	if request.Message == nil || request.Message.Text != "legacy" {
		t.Fatalf("expected legacy message text %q, got %#v", "legacy", request.Message)
	}
}
