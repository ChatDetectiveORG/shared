package telegram

import (
	"strings"

	tele "gopkg.in/telebot.v4"
)

// MirrorFileAsset describes a static media asset that may need per-mirror file_id caching.
type MirrorFileAsset struct {
	PrimaryFileID string
	FallbackPath  string
	MimeType      string
	MirrorFileKey string
}

// FileResolver can validate Telegram file_id values via getFile.
type FileResolver interface {
	FileByID(fileID string) (tele.File, error)
}

// ExtractSentFileID returns the Telegram file_id from a successfully sent media message.
func ExtractSentFileID(msg *tele.Message) string {
	if msg == nil {
		return ""
	}

	switch {
	case msg.Photo != nil:
		return msg.Photo.FileID
	case msg.Video != nil:
		return msg.Video.FileID
	case msg.Animation != nil:
		return msg.Animation.FileID
	case msg.Audio != nil:
		return msg.Audio.FileID
	case msg.Voice != nil:
		return msg.Voice.FileID
	case msg.Document != nil:
		return msg.Document.FileID
	case msg.VideoNote != nil:
		return msg.VideoNote.FileID
	case msg.Sticker != nil:
		return msg.Sticker.FileID
	default:
		return ""
	}
}

func collectMessageFiles(msg *tele.Message) []*tele.File {
	if msg == nil {
		return nil
	}

	var files []*tele.File

	switch {
	case msg.Photo != nil:
		files = append(files, &msg.Photo.File)
	case msg.Video != nil:
		files = append(files, &msg.Video.File)
	case msg.Animation != nil:
		files = append(files, &msg.Animation.File)
	case msg.Audio != nil:
		files = append(files, &msg.Audio.File)
	case msg.Voice != nil:
		files = append(files, &msg.Voice.File)
	case msg.Document != nil:
		files = append(files, &msg.Document.File)
	case msg.VideoNote != nil:
		files = append(files, &msg.VideoNote.File)
	case msg.Sticker != nil:
		files = append(files, &msg.Sticker.File)
	}

	return files
}

// InvalidateStaleFileID pings Telegram getFile and clears FileID when it is no longer valid.
// When FileLocal is present the sender can transparently re-upload from disk.
func InvalidateStaleFileID(resolver FileResolver, file *tele.File) {
	if resolver == nil || file == nil || file.FileID == "" {
		return
	}

	if _, err := resolver.FileByID(file.FileID); err == nil {
		return
	}

	file.FileID = ""
}

// PrepareOutgoingMessageFiles validates all file_id references before send.
func PrepareOutgoingMessageFiles(resolver FileResolver, msg *tele.Message) {
	for _, file := range collectMessageFiles(msg) {
		InvalidateStaleFileID(resolver, file)
	}
}

// PrepareOutgoingAlbumFiles validates all file_id references in an album payload.
func PrepareOutgoingAlbumFiles(resolver FileResolver, album *MediaGroup) {
	if album == nil {
		return
	}

	for _, msg := range album.Messages {
		PrepareOutgoingMessageFiles(resolver, msg)
	}
}

// ForceLocalFileFallback clears cloud file_id values when a local fallback path is available.
func ForceLocalFileFallback(msg *tele.Message) bool {
	changed := false
	for _, file := range collectMessageFiles(msg) {
		if file == nil || file.FileID == "" || file.FileLocal == "" {
			continue
		}
		file.FileID = ""
		changed = true
	}
	return changed
}

// IsInvalidFileIDError reports Telegram errors that indicate a stale or wrong file_id.
func IsInvalidFileIDError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "wrong file_id") ||
		strings.Contains(msg, "wrong file_identifier") ||
		strings.Contains(msg, "file_id is invalid")
}
