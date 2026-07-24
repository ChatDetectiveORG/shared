package rawmessage

import (
	"encoding/json"
	"testing"
)

func TestConvertRichBlockPhotoToInputMedia(t *testing.T) {
	raw := json.RawMessage(`{
		"rich_message": {
			"blocks": [{
				"type": "photo",
				"photo": [{"file_id": "pid"}]
			}]
		}
	}`)
	_, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 1})
	if err != nil {
		t.Fatal(err)
	}
	assertInputMediaBlock(t, payload, "photo", "photo", "pid")
}

func TestConvertRichBlockVideoToInputMedia(t *testing.T) {
	raw := json.RawMessage(`{
		"rich_message": {
			"blocks": [{
				"type": "video",
				"video": {"file_id": "vid", "width": 100},
				"has_spoiler": true
			}]
		}
	}`)
	_, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 1})
	if err != nil {
		t.Fatal(err)
	}
	assertInputMediaBlock(t, payload, "video", "video", "vid")
}

func TestConvertRichBlockAnimationAudioVoiceToInputMedia(t *testing.T) {
	tests := []struct {
		blockType string
		key       string
		fileID    string
	}{
		{blockType: "animation", key: "animation", fileID: "anim-id"},
		{blockType: "audio", key: "audio", fileID: "audio-id"},
		{blockType: "voice_note", key: "voice_note", fileID: "voice-id"},
	}
	for _, tt := range tests {
		t.Run(tt.blockType, func(t *testing.T) {
			raw := json.RawMessage(`{
				"rich_message": {
					"blocks": [{
						"type": "` + tt.blockType + `",
						"` + tt.key + `": {"file_id": "` + tt.fileID + `"}
					}]
				}
			}`)
			_, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 1})
			if err != nil {
				t.Fatal(err)
			}
			assertInputMediaBlock(t, payload, tt.blockType, tt.key, tt.fileID)
		})
	}
}

func TestConvertRichBlockCollageNestedMedia(t *testing.T) {
	raw := json.RawMessage(`{
		"rich_message": {
			"blocks": [{
				"type": "collage",
				"blocks": [
					{"type": "photo", "photo": [{"file_id": "p1"}]},
					{"type": "video", "video": {"file_id": "v1"}}
				]
			}]
		}
	}`)
	_, payload, err := BuildSendPayload(raw, ReplayOptions{TargetChatID: 1})
	if err != nil {
		t.Fatal(err)
	}
	rich := payload["rich_message"].(map[string]any)
	collage := rich["blocks"].([]any)[0].(map[string]any)
	nested := collage["blocks"].([]any)
	if len(nested) != 2 {
		t.Fatalf("nested len = %d", len(nested))
	}
	photoMedia := nested[0].(map[string]any)["photo"].(map[string]any)
	if photoMedia["media"] != "p1" {
		t.Fatalf("photo media = %#v", photoMedia)
	}
	videoMedia := nested[1].(map[string]any)["video"].(map[string]any)
	if videoMedia["media"] != "v1" {
		t.Fatalf("video media = %#v", videoMedia)
	}
}

func assertInputMediaBlock(t *testing.T, payload map[string]any, blockType, mediaKey, wantFileID string) {
	t.Helper()
	rich, ok := payload["rich_message"].(map[string]any)
	if !ok {
		t.Fatalf("rich_message = %T", payload["rich_message"])
	}
	block := rich["blocks"].([]any)[0].(map[string]any)
	if block["type"] != blockType {
		t.Fatalf("type = %q", block["type"])
	}
	media, ok := block[mediaKey].(map[string]any)
	if !ok {
		t.Fatalf("%s = %T (%#v)", mediaKey, block[mediaKey], block[mediaKey])
	}
	if media["type"] != blockType {
		t.Fatalf("media.type = %#v", media["type"])
	}
	if media["media"] != wantFileID {
		t.Fatalf("media.media = %#v", media["media"])
	}
}
