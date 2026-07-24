package rawmessage

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/ChatDetectiveORG/shared/utils"
	tele "gopkg.in/telebot.v4"
)

func TestMarshalPreservesUnknownFields(t *testing.T) {
	raw := json.RawMessage(`{"message_id":1,"text":"hi","unknown_field":42}`)
	var msg tele.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(&msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != string(raw) {
		t.Fatalf("got %s want %s", out, raw)
	}
}

func TestBuildSendPayloadPhoto(t *testing.T) {
	raw := json.RawMessage(`{
		"message_id": 10,
		"date": 1,
		"photo": [{"file_id":"photo-id","file_unique_id":"u","width":100,"height":100,"file_size":1000}]
	}`)
	method, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("BuildSendPayload: %v", err)
	}
	if method != "sendPhoto" {
		t.Fatalf("method = %q", method)
	}
	if payload["chat_id"] != int64(42) {
		t.Fatalf("chat_id = %#v", payload["chat_id"])
	}
	if payload["photo"] != "photo-id" {
		t.Fatalf("photo = %#v", payload["photo"])
	}
	if _, ok := payload["message_id"]; ok {
		t.Fatalf("read-only message_id not stripped")
	}
}

func TestBuildSendPayloadSupportedMedia(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		mediaKey  string
		mediaJSON string
		fileID    string
	}{
		{name: "photo", method: "sendPhoto", mediaKey: "photo", mediaJSON: `[{"file_id":"small","file_size":10},{"file_id":"photo-id","file_size":20}]`, fileID: "photo-id"},
		{name: "video", method: "sendVideo", mediaKey: "video", mediaJSON: `{"file_id":"video-id","file_unique_id":"uv"}`, fileID: "video-id"},
		{name: "document", method: "sendDocument", mediaKey: "document", mediaJSON: `{"file_id":"document-id","mime_type":"application/pdf"}`, fileID: "document-id"},
		{name: "audio", method: "sendAudio", mediaKey: "audio", mediaJSON: `{"file_id":"audio-id","duration":2}`, fileID: "audio-id"},
		{name: "voice", method: "sendVoice", mediaKey: "voice", mediaJSON: `{"file_id":"voice-id","file_size":12}`, fileID: "voice-id"},
		{name: "video note", method: "sendVideoNote", mediaKey: "video_note", mediaJSON: `{"file_id":"note-id","length":120}`, fileID: "note-id"},
		{name: "animation", method: "sendAnimation", mediaKey: "animation", mediaJSON: `{"file_id":"animation-id","thumbnail":{"file_id":"thumb"}}`, fileID: "animation-id"},
		{name: "sticker", method: "sendSticker", mediaKey: "sticker", mediaJSON: `{"file_id":"sticker-id","emoji":"x"}`, fileID: "sticker-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := json.RawMessage(`{
				"message_id": 10,
				"date": 20,
				"chat": {"id": 30},
				"from": {"id": 40},
				"business_connection_id": "business",
				"` + tt.mediaKey + `": ` + tt.mediaJSON + `
			}`)
			method, payload, err := BuildSendPayload(raw, ReplayOptions{
				TargetChatID:     42,
				ReplyToMessageID: 7,
			})
			if err != nil {
				t.Fatalf("BuildSendPayload: %v", err)
			}
			if method != tt.method {
				t.Fatalf("method = %q, want %q", method, tt.method)
			}
			if payload[tt.mediaKey] != tt.fileID {
				t.Fatalf("%s = %#v, want %q", tt.mediaKey, payload[tt.mediaKey], tt.fileID)
			}
			if payload["chat_id"] != int64(42) || payload["reply_to_message_id"] != 7 {
				t.Fatalf("replay addressing fields = %#v", payload)
			}
			if _, exists := payload["business_connection_id"]; exists {
				t.Fatalf("notification replay must not include business_connection_id: %#v", payload)
			}
			for _, forbidden := range []string{
				"message_id", "date", "chat", "from",
				"file_unique_id", "file_size", "mime_type", "thumbnail",
			} {
				if _, exists := payload[forbidden]; exists {
					t.Fatalf("output-only field %q leaked into payload: %#v", forbidden, payload)
				}
			}
			if _, nested := payload[tt.mediaKey].(map[string]any); nested {
				t.Fatalf("nested media object leaked into payload: %#v", payload)
			}
		})
	}
}

func TestBuildSendPayloadPreservesCaptionEntities(t *testing.T) {
	raw := json.RawMessage(`{
		"video": {"file_id":"video-id"},
		"caption": "bold link emoji",
		"caption_entities": [
			{"type":"bold","offset":0,"length":4},
			{"type":"text_link","offset":5,"length":4,"url":"https://example.com"},
			{"type":"custom_emoji","offset":10,"length":5,"custom_emoji_id":"emoji-id"}
		],
		"parse_mode": "HTML",
		"caption_parse_mode": "MarkdownV2",
		"show_caption_above_media": true,
		"has_spoiler": true
	}`)
	_, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("BuildSendPayload: %v", err)
	}
	if payload["caption"] != "bold link emoji" {
		t.Fatalf("caption = %#v", payload["caption"])
	}
	var source map[string]any
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatalf("unmarshal source: %v", err)
	}
	if !reflect.DeepEqual(payload["caption_entities"], source["caption_entities"]) {
		t.Fatalf("caption_entities changed: got %#v want %#v", payload["caption_entities"], source["caption_entities"])
	}
	if _, exists := payload["parse_mode"]; exists {
		t.Fatal("parse_mode must be removed when caption_entities are present")
	}
	if _, exists := payload["caption_parse_mode"]; exists {
		t.Fatal("caption_parse_mode must not be emitted")
	}
	if payload["show_caption_above_media"] != true || payload["has_spoiler"] != true {
		t.Fatalf("caption flags missing: %#v", payload)
	}
}

func TestBuildSendPayloadMapsMediaSpoilerOutputField(t *testing.T) {
	raw := json.RawMessage(`{
		"photo": [{"file_id":"photo-id","width":100,"height":100}],
		"caption": "spoiler",
		"has_media_spoiler": true
	}`)

	_, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("BuildSendPayload: %v", err)
	}
	if payload["has_spoiler"] != true {
		t.Fatalf("has_spoiler = %#v, want true; payload = %#v", payload["has_spoiler"], payload)
	}
	if _, exists := payload["has_media_spoiler"]; exists {
		t.Fatalf("output-only has_media_spoiler leaked into payload: %#v", payload)
	}
}

func TestBuildSendPayloadPreservesTextEntities(t *testing.T) {
	raw := json.RawMessage(`{
		"text": "formatted",
		"entities": [{"type":"bold","offset":0,"length":9}],
		"parse_mode": "HTML",
		"link_preview_options": {"is_disabled":true},
		"message_id": 1
	}`)
	method, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("BuildSendPayload: %v", err)
	}
	if method != "sendMessage" {
		t.Fatalf("method = %q", method)
	}
	if payload["text"] != "formatted" {
		t.Fatalf("text = %#v", payload["text"])
	}
	wantEntities := []any{map[string]any{"type": "bold", "offset": float64(0), "length": float64(9)}}
	if !reflect.DeepEqual(payload["entities"], wantEntities) {
		t.Fatalf("entities = %#v", payload["entities"])
	}
	if _, exists := payload["parse_mode"]; exists {
		t.Fatal("parse_mode must be removed when entities are present")
	}
	if _, exists := payload["link_preview_options"]; !exists {
		t.Fatal("link_preview_options missing")
	}
	if _, exists := payload["message_id"]; exists {
		t.Fatal("message_id leaked into text payload")
	}
}

func TestBuildSendPayloadRejectsMissingFileID(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "photo empty array", raw: `{"photo":[]}`},
		{name: "photo empty file ID", raw: `{"photo":[{"file_id":""}]}`},
		{name: "photo null", raw: `{"photo":null}`},
		{name: "video missing file ID", raw: `{"video":{}}`},
		{name: "video empty file ID", raw: `{"video":{"file_id":""}}`},
		{name: "video null with text fallback", raw: `{"video":null,"text":"must not be replayed as text"}`},
		{name: "document missing file ID", raw: `{"document":{}}`},
		{name: "document empty file ID", raw: `{"document":{"file_id":""}}`},
		{name: "document null", raw: `{"document":null}`},
		{name: "audio missing file ID", raw: `{"audio":{}}`},
		{name: "audio empty file ID", raw: `{"audio":{"file_id":""}}`},
		{name: "audio null", raw: `{"audio":null}`},
		{name: "voice missing file ID", raw: `{"voice":{}}`},
		{name: "voice empty file ID", raw: `{"voice":{"file_id":""}}`},
		{name: "voice null", raw: `{"voice":null}`},
		{name: "video note missing file ID", raw: `{"video_note":{}}`},
		{name: "video note empty file ID", raw: `{"video_note":{"file_id":""}}`},
		{name: "video note null", raw: `{"video_note":null}`},
		{name: "animation missing file ID", raw: `{"animation":{}}`},
		{name: "animation empty file ID", raw: `{"animation":{"file_id":""}}`},
		{name: "animation null", raw: `{"animation":null}`},
		{name: "sticker missing file ID", raw: `{"sticker":{}}`},
		{name: "sticker empty file ID", raw: `{"sticker":{"file_id":""}}`},
		{name: "sticker null", raw: `{"sticker":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, payload, err := BuildSendPayload(json.RawMessage(tt.raw), ReplayOptions{TargetChatID: 42})
			if err == nil || !strings.Contains(err.Error(), "file_id") {
				t.Fatalf("error = %v, want missing file_id diagnostic", err)
			}
			if method != "" || payload != nil {
				t.Fatalf("unexpected result on error: method=%q payload=%#v", method, payload)
			}
		})
	}
}

