package notification

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChatDetectiveORG/shared/telegram/rawmessage"
	"github.com/ChatDetectiveORG/shared/utils"
	tele "gopkg.in/telebot.v4"
)

func TestBuildNoticeSummaryChecklistTitleCustomEmoji(t *testing.T) {
	raw := json.RawMessage(`{
		"checklist": {
			"title": "🔥",
			"title_entities": [{
				"type": "custom_emoji",
				"offset": 0,
				"length": 2,
				"custom_emoji_id": "header-emoji"
			}]
		}
	}`)

	text, opts := buildNoticeSummary(nil, raw)
	if !strings.Contains(text, rawmessage.ChecklistOriginNotice) {
		t.Fatalf("text must contain checklist notice, got %q", text)
	}
	if !strings.Contains(text, "🔥") {
		t.Fatalf("text must contain checklist title, got %q", text)
	}
	if opts == nil || len(opts.Entities) != 1 {
		t.Fatalf("entities = %#v, want one custom emoji", opts)
	}
	wantOffset := utils.TgLen(rawmessage.ChecklistOriginNotice + "\n")
	if opts.Entities[0].Offset != wantOffset {
		t.Fatalf("entity offset = %d, want %d", opts.Entities[0].Offset, wantOffset)
	}
	if opts.Entities[0].CustomEmojiID != "header-emoji" {
		t.Fatalf("custom_emoji_id = %q", opts.Entities[0].CustomEmojiID)
	}
}

func TestBuildNoticeSummaryChecklistShiftsExistingSummaryEntities(t *testing.T) {
	raw := json.RawMessage(`{"checklist":{"title":"Todo","tasks":[{"id":1,"text":"x"}]}}`)
	msg := &tele.Message{
		ReplyTo: &tele.Message{
			Sender: &tele.User{Username: "alice"},
			Chat:   &tele.Chat{Type: tele.ChatPrivate},
			Text:   "original",
		},
	}

	_, opts := buildNoticeSummary(msg, raw)
	if opts == nil || len(opts.Entities) == 0 {
		t.Fatal("expected summary entities to be preserved")
	}
	prefixLen := utils.TgLen(rawmessage.ChecklistOriginNotice + "\nTodo\n")
	for _, ent := range opts.Entities {
		if ent.Type != tele.EntityTextLink {
			continue
		}
		if ent.Offset <= prefixLen {
			t.Fatalf("summary entity offset = %d, want > %d", ent.Offset, prefixLen)
		}
	}
}
