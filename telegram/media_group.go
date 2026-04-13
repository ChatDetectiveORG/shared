package telegram

import (
	"sort"

	tele "gopkg.in/telebot.v4"
)

// BuildMediaGroup normalizes Telegram album messages into project-friendly
// tele.Message payloads that can be serialized and sent via telegram.message.send.
// Messages should already belong to the same media_group_id.
//
// Telegram allows a shared caption for an album only on its first element, so
// this helper moves the first found caption/text and its send options to the
// first supported media message in the group.
//
// Supported media types: Photo, Video, Document, Audio, Animation.
// Voice, VideoNote, Sticker are not valid album items and are skipped.
func BuildMediaGroup(msgs []*tele.Message) ([]*tele.Message, bool) {
	if len(msgs) == 0 {
		return nil, false
	}

	// Sort by MessageID to preserve order
	sorted := make([]*tele.Message, len(msgs))
	copy(sorted, msgs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	mediaGroup := make([]*tele.Message, 0, len(sorted))
	var captionSource *tele.Message

	for _, msg := range sorted {
		mediaMessage := buildMediaGroupMessage(msg)
		if mediaMessage == nil {
			continue
		}
		mediaGroup = append(mediaGroup, mediaMessage)
		if captionSource == nil && (msg.Caption != "" || msg.Text != "") {
			captionSource = msg
		}
	}

	if len(mediaGroup) == 0 {
		return nil, false
	}

	if captionSource != nil {
		caption := captionSource.Caption
		if caption == "" {
			caption = captionSource.Text
		}
		setMediaGroupCaption(mediaGroup[0], caption)
		HideSendOptsIntoMessage(mediaGroup[0], getSendOptions(captionSource))
	}

	return mediaGroup, true
}

func buildMediaGroupMessage(msg *tele.Message) *tele.Message {
	if msg == nil {
		return nil
	}

	switch {
	case msg.Photo != nil:
		return &tele.Message{
			Photo: copyPhoto(msg.Photo, "", msg.CaptionAbove, msg.HasMediaSpoiler),
		}
	case msg.Video != nil:
		return &tele.Message{
			Video: copyVideo(msg.Video, "", msg.CaptionAbove, msg.HasMediaSpoiler),
		}
	case msg.Document != nil:
		return &tele.Message{
			Document: copyDocument(msg.Document, ""),
		}
	case msg.Audio != nil:
		return &tele.Message{
			Audio: copyAudio(msg.Audio, ""),
		}
	case msg.Animation != nil:
		return &tele.Message{
			Animation: copyAnimation(msg.Animation, "", msg.HasMediaSpoiler),
		}
	default:
		return nil
	}
}

func setMediaGroupCaption(msg *tele.Message, caption string) {
	if msg == nil {
		return
	}

	msg.Caption = caption

	switch {
	case msg.Photo != nil:
		msg.Photo.Caption = caption
	case msg.Video != nil:
		msg.Video.Caption = caption
	case msg.Document != nil:
		msg.Document.Caption = caption
	case msg.Audio != nil:
		msg.Audio.Caption = caption
	case msg.Animation != nil:
		msg.Animation.Caption = caption
	}
}