func TestBuildSendPayloadRejectsUnsupportedOrEmptyContent(t *testing.T) {
	for _, raw := range []string{
		`{}`,
		`{"caption":"caption only"}`,
	} {
		method, payload, err := BuildSendPayload(json.RawMessage(raw), ReplayOptions{TargetChatID: 42})
		if err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("raw %s: error = %v", raw, err)
		}
		if method == "sendMessage" || payload != nil {
			t.Fatalf("raw %s: unexpected fallback method=%q payload=%#v", raw, method, payload)
		}
	}
}

func TestBuildSendPayloadExoticContent(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		wantMethod     string
		assertPayload  func(t *testing.T, payload map[string]any)
	}{
		{
			name: "poll",
			raw: `{
				"message_id": 1,
				"business_connection_id": "biz-conn",
				"poll": {
					"id": "poll-id",
					"question": "Pick one?",
					"total_voter_count": 5,
					"options": [
						{"text": "A", "voter_count": 3},
						{"text": "B", "voter_count": 2}
					]
				}
			}`,
			wantMethod: "sendPoll",
			assertPayload: func(t *testing.T, payload map[string]any) {
				if payload["question"] != "Pick one?" {
					t.Fatalf("question = %#v", payload["question"])
				}
				if payload["is_closed"] != true {
					t.Fatalf("is_closed = %#v", payload["is_closed"])
				}
				if _, exists := payload["id"]; exists {
					t.Fatal("poll id leaked into payload")
				}
				if _, exists := payload["total_voter_count"]; exists {
					t.Fatal("total_voter_count leaked into payload")
				}
				options, ok := payload["options"].([]map[string]any)
				if !ok {
					t.Fatalf("options type = %T", payload["options"])
				}
				if len(options) != 2 {
					t.Fatalf("options len = %d", len(options))
				}
				if _, exists := options[0]["voter_count"]; exists {
					t.Fatal("voter_count leaked into poll option")
				}
			},
		},
		{
			name: "checklist notification poll fallback",
			raw: `{
				"checklist": {
					"title": "Todo",
					"tasks": [
						{"id": 1, "text": "Buy milk", "completed_by_user": {"id": 1}, "completion_date": 123},
						{"id": 2, "text": "Walk dog"}
					]
				}
			}`,
			wantMethod: "sendPoll",
			assertPayload: func(t *testing.T, payload map[string]any) {
				if payload["allows_multiple_answers"] != true || payload["is_closed"] != true {
					t.Fatalf("poll flags = %#v", payload)
				}
				question, _ := payload["question"].(string)
				if !strings.Contains(question, "Todo") || !strings.Contains(question, ChecklistOriginNotice) {
					t.Fatalf("question = %#v", payload["question"])
				}
				options, ok := payload["options"].([]map[string]any)
				if !ok {
					t.Fatalf("options type = %T", payload["options"])
				}
				if len(options) != 2 {
					t.Fatalf("options len = %d", len(options))
				}
				if _, exists := payload["business_connection_id"]; exists {
					t.Fatal("business_connection_id must not be in notification replay")
				}
			},
		},
		{
			name: "checklist notification text fallback single task",
			raw: `{
				"checklist": {
					"title": "Todo",
					"tasks": [{"id": 1, "text": "Buy milk"}]
				}
			}`,
			wantMethod: "sendMessage",
			assertPayload: func(t *testing.T, payload map[string]any) {
				text, _ := payload["text"].(string)
				if !strings.Contains(text, "Todo") || !strings.Contains(text, "Buy milk") {
					t.Fatalf("text = %#v", payload["text"])
				}
				if !strings.Contains(text, ChecklistOriginNotice) {
					t.Fatalf("text must mention checklist origin, got %#v", text)
				}
				if _, exists := payload["business_connection_id"]; exists {
					t.Fatal("business_connection_id must not be in notification replay")
				}
			},
		},
		{
			name: "rich message photo block",
			raw: `{
				"rich_message": {
					"blocks": [{
						"type": "photo",
						"photo": [
							{"file_id": "small", "file_size": 10},
							{"file_id": "photo-large", "file_size": 100}
						],
						"has_spoiler": true
					}]
				}
			}`,
			wantMethod: "sendRichMessage",
			assertPayload: func(t *testing.T, payload map[string]any) {
				rich, ok := payload["rich_message"].(map[string]any)
				if !ok {
					t.Fatalf("rich_message type = %T", payload["rich_message"])
				}
				blocks, ok := rich["blocks"].([]any)
				if !ok || len(blocks) == 0 {
					t.Fatalf("blocks = %#v", rich["blocks"])
				}
				block := blocks[0].(map[string]any)
				photo, ok := block["photo"].(map[string]any)
				if !ok {
					t.Fatalf("photo block media = %T (%#v)", block["photo"], block["photo"])
				}
				if photo["type"] != "photo" || photo["media"] != "photo-large" {
					t.Fatalf("photo InputMedia = %#v", photo)
				}
				if photo["has_spoiler"] != true {
					t.Fatalf("has_spoiler = %#v", photo["has_spoiler"])
				}
			},
		},
		{
			name: "rich message paragraph",
			raw: `{
				"rich_message": {
					"blocks": [
						{"type": "paragraph", "text": "Hello", "message_id": 99}
					],
					"is_rtl": false
				}
			}`,
			wantMethod: "sendRichMessage",
			assertPayload: func(t *testing.T, payload map[string]any) {
				rich, ok := payload["rich_message"].(map[string]any)
				if !ok {
					t.Fatalf("rich_message type = %T", payload["rich_message"])
				}
				blocks, ok := rich["blocks"].([]any)
				if !ok || len(blocks) == 0 {
					t.Fatalf("blocks = %#v", rich["blocks"])
				}
				block := blocks[0].(map[string]any)
				if block["text"] != "Hello" {
					t.Fatalf("block text = %#v", block["text"])
				}
				if _, exists := block["message_id"]; exists {
					t.Fatal("message_id leaked into input block")
				}
			},
		},
		{
			name:       "location",
			raw:        `{"location":{"latitude":55.75,"longitude":37.62,"live_period":900,"heading":90}}`,
			wantMethod: "sendLocation",
			assertPayload: func(t *testing.T, payload map[string]any) {
				if payload["latitude"] != float64(55.75) || payload["longitude"] != float64(37.62) {
					t.Fatalf("coordinates = %#v", payload)
				}
			},
		},
		{
			name: "venue",
			raw: `{"venue":{
				"location":{"latitude":1,"longitude":2},
				"title":"Place",
				"address":"Street 1",
				"google_place_id":"gp-id"
			}}`,
			wantMethod: "sendVenue",
			assertPayload: func(t *testing.T, payload map[string]any) {
				if payload["title"] != "Place" || payload["address"] != "Street 1" {
					t.Fatalf("venue fields = %#v", payload)
				}
			},
		},
		{
			name:       "contact",
			raw:        `{"contact":{"phone_number":"+1000","first_name":"Ann","last_name":"B","user_id":42,"vcard":"BEGIN:VCARD"}}`,
			wantMethod: "sendContact",
			assertPayload: func(t *testing.T, payload map[string]any) {
				if payload["phone_number"] != "+1000" || payload["first_name"] != "Ann" {
					t.Fatalf("contact fields = %#v", payload)
				}
				if _, exists := payload["user_id"]; exists {
					t.Fatal("user_id leaked into payload")
				}
			},
		},
		{
			name:       "dice",
			raw:        `{"dice":{"emoji":"🎲","value":4}}`,
			wantMethod: "sendDice",
			assertPayload: func(t *testing.T, payload map[string]any) {
				if payload["emoji"] != "🎲" {
					t.Fatalf("emoji = %#v", payload["emoji"])
				}
				if _, exists := payload["value"]; exists {
					t.Fatal("value leaked into payload")
				}
			},
		},
		{
			name:       "game",
			raw:        `{"game":{"title":"My Game","description":"fun"}}`,
			wantMethod: "sendGame",
			assertPayload: func(t *testing.T, payload map[string]any) {
				if payload["game_short_name"] != "My Game" {
					t.Fatalf("game_short_name = %#v", payload["game_short_name"])
				}
			},
		},
		{
			name: "location with text does not fallback to sendMessage",
			raw:  `{"location":{"latitude":1,"longitude":2},"text":"must not be replayed as text"}`,
			wantMethod: "sendLocation",
			assertPayload: func(t *testing.T, payload map[string]any) {
				if _, exists := payload["text"]; exists {
					t.Fatal("text leaked into location payload")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, payload, err := BuildSendPayload(json.RawMessage(tt.raw), ReplayOptions{
				TargetChatID: 42,
			})
			if err != nil {
				t.Fatalf("BuildSendPayload: %v", err)
			}
			if method != tt.wantMethod {
				t.Fatalf("method = %q, want %q", method, tt.wantMethod)
			}
			if payload["chat_id"] != int64(42) {
				t.Fatalf("chat_id = %#v", payload["chat_id"])
			}
			tt.assertPayload(t, payload)
		})
	}
}

