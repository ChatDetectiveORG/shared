package rawmessage

import (
	"testing"

	"github.com/ChatDetectiveORG/shared/utils"
)

func TestJoinFormattedOffsetsCustomEmoji(t *testing.T) {
	header := FormattedText{Text: "Header"}
	title := FormattedText{
		Text: "🔥",
		Entities: []any{
			map[string]any{
				"type":            "custom_emoji",
				"offset":          0,
				"length":          2,
				"custom_emoji_id": "e1",
			},
		},
	}

	combined := JoinFormatted("\n", header, title)
	if len(combined.Entities) != 1 {
		t.Fatalf("entities len = %d", len(combined.Entities))
	}
	wantOffset := utils.TgLen("Header\n")
	ent := combined.Entities[0].(map[string]any)
	if intFromAny(ent["offset"]) != wantOffset {
		t.Fatalf("offset = %v, want %d", ent["offset"], wantOffset)
	}
}

func TestWithLiteralPrefixOffsetsEntities(t *testing.T) {
	task := FormattedText{
		Text: "🔥",
		Entities: []any{
			map[string]any{
				"type":            "custom_emoji",
				"offset":          0,
				"length":          2,
				"custom_emoji_id": "task",
			},
		},
	}
	line := task.WithLiteralPrefix("- ")
	if intFromAny(line.Entities[0].(map[string]any)["offset"]) != utils.TgLen("- ") {
		t.Fatalf("offset = %#v", line.Entities[0])
	}
}
