package telegram

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"
)

// TgMessageToSendable converts a tele.Message to a sendable object (string or tele.Sendable).
// The result can be passed to bot.Send() or bot.SendAlbum() for media groups.
//
// Reply/reply-to information is NOT preserved (message is sent to a different chat).
// Supports: text, photo, video, document, audio, voice, video_note, sticker, animation,
// location, venue, contact, dice, poll, invoice, game, and service messages (as text).
//
// For text messages with entities (formatting), pass &tele.SendOptions{Entities: msg.Entities}
// as additional opts to bot.Send() to preserve bold, italic, links, etc.
func TgMessageToSendable(msg *tele.Message) (interface{}, bool) {
	if msg == nil {
		return nil, false
	}

	// For media groups: prefer BuildMediaGroup when you have all messages.
	// Here we still return the single media item as fallback.

	// Content types (order matters - check media before text for messages with both)
	switch {
	case msg.Photo != nil:
		return copyPhoto(msg.Photo, msg.Caption, msg.CaptionEntities, msg.CaptionAbove, msg.HasMediaSpoiler), true
	case msg.Video != nil:
		return copyVideo(msg.Video, msg.Caption, msg.CaptionEntities, msg.CaptionAbove, msg.HasMediaSpoiler), true
	case msg.Document != nil:
		return copyDocument(msg.Document, msg.Caption, msg.CaptionEntities), true
	case msg.Audio != nil:
		return copyAudio(msg.Audio, msg.Caption, msg.CaptionEntities), true
	case msg.Voice != nil:
		return copyVoice(msg.Voice, msg.Caption, msg.CaptionEntities), true
	case msg.VideoNote != nil:
		return copyVideoNote(msg.VideoNote), true
	case msg.Sticker != nil:
		return copySticker(msg.Sticker), true
	case msg.Animation != nil:
		return copyAnimation(msg.Animation, msg.Caption, msg.CaptionEntities, msg.HasMediaSpoiler), true
	case msg.Location != nil:
		return copyLocation(msg.Location), true
	case msg.Venue != nil:
		return copyVenue(msg.Venue), true
	case msg.Contact != nil:
		return copyContact(msg.Contact), true
	case msg.Dice != nil:
		return copyDice(msg.Dice), true
	case msg.Poll != nil:
		return copyPoll(msg.Poll), true
	case msg.Invoice != nil:
		return copyInvoice(msg.Invoice), true
	case msg.Game != nil:
		return copyGame(msg.Game), true
	case len(msg.PaidMedia.PaidMedia) > 0:
		return fmt.Sprintf("💰 Платный контент (%d медиа, %d звёзд)", len(msg.PaidMedia.PaidMedia), msg.PaidMedia.Stars), true
	case msg.Text != "" || msg.Caption != "":
		text := msg.Text
		if text == "" {
			text = msg.Caption
		}
		return text, true
	default:
		// Service messages and other types - format as text
		return formatServiceMessage(msg), true
	}
}

func copyPhoto(p *tele.Photo, caption string, entities tele.Entities, captionAbove, hasSpoiler bool) *tele.Photo {
	if p == nil {
		return nil
	}
	return &tele.Photo{
		File:         p.File,
		Width:        p.Width,
		Height:       p.Height,
		Caption:      caption,
		HasSpoiler:   hasSpoiler || p.HasSpoiler,
		CaptionAbove: captionAbove || p.CaptionAbove,
	}
}

func copyVideo(v *tele.Video, caption string, entities tele.Entities, captionAbove, hasSpoiler bool) *tele.Video {
	if v == nil {
		return nil
	}
	return &tele.Video{
		File:              v.File,
		Width:             v.Width,
		Height:            v.Height,
		Duration:          v.Duration,
		Caption:           caption,
		Thumbnail:         v.Thumbnail,
		Streaming:         v.Streaming,
		MIME:              v.MIME,
		FileName:          v.FileName,
		HasSpoiler:        hasSpoiler || v.HasSpoiler,
		CaptionAbove:      captionAbove || v.CaptionAbove,
		Cover:             v.Cover,
		StartTimestamp:    v.StartTimestamp,
	}
}

func copyDocument(d *tele.Document, caption string, entities tele.Entities) *tele.Document {
	if d == nil {
		return nil
	}
	return &tele.Document{
		File:                   d.File,
		Thumbnail:              d.Thumbnail,
		Caption:                caption,
		MIME:                   d.MIME,
		FileName:               d.FileName,
		DisableTypeDetection:   d.DisableTypeDetection,
	}
}