func TestBuildSendPayloadNullPollKeyWithTextUsesSendMessage(t *testing.T) {
	raw := json.RawMessage(`{"poll":null,"text":"hello plain"}`)
	method, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("BuildSendPayload: %v", err)
	}
	if method != "sendMessage" {
		t.Fatalf("method = %q, want sendMessage", method)
	}
	if payload["text"] != "hello plain" {
		t.Fatalf("text = %#v", payload["text"])
	}
}

func TestBuildSendPayloadChecklistBusinessChat(t *testing.T) {
	raw := json.RawMessage(`{"checklist":{"title":"Todo","tasks":[{"id":1,"text":"Task"}]}}`)
	method, payload, err := BuildSendPayload(raw, ReplayOptions{
		TargetChatID:              42,
		IncludeBusinessConnection: true,
		BusinessConnectionID:      "injected-conn",
	})
	if err != nil {
		t.Fatalf("BuildSendPayload: %v", err)
	}
	if method != "sendChecklist" {
		t.Fatalf("method = %q, want sendChecklist", method)
	}
	if payload["business_connection_id"] != "injected-conn" {
		t.Fatalf("business_connection_id = %#v", payload["business_connection_id"])
	}
}

func TestBuildSendPayloadChecklistBusinessChatMissingConnectionErrors(t *testing.T) {
	raw := json.RawMessage(`{"checklist":{"title":"Todo","tasks":[{"id":1,"text":"Task"}]}}`)
	method, payload, err := BuildSendPayload(raw, ReplayOptions{
		TargetChatID:              42,
		IncludeBusinessConnection: true,
	})
	if err == nil || !strings.Contains(err.Error(), "business_connection_id") {
		t.Fatalf("error = %v, want business_connection_id diagnostic", err)
	}
	if method != "" || payload != nil {
		t.Fatalf("unexpected result on error: method=%q payload=%#v", method, payload)
	}
}

