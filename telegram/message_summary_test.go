package telegram

import (
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v4"
)

func TestBuildMessageSummaryNilMessage(t *testing.T) {
	text, opts := BuildMessageSummary(nil)

	if text != "" {
		t.Fatalf("expected empty text, got %q", text)
	}
	if opts == nil {
		t.Fatal("expected non-nil send options")
	}
	if !opts.DisableWebPagePreview {
		t.Fatal("expected web page previews to be disabled")
	}
	if len(opts.Entities) != 0 {
		t.Fatalf("expected no entities, got %d", len(opts.Entities))
	}
}

func TestBuildMessageSummaryInternalReplyWithQuote(t *testing.T) {
	msg := &tele.Message{
		ReplyTo: &tele.Message{
			Sender: &tele.User{Username: "alice"},
			Chat:   &tele.Chat{Type: tele.ChatPrivate},
			Text:   "Исходное сообщение",
		},
		Quote: &tele.TextQuote{
			Text: "  фрагмент   цитаты ",
		},
	}

	text, opts := BuildMessageSummary(msg)

	expected := "Это ответ цитатой на сообщение пользователя @alice.\nЦитата: «фрагмент цитаты»."
	if text != expected {
		t.Fatalf("unexpected summary text:\nwant: %q\ngot:  %q", expected, text)
	}
	if opts == nil {
		t.Fatal("expected non-nil send options")
	}
	if !opts.DisableWebPagePreview {
		t.Fatal("expected web page previews to be disabled")
	}
	if len(opts.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(opts.Entities))
	}

	assertTextLinkEntity(t, text, opts.Entities[0], "цитатой", 0)
	assertTextLinkEntity(t, text, opts.Entities[1], "сообщение", 0)
}

func TestBuildMessageSummaryExternalReplyFromChannel(t *testing.T) {
	msg := &tele.Message{
		ExternalReply: &tele.ExternalReply{
			Video: &tele.Video{},
			Chat: &tele.Chat{
				Type:     tele.ChatChannel,
				Username: "kot_meme",
			},
		},
	}

	text, opts := BuildMessageSummary(msg)

	expected := "Это ответ на сообщение с видео из канала @kot_meme."
	if text != expected {
		t.Fatalf("unexpected summary text:\nwant: %q\ngot:  %q", expected, text)
	}
	if len(opts.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(opts.Entities))
	}

	assertTextLinkEntity(t, text, opts.Entities[0], "сообщение", 0)
}

func TestBuildMessageSummaryForwardedMessage(t *testing.T) {
	unix := time.Date(2026, time.April, 12, 19, 4, 0, 0, time.FixedZone("GMT+3", 3*60*60)).Unix()
	msg := &tele.Message{
		OriginalChat:      &tele.Chat{Type: tele.ChatChannel, Username: "kot_meme"},
		OriginalUnixtime:  int(unix),
		OriginalSignature: "Юлик",
		AutomaticForward:  true,
	}

	text, opts := BuildMessageSummary(msg)

	expected := "Сообщение переслано из канала @kot_meme, дата отправки оригинала (GMT+3): 12.04.2026 19:04, подпись автора: Юлик, это автоматическая пересылка."
	if text != expected {
		t.Fatalf("unexpected summary text:\nwant: %q\ngot:  %q", expected, text)
	}
	if len(opts.Entities) != 0 {
		t.Fatalf("expected no entities, got %d", len(opts.Entities))
	}
}

func TestBuildMessageSummaryStoryViaBotAndState(t *testing.T) {
	msg := &tele.Message{
		ReplyToStory: &tele.Story{
			Poster: &tele.Chat{
				Type:     tele.ChatChannel,
				Username: "stories",
			},
		},
		Via:         &tele.User{Username: "helper_bot"},
		Protected:   true,
		FromOffline: true,
	}

	text, opts := BuildMessageSummary(msg)

	expected := "Это ответ на историю из канала @stories.\nСообщение отправлено через бота @helper_bot.\nСообщение защищено от пересылки. Сообщение было отправлено офлайн."
	if text != expected {
		t.Fatalf("unexpected summary text:\nwant: %q\ngot:  %q", expected, text)
	}
	if len(opts.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(opts.Entities))
	}

	assertTextLinkEntity(t, text, opts.Entities[0], "историю", 0)
}

func TestBuildMessageSummaryTruncatesLongQuote(t *testing.T) {
	msg := &tele.Message{
		Quote: &tele.TextQuote{
			Text: strings.Repeat("а", 5000),
		},
	}

	text, opts := BuildMessageSummary(msg)

	if telegramTextLen(text) > maxSummaryTextLen {
		t.Fatalf("expected summary length <= %d, got %d", maxSummaryTextLen, telegramTextLen(text))
	}
	if !strings.Contains(text, "Это ответ цитатой на сообщение.") {
		t.Fatalf("expected reply intro in summary, got %q", text)
	}
	if !strings.Contains(text, "...».") {
		t.Fatalf("expected truncated quote suffix, got %q", text)
	}
	if len(opts.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(opts.Entities))
	}

	assertTextLinkEntity(t, text, opts.Entities[0], "цитатой", 0)
	assertTextLinkEntity(t, text, opts.Entities[1], "сообщение", 0)
}

func assertTextLinkEntity(t *testing.T, text string, entity tele.MessageEntity, target string, occurrence int) {
	t.Helper()

	if entity.Type != tele.EntityTextLink {
		t.Fatalf("expected text_link entity, got %q", entity.Type)
	}
	if entity.URL != summaryLinkURL {
		t.Fatalf("expected entity URL %q, got %q", summaryLinkURL, entity.URL)
	}

	idx := nthIndex(text, target, occurrence)
	if idx < 0 {
		t.Fatalf("substring %q occurrence %d not found in %q", target, occurrence, text)
	}

	expectedOffset := telegramTextLen(text[:idx])
	expectedLength := telegramTextLen(target)

	if entity.Offset != expectedOffset {
		t.Fatalf("expected offset %d for %q, got %d", expectedOffset, target, entity.Offset)
	}
	if entity.Length != expectedLength {
		t.Fatalf("expected length %d for %q, got %d", expectedLength, target, entity.Length)
	}
}

func nthIndex(text, target string, occurrence int) int {
	start := 0
	for i := 0; i <= occurrence; i++ {
		idx := strings.Index(text[start:], target)
		if idx < 0 {
			return -1
		}
		if i == occurrence {
			return start + idx
		}
		start += idx + len(target)
	}
	return -1
}