func copyAudio(a *tele.Audio, caption string, entities tele.Entities) *tele.Audio {
	if a == nil {
		return nil
	}
	return &tele.Audio{
		File:      a.File,
		Duration:  a.Duration,
		Caption:   caption,
		Thumbnail: a.Thumbnail,
		Title:     a.Title,
		Performer: a.Performer,
		MIME:      a.MIME,
		FileName:  a.FileName,
	}
}

func copyVoice(v *tele.Voice, caption string, entities tele.Entities) *tele.Voice {
	if v == nil {
		return nil
	}
	return &tele.Voice{
		File:     v.File,
		Duration: v.Duration,
		Caption:  caption,
		MIME:    v.MIME,
	}
}

func copyVideoNote(v *tele.VideoNote) *tele.VideoNote {
	if v == nil {
		return nil
	}
	return &tele.VideoNote{
		File:      v.File,
		Duration:  v.Duration,
		Thumbnail: v.Thumbnail,
		Length:    v.Length,
	}
}

func copySticker(s *tele.Sticker) *tele.Sticker {
	if s == nil {
		return nil
	}
	return &tele.Sticker{
		File:              s.File,
		Type:              s.Type,
		Width:             s.Width,
		Height:            s.Height,
		Animated:          s.Animated,
		Video:             s.Video,
		Thumbnail:         s.Thumbnail,
		Emoji:             s.Emoji,
		SetName:           s.SetName,
		PremiumAnimation:  s.PremiumAnimation,
		MaskPosition:      s.MaskPosition,
		CustomEmojiID:     s.CustomEmojiID,
		Repaint:           s.Repaint,
	}
}

func copyAnimation(a *tele.Animation, caption string, entities tele.Entities, hasSpoiler bool) *tele.Animation {
	if a == nil {
		return nil
	}
	return &tele.Animation{
		File:         a.File,
		Width:       a.Width,
		Height:      a.Height,
		Duration:    a.Duration,
		Caption:     caption,
		Thumbnail:   a.Thumbnail,
		MIME:       a.MIME,
		FileName:   a.FileName,
		HasSpoiler: hasSpoiler || a.HasSpoiler,
		CaptionAbove: a.CaptionAbove,
	}
}

func copyLocation(l *tele.Location) *tele.Location {
	if l == nil {
		return nil
	}
	loc := &tele.Location{
		Lat:       l.Lat,
		Lng:       l.Lng,
		Heading:   l.Heading,
		AlertRadius: l.AlertRadius,
		LivePeriod: l.LivePeriod,
	}
	if l.HorizontalAccuracy != nil {
		acc := *l.HorizontalAccuracy
		loc.HorizontalAccuracy = &acc
	}
	return loc
}

func copyVenue(v *tele.Venue) *tele.Venue {
	if v == nil {
		return nil
	}
	return &tele.Venue{
		Location:       v.Location,
		Title:         v.Title,
		Address:       v.Address,
		FoursquareID:  v.FoursquareID,
		FoursquareType: v.FoursquareType,
		GooglePlaceID: v.GooglePlaceID,
		GooglePlaceType: v.GooglePlaceType,
	}
}

func copyContact(c *tele.Contact) *tele.Contact {
	if c == nil {
		return nil
	}
	return &tele.Contact{
		PhoneNumber: c.PhoneNumber,
		FirstName:  c.FirstName,
		LastName:   c.LastName,
		UserID:     c.UserID,
		VCard:      c.VCard,
	}
}

func copyDice(d *tele.Dice) *tele.Dice {
	if d == nil {
		return nil
	}
	return &tele.Dice{
		Type:  d.Type,
		Value: d.Value,
	}
}

func copyPoll(p *tele.Poll) *tele.Poll {
	if p == nil {
		return nil
	}
	return &tele.Poll{
		Type:           p.Type,
		Question:       p.Question,
		Options:        p.Options,
		VoterCount:     p.VoterCount,
		Closed:         p.Closed,
		Anonymous:      p.Anonymous,
		MultipleAnswers: p.MultipleAnswers,
		CorrectOption:  p.CorrectOption,
		Explanation:    p.Explanation,
		ParseMode:      p.ParseMode,
		OpenPeriod:     p.OpenPeriod,
		CloseUnixdate:  p.CloseUnixdate,
	}
}

func copyInvoice(i *tele.Invoice) *tele.Invoice {
	if i == nil {
		return nil
	}
	return i
}

func copyGame(g *tele.Game) *tele.Game {
	if g == nil {
		return nil
	}
	return g
}