func TestBuildSendPayloadRichMessageWithoutBlocks(t *testing.T) {
	raw := json.RawMessage(`{"rich_message":{"is_rtl":false}}`)
	method, payload, err := BuildSendPayload(raw, ReplayOptions{
		TargetChatID: 42,
	})
	if err == nil || !strings.Contains(err.Error(), "rich_message blocks, html, or markdown are required") {
		t.Fatalf("error = %v, want rich_message content diagnostic", err)
	}
	if method != "" || payload != nil {
		t.Fatalf("unexpected result on error: method=%q payload=%#v", method, payload)
	}
}

func TestBuildSendPayloadRichMessageHTML(t *testing.T) {
	raw := json.RawMessage(`{"rich_message":{"html":"<b>Hello</b> rich"}}`)
	method, payload, err := BuildSendPayload(raw, ReplayOptions{
		TargetChatID: 42,
	})
	if err != nil {
		t.Fatalf("BuildSendPayload: %v", err)
	}
	if method != "sendRichMessage" {
		t.Fatalf("method = %q, want sendRichMessage", method)
	}
	rich, ok := payload["rich_message"].(map[string]any)
	if !ok || rich["html"] == nil {
		t.Fatalf("rich_message html missing: %#v", payload["rich_message"])
	}
}

func TestBuildSendPayloadRejectsServiceContent(t *testing.T) {
	for _, raw := range []string{
		`{"giveaway":{"chat_ids":[1],"winners_selection_date":1,"winner_count":1}}`,
		`{"giveaway_winners":{"chat":{"id":1},"giveaway_message_id":1,"winners_selection_date":1,"winner_count":1,"winners":[]}}`,
	} {
		method, payload, err := BuildSendPayload(json.RawMessage(raw), ReplayOptions{TargetChatID: 42})
		if err == nil || !strings.Contains(err.Error(), "service or unreplayable") {
			t.Fatalf("raw %s: error = %v", raw, err)
		}
		if method != "" || payload != nil {
			t.Fatalf("raw %s: unexpected result method=%q payload=%#v", raw, method, payload)
		}
	}
}

