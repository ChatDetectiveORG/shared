package telegram

import (
	"testing"

	tele "gopkg.in/telebot.v4"
)

func TestBuildMediaGroupMovesCaptionToFirstMessage(t *testing.T) {
	msgs := []*tele.Message{
		{
			ID:      20,
			Video:   &tele.Video{},
			Caption: "album caption",
			Entities: tele.Entities{
				{
					Type:   tele.EntityBold,
					Offset: 0,
					Length: 5,
				},
			},
			Protected: true,
		},
		{
			ID:    10,
			Photo: &tele.Photo{},
		},
	}

	group, ok := BuildMediaGroup(msgs)
	if !ok {
		t.Fatal("expected media group to be built")
	}
	if len(group) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(group))
	}
	if group[0].Photo == nil {
		t.Fatal("expected first message to stay photo after sorting")
	}
	if group[0].Photo.Caption != "album caption" {
		t.Fatalf("expected caption to be moved to first item, got %q", group[0].Photo.Caption)
	}
	if len(group[0].Entities) != 1 {
		t.Fatalf("expected entities to be copied to first item, got %d", len(group[0].Entities))
	}
	if !group[0].Protected {
		t.Fatal("expected send options to be hidden in first message")
	}
	if group[1].Video == nil {
		t.Fatal("expected second message to be video")
	}
	if group[1].Video.Caption != "" {
		t.Fatalf("expected non-leading album items to have empty caption, got %q", group[1].Video.Caption)
	}
}

func TestBuildMediaGroupUsesTextAsFallbackCaption(t *testing.T) {
	msgs := []*tele.Message{
		{
			ID:       10,
			Document: &tele.Document{},
			Text:     "fallback caption",
		},
	}

	group, ok := BuildMediaGroup(msgs)
	if !ok {
		t.Fatal("expected media group to be built")
	}
	if len(group) != 1 {
		t.Fatalf("expected 1 message, got %d", len(group))
	}
	if group[0].Document == nil {
		t.Fatal("expected document message")
	}
	if group[0].Document.Caption != "fallback caption" {
		t.Fatalf("expected text to be used as caption fallback, got %q", group[0].Document.Caption)
	}
}

func TestBuildMediaGroupSkipsUnsupportedMessages(t *testing.T) {
	msgs := []*tele.Message{
		{
			ID:    10,
			Voice: &tele.Voice{},
		},
	}

	group, ok := BuildMediaGroup(msgs)
	if ok {
		t.Fatal("expected unsupported media group to fail")
	}
	if group != nil {
		t.Fatal("expected nil group for unsupported media")
	}
}
