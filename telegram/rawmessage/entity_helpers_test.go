package rawmessage

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ChatDetectiveORG/shared/utils"
)

func TestOffsetRawEntitiesPreservesCustomEmojiID(t *testing.T) {
	raw := []any{
		map[string]any{
			"type":            "custom_emoji",
			"offset":          float64(0),
			"length":          float64(2),
			"custom_emoji_id": "emoji-123",
		},
	}
	got, ok := offsetRawEntities(raw, 10).([]any)
	if !ok {
		t.Fatalf("result type = %T", offsetRawEntities(raw, 10))
	}
	ent := got[0].(map[string]any)
	if ent["offset"] != float64(10) && ent["offset"] != 10 {
		t.Fatalf("offset = %#v", ent["offset"])
	}
	if ent["custom_emoji_id"] != "emoji-123" {
		t.Fatalf("custom_emoji_id = %#v", ent["custom_emoji_id"])
	}
}

func TestChecklistPollPreservesCustomEmojiEntities(t *testing.T) {
	raw := map[string]any{
		"checklist": map[string]any{
			"title": "🔥 list",
			"title_entities": []any{
				map[string]any{
					"type":            "custom_emoji",
					"offset":          float64(0),
					"length":          float64(2),
					"custom_emoji_id": "title-emoji",
				},
			},
			"tasks": []any{
				map[string]any{
					"id":   float64(1),
					"text": "🔥 one",
					"text_entities": []any{
						map[string]any{
							"type":            "custom_emoji",
							"offset":          float64(0),
							"length":          float64(2),
							"custom_emoji_id": "task-emoji-1",
						},
					},
				},
				map[string]any{
					"id":   float64(2),
					"text": "two",
				},
			},
		},
	}

	payload, err := buildChecklistPollPayload(raw, ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("buildChecklistPollPayload: %v", err)
	}

	titleOffset := utils.TgLen(ChecklistOriginNotice + "\n")

	questionEntities, ok := payload["question_entities"].([]any)
	if !ok || len(questionEntities) != 1 {
		t.Fatalf("question_entities = %#v", payload["question_entities"])
	}
	qEnt := questionEntities[0].(map[string]any)
	if intFromAny(qEnt["offset"]) != titleOffset {
		t.Fatalf("question entity offset = %v, want %d", qEnt["offset"], titleOffset)
	}
	if qEnt["custom_emoji_id"] != "title-emoji" {
		t.Fatalf("question custom_emoji_id = %#v", qEnt["custom_emoji_id"])
	}

	options := payload["options"].([]map[string]any)
	if len(options) != 2 {
		t.Fatalf("options len = %d", len(options))
	}
	optEntities, ok := options[0]["text_entities"].([]any)
	if !ok || len(optEntities) != 1 {
		t.Fatalf("option text_entities = %#v", options[0]["text_entities"])
	}
	oEnt := optEntities[0].(map[string]any)
	if oEnt["custom_emoji_id"] != "task-emoji-1" {
		t.Fatalf("option custom_emoji_id = %#v", oEnt["custom_emoji_id"])
	}
}

func TestChecklistTextFallbackPreservesCustomEmojiEntities(t *testing.T) {
	raw := map[string]any{
		"checklist": map[string]any{
			"title": "🔥 list",
			"title_entities": []any{
				map[string]any{
					"type":            "custom_emoji",
					"offset":          float64(0),
					"length":          float64(2),
					"custom_emoji_id": "title-emoji",
				},
			},
			"tasks": []any{
				map[string]any{
					"id":   float64(1),
					"text": "🔥 only",
					"text_entities": []any{
						map[string]any{
							"type":            "custom_emoji",
							"offset":          float64(0),
							"length":          float64(2),
							"custom_emoji_id": "task-emoji",
						},
					},
				},
			},
		},
	}

	payload, err := buildChecklistTextNotificationPayload(raw)
	if err != nil {
		t.Fatalf("buildChecklistTextNotificationPayload: %v", err)
	}

	prefixLen := utils.TgLen(ChecklistOriginNotice + "\n\n")
	entities, ok := payload["entities"].([]any)
	if !ok || len(entities) != 2 {
		t.Fatalf("entities = %#v", payload["entities"])
	}

	titleEnt := entities[0].(map[string]any)
	if intFromAny(titleEnt["offset"]) != prefixLen {
		t.Fatalf("title entity offset = %v, want %d", titleEnt["offset"], prefixLen)
	}
	if titleEnt["custom_emoji_id"] != "title-emoji" {
		t.Fatalf("title custom_emoji_id = %#v", titleEnt["custom_emoji_id"])
	}

	taskEnt := entities[1].(map[string]any)
	wantTaskOffset := prefixLen + utils.TgLen("🔥 list\n- ")
	if intFromAny(taskEnt["offset"]) != wantTaskOffset {
		t.Fatalf("task entity offset = %v, want %d", taskEnt["offset"], wantTaskOffset)
	}
	if taskEnt["custom_emoji_id"] != "task-emoji" {
		t.Fatalf("task custom_emoji_id = %#v", taskEnt["custom_emoji_id"])
	}
}

func TestBuildSendPayloadChecklistTextIncludesEntities(t *testing.T) {
	raw := json.RawMessage(`{
		"checklist": {
			"title": "🔥",
			"title_entities": [{"type":"custom_emoji","offset":0,"length":2,"custom_emoji_id":"e1"}],
			"tasks": [{"id":1,"text":"done"}]
		}
	}`)
	_, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("BuildSendPayload: %v", err)
	}
	entities, ok := payload["entities"].([]any)
	if !ok || len(entities) != 1 {
		t.Fatalf("entities = %#v", payload["entities"])
	}
	ent := entities[0].(map[string]any)
	if ent["custom_emoji_id"] != "e1" {
		t.Fatalf("custom_emoji_id = %#v", ent["custom_emoji_id"])
	}
}

func TestConvertRichBlockPhotoPreservesCaptionEntities(t *testing.T) {
	raw := json.RawMessage(`{
		"rich_message": {
			"blocks": [{
				"type": "photo",
				"photo": [{"file_id": "pid"}],
				"caption": "🔥 caption",
				"caption_entities": [{
					"type": "custom_emoji",
					"offset": 0,
					"length": 2,
					"custom_emoji_id": "cap-emoji"
				}]
			}]
		}
	}`)
	_, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 1})
	if err != nil {
		t.Fatal(err)
	}
	rich := payload["rich_message"].(map[string]any)
	block := rich["blocks"].([]any)[0].(map[string]any)
	if block["caption"] != "🔥 caption" {
		t.Fatalf("caption = %#v", block["caption"])
	}
	want := []any{
		map[string]any{
			"type":            "custom_emoji",
			"offset":          float64(0),
			"length":          float64(2),
			"custom_emoji_id": "cap-emoji",
		},
	}
	if !reflect.DeepEqual(block["caption_entities"], want) {
		t.Fatalf("caption_entities = %#v, want %#v", block["caption_entities"], want)
	}
}
