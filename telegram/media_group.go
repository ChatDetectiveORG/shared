package telegram

import (
	"sort"

	tele "gopkg.in/telebot.v4"
)

type MediaGroup struct {
	Chat     *tele.Chat      `json:"chat,omitempty"`
	Messages []*tele.Message `json:"messages"`
	Silent   bool            `json:"silent,omitempty"`
}

type albumInputtable struct {
	media      tele.Inputtable
	inputMedia tele.InputMedia
}

func (a *albumInputtable) MediaType() string {
	return a.media.MediaType()
}

func (a *albumInputtable) MediaFile() *tele.File {
	return a.media.MediaFile()
}

func (a *albumInputtable) InputMedia() tele.InputMedia {
	return a.inputMedia
}

// BuildMediaGroup normalizes Telegram album messages into a serializable
// project payload that can be published and later sent as a native Telegram
// media group.
func BuildMediaGroup(msgs []*tele.Message) (*MediaGroup, bool) {
	if len(msgs) == 0 {
		return nil, false
	}

	// Sort by MessageID to preserve order
	sorted := make([]*tele.Message, len(msgs))
	copy(sorted, msgs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	mediaGroup := &MediaGroup{
		Messages: make([]*tele.Message, 0, len(sorted)),
	}
	var captionSource *tele.Message

	for _, msg := range sorted {
		mediaMessage := buildMediaGroupMessage(msg)
		if mediaMessage == nil {
			continue
		}
		mediaGroup.Messages = append(mediaGroup.Messages, mediaMessage)
		if captionSource == nil && (msg.Caption != "" || msg.Text != "") {
			captionSource = msg
		}
	}

	if len(mediaGroup.Messages) == 0 {
		return nil, false
	}

	if captionSource != nil {
		caption := captionSource.Caption
		if caption == "" {
			caption = captionSource.Text
		}
		setMediaGroupCaption(mediaGroup.Messages[0], caption)
		HideSendOptsIntoMessage(mediaGroup.Messages[0], getSendOptions(captionSource))
	}

	return mediaGroup, true
}

func (mg *MediaGroup) ToAlbum() (tele.Album, bool) {
	if mg == nil || len(mg.Messages) == 0 {
		return nil, false
	}

	album := make(tele.Album, 0, len(mg.Messages))
	for _, msg := range mg.Messages {
		item := ExtractInputtable(msg)
		if item == nil {
			return nil, false
		}

		inputMedia := item.InputMedia()
		inputMedia.Entities = messageCaptionEntities(msg)

		album = append(album, &albumInputtable{
			media:      item,
			inputMedia: inputMedia,
		})
	}

	return album, len(album) > 0
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
			Animation: copyAnimation(msg.Animation, "", msg.CaptionAbove, msg.HasMediaSpoiler),
		}
	default:
		return nil
	}
}

// ExtractInputtable extracts album-compatible media from a message.
func ExtractInputtable(msg *tele.Message) tele.Inputtable {
	if msg == nil {
		return nil
	}

	switch {
	case msg.Photo != nil:
		return copyPhoto(msg.Photo, msg.Caption, msg.CaptionAbove, msg.HasMediaSpoiler)
	case msg.Video != nil:
		return copyVideo(msg.Video, msg.Caption, msg.CaptionAbove, msg.HasMediaSpoiler)
	case msg.Document != nil:
		return copyDocument(msg.Document, msg.Caption)
	case msg.Audio != nil:
		return copyAudio(msg.Audio, msg.Caption)
	case msg.Animation != nil:
		return copyAnimation(msg.Animation, msg.Caption, msg.CaptionAbove, msg.HasMediaSpoiler)
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

func messageCaptionEntities(msg *tele.Message) tele.Entities {
	if msg == nil {
		return nil
	}
	if len(msg.CaptionEntities) > 0 {
		return msg.CaptionEntities
	}
	if len(msg.Entities) > 0 {
		return msg.Entities
	}
	return nil
}