func formatServiceMessage(msg *tele.Message) string {
	var parts []string

	switch {
	case msg.UserJoined != nil:
		parts = append(parts, fmt.Sprintf("👤 Пользователь %s присоединился", formatUser(msg.UserJoined)))
	case len(msg.UsersJoined) > 0:
		names := make([]string, len(msg.UsersJoined))
		for i, u := range msg.UsersJoined {
			names[i] = formatUser(&u)
		}
		parts = append(parts, fmt.Sprintf("👥 Присоединились: %s", strings.Join(names, ", ")))
	case msg.UserLeft != nil:
		parts = append(parts, fmt.Sprintf("👤 Пользователь %s вышел", formatUser(msg.UserLeft)))
	case msg.NewGroupTitle != "":
		parts = append(parts, fmt.Sprintf("📝 Новое название: %s", msg.NewGroupTitle))
	case msg.NewGroupPhoto != nil:
		parts = append(parts, "📷 Установлено новое фото чата")
	case msg.GroupPhotoDeleted:
		parts = append(parts, "📷 Фото чата удалено")
	case msg.GroupCreated:
		parts = append(parts, "💬 Группа создана")
	case msg.SuperGroupCreated:
		parts = append(parts, "💬 Супергруппа создана")
	case msg.ChannelCreated:
		parts = append(parts, "📢 Канал создан")
	case msg.MigrateTo != 0:
		parts = append(parts, fmt.Sprintf("🔄 Миграция в чат: %d", msg.MigrateTo))
	case msg.MigrateFrom != 0:
		parts = append(parts, fmt.Sprintf("🔄 Миграция из чата: %d", msg.MigrateFrom))
	case msg.PinnedMessage != nil:
		pinned := msg.PinnedMessage
		preview := getMessageTextPreview(pinned, 80)
		parts = append(parts, fmt.Sprintf("📌 Закреплено: %s", preview))
	case msg.VideoChatStarted != nil:
		parts = append(parts, "📹 Видеозвонок начат")
	case msg.VideoChatEnded != nil:
		if msg.VideoChatEnded.Duration > 0 {
			parts = append(parts, fmt.Sprintf("📹 Видеозвонок завершён (длительность: %d сек)", msg.VideoChatEnded.Duration))
		} else {
			parts = append(parts, "📹 Видеозвонок завершён")
		}
	case msg.VideoChatParticipants != nil:
		parts = append(parts, "📹 Приглашение в видеозвонок")
	case msg.VideoChatScheduled != nil:
		parts = append(parts, "📹 Видеозвонок запланирован")
	case msg.Payment != nil:
		parts = append(parts, fmt.Sprintf("💰 Оплата: %s %.2f", msg.Payment.Currency, float64(msg.Payment.Total)/100))
	case msg.Invoice != nil:
		parts = append(parts, fmt.Sprintf("🧾 Счёт: %s", msg.Invoice.Title))
	case msg.Giveaway != nil:
		parts = append(parts, "🎁 Розыгрыш")
	case msg.GiveawayWinners != nil:
		parts = append(parts, "🎁 Розыгрыш завершён")
	case msg.TopicCreated != nil:
		parts = append(parts, fmt.Sprintf("📋 Топик создан: %s", msg.TopicCreated.Name))
	case msg.TopicClosed != nil:
		parts = append(parts, "📋 Топик закрыт")
	case msg.TopicReopened != nil:
		parts = append(parts, "📋 Топик открыт")
	case msg.BoostAdded != nil:
		parts = append(parts, "⭐ Усиление чата")
	default:
		parts = append(parts, "📢 Служебное сообщение")
	}

	if msg.Sender != nil {
		parts = append([]string{formatUser(msg.Sender) + ":"}, parts...)
	}

	return strings.Join(parts, " ")
}

func formatUser(u *tele.User) string {
	if u == nil {
		return ""
	}
	if u.Username != "" {
		return "@" + u.Username
	}
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

func getMessageTextPreview(msg *tele.Message, maxLen int) string {
	if msg == nil {
		return ""
	}
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	if text == "" {
		// Describe media type
		switch {
		case msg.Photo != nil:
			return "[фото]"
		case msg.Video != nil:
			return "[видео]"
		case msg.Document != nil:
			return "[документ]"
		case msg.Audio != nil:
			return "[аудио]"
		case msg.Voice != nil:
			return "[голосовое]"
		case msg.VideoNote != nil:
			return "[видеообщение]"
		case msg.Sticker != nil:
			return "[стикер]"
		case msg.Animation != nil:
			return "[GIF]"
		case msg.Location != nil:
			return "[локация]"
		case msg.Contact != nil:
			return "[контакт]"
		default:
			return "[медиа]"
		}
	}
	if len(text) > maxLen {
		return text[:maxLen] + "..."
	}
	return text
}
