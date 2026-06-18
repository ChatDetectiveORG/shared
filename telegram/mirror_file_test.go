package telegram

import (
	"errors"
	"testing"

	tele "gopkg.in/telebot.v4"
)

type stubFileResolver struct {
	ids map[string]bool
}

func (s stubFileResolver) FileByID(fileID string) (tele.File, error) {
	if s.ids[fileID] {
		return tele.File{FileID: fileID, FilePath: "path"}, nil
	}
	return tele.File{}, errors.New("wrong file_id")
}

func TestResolveBuilderFileKeepsFallback(t *testing.T) {
	file := resolveBuilderFile("cloud-id", "static/photo.png")
	if file.FileID != "cloud-id" {
		t.Fatalf("file id = %q", file.FileID)
	}
	if file.FileLocal != "static/photo.png" {
		t.Fatalf("file local = %q", file.FileLocal)
	}
}

func TestInvalidateStaleFileIDClearsBrokenID(t *testing.T) {
	file := tele.File{FileID: "broken", FileLocal: "static/photo.png"}
	InvalidateStaleFileID(stubFileResolver{ids: map[string]bool{"valid": true}}, &file)
	if file.FileID != "" {
		t.Fatalf("expected stale file id to be cleared, got %q", file.FileID)
	}
}

func TestInvalidateStaleFileIDKeepsValidID(t *testing.T) {
	file := tele.File{FileID: "valid", FileLocal: "static/photo.png"}
	InvalidateStaleFileID(stubFileResolver{ids: map[string]bool{"valid": true}}, &file)
	if file.FileID != "valid" {
		t.Fatalf("expected valid file id to stay, got %q", file.FileID)
	}
}

func TestForceLocalFileFallback(t *testing.T) {
	msg := &tele.Message{
		Photo: &tele.Photo{File: tele.File{FileID: "id", FileLocal: "static/photo.png"}},
	}
	if !ForceLocalFileFallback(msg) {
		t.Fatal("expected fallback to be applied")
	}
	if msg.Photo.File.FileID != "" {
		t.Fatalf("file id = %q, want empty", msg.Photo.File.FileID)
	}
}

func TestExtractSentFileID(t *testing.T) {
	msg := &tele.Message{
		Animation: &tele.Animation{File: tele.File{FileID: "anim-id"}},
	}
	if got := ExtractSentFileID(msg); got != "anim-id" {
		t.Fatalf("file id = %q", got)
	}
}
