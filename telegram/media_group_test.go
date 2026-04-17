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
				tele.MessageEntity{
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
	if len(group.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(group.Messages))
	}
	if group.Messages[0].Photo == nil {
		t.Fatal("expected first message to stay photo after sorting")
	}
	if group.Messages[0].Photo.Caption != "album caption" {
		t.Fatalf("expected caption to be moved to first item, got %q", group.Messages[0].Photo.Caption)
	}
	if len(group.Messages[0].Entities) != 1 {
		t.Fatalf("expected entities to be copied to first item, got %d", len(group.Messages[0].Entities))
	}
	if !group.Messages[0].Protected {
		t.Fatal("expected send options to be hidden in first message")
	}
	if group.Messages[1].Video == nil {
		t.Fatal("expected second message to be video")
	}
	if group.Messages[1].Video.Caption != "" {
		t.Fatalf("expected non-leading album items to have empty caption, got %q", group.Messages[1].Video.Caption)
	}

	album, ok := group.ToAlbum()
	if !ok {
		t.Fatal("expected album conversion to succeed")
	}
	if len(album) != 2 {
		t.Fatalf("expected 2 album items, got %d", len(album))
	}
	if got := album[0].InputMedia().Caption; got != "album caption" {
		t.Fatalf("expected caption on first album item, got %q", got)
	}
	if len(album[0].InputMedia().Entities) != 1 {
		t.Fatalf("expected caption entities on first album item, got %d", len(album[0].InputMedia().Entities))
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
	if len(group.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(group.Messages))
	}
	if group.Messages[0].Document == nil {
		t.Fatal("expected document message")
	}
	if group.Messages[0].Document.Caption != "fallback caption" {
		t.Fatalf("expected text to be used as caption fallback, got %q", group.Messages[0].Document.Caption)
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

func TestExtractInputtablePreservesAnimationCaptionAbove(t *testing.T) {
	msg := &tele.Message{
		Caption:      "gif caption",
		CaptionAbove: true,
		Animation:    &tele.Animation{},
	}

	input := ExtractInputtable(msg)
	if input == nil {
		t.Fatal("expected animation inputtable")
	}

	media := input.InputMedia()
	if media.Caption != "gif caption" {
		t.Fatalf("expected caption to be preserved, got %q", media.Caption)
	}
	if !media.CaptionAbove {
		t.Fatal("expected caption above flag to be preserved")
	}
}
