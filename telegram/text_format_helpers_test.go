package telegram

import "testing"

func TestTextFormatWithUserMention(t *testing.T) {
	f := (&TextFormat{Type: "link"}).WithUserMention(99)
	if f.URL != "tg://user?id=99" {
		t.Fatalf("URL = %q, want tg://user?id=99", f.URL)
	}
}

func TestTextFormatWithCustomEmojiID(t *testing.T) {
	f := (&TextFormat{Type: "link"}).WithCustomEmojiID("emoji-1")
	if f.URL != "tg://emoji?id=emoji-1" {
		t.Fatalf("URL = %q, want tg://emoji?id=emoji-1", f.URL)
	}
}
