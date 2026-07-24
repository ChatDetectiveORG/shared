package notification

import (
	"strings"
	"testing"

	"github.com/ChatDetectiveORG/shared/telegram"
	tele "gopkg.in/telebot.v4"
)

func TestComputeDiffBoldEntitiesMarksChangedWords(t *testing.T) {
	oldDiff, newDiff := computeDiffBoldEntities("hello world", "hello brave world")
	if len(newDiff) == 0 {
		t.Fatalf("expected diff entities on new text, got old=%#v new=%#v", oldDiff, newDiff)
	}
	if newDiff[0].Type != tele.EntityBold {
		t.Fatalf("expected bold entity on new text, got %#v", newDiff)
	}

	oldDiff2, newDiff2 := computeDiffBoldEntities("hello world", "hello there")
	if len(oldDiff2) == 0 || len(newDiff2) == 0 {
		t.Fatalf("expected diff on both sides for replacement, old=%#v new=%#v", oldDiff2, newDiff2)
	}
}

func TestDeleteNoticeIncludesForwardSummary(t *testing.T) {
	forwarded := &tele.Message{
		OriginalChat:     &tele.Chat{Type: tele.ChatChannel, Username: "kot_meme"},
		OriginalUnixtime: 1_746_470_640,
		AutomaticForward: true,
	}
	summary, _ := telegram.BuildMessageSummary(forwarded)
	prefix := "Пользователь Bob удалил сообщение!\n"
	noticeText := prefix + summary
	if !strings.Contains(noticeText, "Сообщение переслано из канала @kot_meme") {
		t.Fatalf("expected forward summary in notice, got %q", noticeText)
	}
}
