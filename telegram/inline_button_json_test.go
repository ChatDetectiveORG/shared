package telegram

import (
	"encoding/json"
	"testing"

	tele "gopkg.in/telebot.v4"
)

// Regression: message-sender must use the same telebot fork as command-handler. Upstream
// InlineButton has no icon_custom_emoji_id field; json.Unmarshal would drop it and Telegram
// would never receive the icon. See go.mod replace: ../forks/telebot
func TestJSONUnmarshalInlineButtonPreservesIconCustomEmojiID(t *testing.T) {
	const raw = `{
		"text": "Label",
		"callback_data": "x",
		"icon_custom_emoji_id": "5411197345968701560"
	}`
	var b tele.InlineButton
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if b.IconCustomEmojiID != "5411197345968701560" {
		t.Fatalf("IconCustomEmojiID: got %q, want 5411197345968701560 (wrong telebot build?)", b.IconCustomEmojiID)
	}
}
