package notification

import (
	"encoding/json"
	"testing"

	"github.com/ChatDetectiveORG/shared/utils"
)

func TestClassifyEditShortPlainText(t *testing.T) {
	oldRaw := json.RawMessage(`{"text":"old"}`)
	newRaw := json.RawMessage(`{"text":"new"}`)
	if ClassifyEdit(oldRaw, newRaw, "") != StrategyTextCombined {
		t.Fatalf("expected combined strategy")
	}
}

func TestClassifyEditLongText(t *testing.T) {
	long := string(make([]rune, 2000))
	oldRaw, _ := json.Marshal(map[string]string{"text": long})
	newRaw, _ := json.Marshal(map[string]string{"text": long})
	if ClassifyEdit(oldRaw, newRaw, "") != StrategyReplayWithNotice {
		t.Fatalf("expected replay strategy for long text")
	}
	if utils.TgLen(long)+utils.TgLen(long) <= 3900 {
		t.Fatal("test setup: expected combined length above threshold")
	}
}

func TestClassifyEditMediaGroup(t *testing.T) {
	if ClassifyEdit(nil, nil, "group-hash") != StrategyMediaGroup {
		t.Fatal("expected media group strategy")
	}
}

func TestClassifyEditMediaWithFormattedCaption(t *testing.T) {
	oldRaw := json.RawMessage(`{
		"animation":{"file_id":"old-id"},
		"caption":"old",
		"caption_entities":[{"type":"bold","offset":0,"length":3}]
	}`)
	newRaw := json.RawMessage(`{
		"animation":{"file_id":"new-id"},
		"caption":"new",
		"caption_entities":[{"type":"italic","offset":0,"length":3}]
	}`)
	if ClassifyEdit(oldRaw, newRaw, "") != StrategyReplayWithNotice {
		t.Fatal("expected replay strategy for media with formatted caption")
	}
}

func TestClassifyEditRichMessageUsesReplay(t *testing.T) {
	oldRaw := json.RawMessage(`{"rich_message":{"blocks":[{"type":"paragraph","text":"old"}]}}`)
	newRaw := json.RawMessage(`{"rich_message":{"blocks":[{"type":"paragraph","text":"new"}]}}`)
	if ClassifyEdit(oldRaw, newRaw, "") != StrategyReplayWithNotice {
		t.Fatal("expected replay strategy for rich_message edit")
	}
}

func TestClassifyDeleteShortText(t *testing.T) {
	raw := json.RawMessage(`{"text":"deleted"}`)
	if ClassifyDelete(raw, "") != StrategyTextCombined {
		t.Fatal("expected combined delete strategy")
	}
}
