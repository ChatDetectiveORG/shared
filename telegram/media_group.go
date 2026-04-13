package telegram

import (
	"sort"

	tele "gopkg.in/telebot.v4"
)

// BuildMediaGroup builds an Album from messages that share the same AlbumID.
// Messages should be pre-filtered to have the same media_group_id.
// Returns (album, caption, ok). Caption is taken from the first message.
// ok is false if the group is empty or contains unsupported media types.
//
// Supported media types for albums: Photo, Video, Document, Audio, Animation.
// Voice, VideoNote, Sticker are NOT supported in media groups by Telegram API.
func BuildMediaGroup(msgs []*tele.Message) (tele.Album, *tele.SendOptions, bool) {
	if len(msgs) == 0 {
		return nil, nil, false
	}

	// Sort by MessageID to preserve order
	sorted := make([]*tele.Message, len(msgs))
	copy(sorted, msgs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	var album tele.Album
	var caption string
	var sendOptions *tele.SendOptions

	for _, msg := range sorted {
		item := extractInputtable(msg)
		if item == nil {
			continue
		}
		album = append(album, item)
		if msg.Caption != "" || msg.Text != "" {
			caption = msg.Caption
			sendOptions = getSendOptions(msg)
		}
	}

	if len(album) == 0 {
		return nil, nil,false
	}

	album.SetCaption(caption)

	return album, sendOptions, true
}

// extractInputtable extracts Inputtable media from a message.
// Returns nil for unsupported types (Voice, VideoNote, Sticker, etc.)
func extractInputtable(msg *tele.Message) tele.Inputtable {
	if msg == nil {
		return nil
	}

	switch {
	case msg.Photo != nil:
		return copyPhoto(msg.Photo, msg.Caption,  msg.CaptionAbove, msg.HasMediaSpoiler)
	case msg.Video != nil:
		return copyVideo(msg.Video, msg.Caption,  msg.CaptionAbove, msg.HasMediaSpoiler)
	case msg.Document != nil:
		return copyDocument(msg.Document, msg.Caption)
	case msg.Audio != nil:
		return copyAudio(msg.Audio, msg.Caption)
	case msg.Animation != nil:
		return copyAnimation(msg.Animation, msg.Caption,  msg.HasMediaSpoiler)
	default:
		return nil
	}
}