func TestBuildSendPayloadOmitsCaptionEntitiesWithoutCaption(t *testing.T) {
	_, payload, err := BuildSendPayload(json.RawMessage(`{
		"document":{"file_id":"document-id"},
		"caption_entities":[{"type":"bold","offset":0,"length":1}],
		"parse_mode":"HTML"
	}`), ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("BuildSendPayload: %v", err)
	}
	for _, key := range []string{"caption", "caption_entities", "parse_mode"} {
		if _, exists := payload[key]; exists {
			t.Fatalf("%s must be omitted without a caption: %#v", key, payload)
		}
	}
}

func TestIsPlainText(t *testing.T) {
	if !IsPlainText(json.RawMessage(`{"text":"hello"}`)) {
		t.Fatal("expected plain text")
	}
	if IsPlainText(json.RawMessage(`{"text":"hello","photo":[]}`)) {
		t.Fatal("photo message must not be plain text")
	}
	if !IsPlainText(json.RawMessage(`{"text":"hello","poll":null}`)) {
		t.Fatal("null poll key must not block plain text")
	}
	if IsPlainText(json.RawMessage(`{"text":"hello","rich_message":{"blocks":[{"type":"paragraph","text":"x"}]}}`)) {
		t.Fatal("rich_message must block plain text")
	}
}

func TestBuildAlbumPayloadOrdering(t *testing.T) {
	raws := []json.RawMessage{
		json.RawMessage(`{"message_id":3,"photo":[{"file_id":"c","file_unique_id":"u","width":1,"height":1}]}`),
		json.RawMessage(`{"message_id":1,"photo":[{"file_id":"a","file_unique_id":"u","width":1,"height":1}]}`),
		json.RawMessage(`{"message_id":2,"photo":[{"file_id":"b","file_unique_id":"u","width":1,"height":1}]}`),
	}
	_, payload, err := BuildAlbumPayload(raws, ReplayOptions{TargetChatID: 7})
	if err != nil {
		t.Fatalf("BuildAlbumPayload: %v", err)
	}
	media, ok := payload["media"].([]map[string]any)
	if !ok {
		t.Fatalf("media type = %T", payload["media"])
	}
	if len(media) != 3 {
		t.Fatalf("media len = %d", len(media))
	}
	if media[0]["media"] != "a" || media[1]["media"] != "b" || media[2]["media"] != "c" {
		t.Fatalf("unexpected order: %#v", media)
	}
}

func TestBuildAlbumPayloadPreservesCaptionEntities(t *testing.T) {
	raws := []json.RawMessage{
		json.RawMessage(`{"message_id":2,"video":{"file_id":"b"},"caption":"second","caption_entities":[{"type":"bold","offset":0,"length":6}]}`),
		json.RawMessage(`{"message_id":1,"photo":[{"file_id":"a"}],"caption":"first","caption_entities":[{"type":"italic","offset":0,"length":5}]}`),
	}
	_, payload, err := BuildAlbumPayload(raws, ReplayOptions{TargetChatID: 7})
	if err != nil {
		t.Fatalf("BuildAlbumPayload: %v", err)
	}
	media := payload["media"].([]map[string]any)
	if media[0]["media"] != "a" || media[1]["media"] != "b" {
		t.Fatalf("album order/file IDs changed: %#v", media)
	}
	firstEntities := media[0]["caption_entities"].([]any)
	if firstEntities[0].(map[string]any)["type"] != "italic" {
		t.Fatalf("first caption entities changed: %#v", media[0])
	}
	secondEntities := media[1]["caption_entities"].([]any)
	if secondEntities[0].(map[string]any)["type"] != "bold" {
		t.Fatalf("second caption entities changed: %#v", media[1])
	}
}

func TestLoadStoredMessageRawAndLegacyFormats(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	legacyPayload := []byte(`{"message_id":5,"text":"legacy"}`)
	rawPayload := []byte(`{"message_id":6,"text":"raw","extra":1}`)

	encLegacy, err := encryptForTest(legacyPayload, key)
	if err != nil {
		t.Fatalf("encrypt legacy: %v", err)
	}
	encRaw, err := encryptForTest(rawPayload, key)
	if err != nil {
		t.Fatalf("encrypt raw: %v", err)
	}

	storedRaw, legacyMsg, loadErr := LoadStoredMessage(MetadataFormatRawAPIv1, encRaw, key)
	if loadErr != nil {
		t.Fatalf("load raw format: %v", loadErr)
	}
	if legacyMsg != nil {
		t.Fatal("expected nil legacy message for raw format")
	}
	if string(storedRaw.Payload) != string(rawPayload) {
		t.Fatalf("raw payload = %s", storedRaw.Payload)
	}

	storedLegacy, legacyParsed, loadErr := LoadStoredMessage(MetadataFormatLegacyStruct, encLegacy, key)
	if loadErr != nil {
		t.Fatalf("load legacy format: %v", loadErr)
	}
	if legacyParsed == nil || legacyParsed.Text != "legacy" {
		t.Fatalf("legacy parsed = %#v", legacyParsed)
	}
	if string(storedLegacy.Payload) != string(legacyPayload) {
		t.Fatalf("legacy payload = %s", storedLegacy.Payload)
	}
}

func TestMarshalBusinessMessagePreservesUpdateRaw(t *testing.T) {
	webhook := []byte(`{"business_message":{"message_id":1,"text":"hi","vendor_field":true}}`)
	var update struct {
		BusinessMessage *tele.Message `json:"business_message"`
	}
	if err := json.Unmarshal(webhook, &update); err != nil {
		t.Fatalf("unmarshal update: %v", err)
	}
	out, err := MarshalBusinessMessage(update.BusinessMessage)
	if err != nil {
		t.Fatalf("marshal business message: %v", err)
	}
	if string(out) != `{"message_id":1,"text":"hi","vendor_field":true}` {
		t.Fatalf("marshal output = %s", out)
	}
}

func TestLegacyStructJSONStillParses(t *testing.T) {
	raw := []byte(`{"message_id":5,"text":"legacy","unknown":1}`)
	var msg tele.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Text != "legacy" {
		t.Fatalf("text = %q", msg.Text)
	}
	out, err := json.Marshal(&msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != string(raw) {
		t.Fatalf("got %s want %s", out, raw)
	}
}

func encryptForTest(plaintext, key []byte) ([]byte, error) {
	encrypted, errInfo := utils.Encrypt(plaintext, key)
	if e.IsNonNil(errInfo) {
		return nil, errInfo
	}
	return encrypted, nil
}
