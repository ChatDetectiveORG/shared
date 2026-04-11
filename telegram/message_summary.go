package telegram

import (
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
)

// MessageSummary — структурированные метаданные сообщения для отображения и аудита.
type MessageSummary struct {
	// Ответ / цитата
	ReplyToPreview    string `json:"reply_to_preview,omitempty"` // до 100 символов текста ответа или описание типа
	ReplyToMessageID  int    `json:"reply_to_message_id,omitempty"`
	ReplyToChatID     int64  `json:"reply_to_chat_id,omitempty"`
	QuoteText         string `json:"quote_text,omitempty"`
	QuoteFromChat     string `json:"quote_from_chat,omitempty"`
	QuoteFromUser     string `json:"quote_from_user,omitempty"`
	ExternalReplyInfo string `json:"external_reply_info,omitempty"`

	// Через бота
	ViaBotUsername string `json:"via_bot_username,omitempty"`
	ViaBotID       int64  `json:"via_bot_id,omitempty"`

	// Пересылка
	ForwardedFromUser      string `json:"forwarded_from_user,omitempty"`
	ForwardedFromChat      string `json:"forwarded_from_chat,omitempty"`
	ForwardedFromMessageID int    `json:"forwarded_from_message_id,omitempty"`
	ForwardedDate          string `json:"forwarded_date,omitempty"`
	ForwardedSignature     string `json:"forwarded_signature,omitempty"`
	ForwardedSenderName    string `json:"forwarded_sender_name,omitempty"`
	// Дополнение к пересылке (только то, что не вынесено в поля выше).
	ForwardOriginInfo  string `json:"forward_origin_info,omitempty"`
	IsAutomaticForward bool   `json:"is_automatic_forward,omitempty"`

	// Ответ на историю
	ReplyToStoryInfo string `json:"reply_to_story_info,omitempty"`

	// Прочее
	ProtectedContent bool `json:"protected_content,omitempty"`
	FromOffline      bool `json:"from_offline,omitempty"`
	ForumTopicID     int  `json:"forum_topic_id,omitempty"`
}

type originDisplayContext struct {
	userDisplay string
	chatDisplay string
	messageID   int
	signature   string
}

// BuildMessageSummary собирает сводку по всем доступным полям сообщения.
func BuildMessageSummary(msg *tele.Message) *MessageSummary {
	if msg == nil {
		return nil
	}

	s := &MessageSummary{
		ProtectedContent: msg.Protected,
		FromOffline:      msg.FromOffline,
		ForumTopicID:     msg.ThreadID,
	}

	if msg.Via != nil {
		s.ViaBotID = msg.Via.ID
		s.ViaBotUsername = msg.Via.Username
		if s.ViaBotUsername != "" {
			s.ViaBotUsername = "@" + s.ViaBotUsername
		}
	}

	if msg.ReplyTo != nil {
		s.ReplyToMessageID = msg.ReplyTo.ID
		s.ReplyToChatID = msg.ReplyTo.Chat.ID
		s.ReplyToPreview = getMessageTextPreview(msg.ReplyTo, 100)
	}

	if msg.Quote != nil {
		s.QuoteText = msg.Quote.Text
		if len(s.QuoteText) > 100 {
			s.QuoteText = s.QuoteText[:100] + "..."
		}
	}

	if msg.ExternalReply != nil {
		er := msg.ExternalReply
		if er.Chat != nil {
			s.QuoteFromChat = formatChat(er.Chat)
		}
		if er.Origin != nil && er.Origin.Sender != nil {
			s.QuoteFromUser = formatUser(er.Origin.Sender)
		}
		if s.QuoteFromUser == "" && er.Origin != nil && er.Origin.SenderUsername != "" {
			s.QuoteFromUser = "@" + strings.TrimPrefix(er.Origin.SenderUsername, "@")
		}
		s.ReplyToMessageID = er.MessageID
		if s.ReplyToPreview == "" && er.Origin != nil {
			s.ReplyToPreview = formatOriginPreview(er.Origin)
		}
		s.ExternalReplyInfo = formatExternalReply(er, s.QuoteFromUser, s.QuoteFromChat, s.ReplyToMessageID)
	}

	if msg.OriginalSender != nil {
		s.ForwardedFromUser = formatUser(msg.OriginalSender)
	}
	if msg.OriginalChat != nil {
		s.ForwardedFromChat = formatChat(msg.OriginalChat)
		s.ForwardedFromMessageID = msg.OriginalMessageID
	}
	if msg.OriginalUnixtime != 0 {
		s.ForwardedDate = formatForwardedDateRussian(int64(msg.OriginalUnixtime))
	}
	s.ForwardedSignature = msg.OriginalSignature
	s.ForwardedSenderName = msg.OriginalSenderName
	s.IsAutomaticForward = msg.AutomaticForward

	if msg.Origin != nil {
		mergeMessageOriginIntoSummary(s, msg.Origin)
		s.ForwardOriginInfo = forwardOriginSupplemental(msg.Origin, s)
	}

	if msg.Story != nil {
		s.ReplyToStoryInfo = formatStory(msg.Story)
	}
	if msg.ReplyToStory != nil {
		s.ReplyToStoryInfo = formatStory(msg.ReplyToStory)
	}

	return s
}

func mergeMessageOriginIntoSummary(s *MessageSummary, o *tele.MessageOrigin) {
	if s == nil || o == nil {
		return
	}
	typ := normalizeOriginType(o.Type)

	if s.ForwardedFromUser == "" && o.Sender != nil {
		s.ForwardedFromUser = formatUser(o.Sender)
	}
	if s.ForwardedFromUser == "" && o.SenderUsername != "" {
		s.ForwardedFromUser = "@" + strings.TrimPrefix(o.SenderUsername, "@")
	}

	switch typ {
	case "chat":
		if o.SenderChat != nil && s.ForwardedFromChat == "" {
			s.ForwardedFromChat = formatChat(o.SenderChat)
		}
	case "channel":
		if o.Chat != nil && s.ForwardedFromChat == "" {
			s.ForwardedFromChat = formatChat(o.Chat)
		}
	default:
		if s.ForwardedFromChat == "" && o.Chat != nil {
			s.ForwardedFromChat = formatChat(o.Chat)
		}
		if s.ForwardedFromChat == "" && o.SenderChat != nil {
			s.ForwardedFromChat = formatChat(o.SenderChat)
		}
	}

	if s.ForwardedFromMessageID == 0 && o.MessageID != 0 {
		s.ForwardedFromMessageID = o.MessageID
	}
	if s.ForwardedSignature == "" && o.Signature != "" {
		s.ForwardedSignature = o.Signature
	}
	if o.DateUnixtime != 0 {
		s.ForwardedDate = formatForwardedDateRussian(o.DateUnixtime)
	}
}

func normalizeOriginType(t string) string {
	t = strings.TrimSpace(t)
	t = strings.TrimPrefix(t, "message_origin_")
	return t
}

func russianOriginTypeNoun(typ string) string {
	switch typ {
	case "user":
		return "пользователь"
	case "hidden_user":
		return "скрытый пользователь"
	case "chat":
		return "чат"
	case "channel":
		return "канал"
	default:
		if typ == "" {
			return ""
		}
		return typ
	}
}

func formatForwardedDateRussian(unix int64) string {
	if unix == 0 {
		return ""
	}
	t := time.Unix(unix, 0)
	return "дата оригинала: " + t.Format("02.01.2006 15:04")
}

// Дополнительная строка про происхождение пересылки без дублирования уже показанных полей.
func forwardOriginSupplemental(o *tele.MessageOrigin, s *MessageSummary) string {
	if o == nil {
		return ""
	}
	ctx := originDisplayContext{
		userDisplay: s.ForwardedFromUser,
		chatDisplay: s.ForwardedFromChat,
		messageID:   s.ForwardedFromMessageID,
		signature:   s.ForwardedSignature,
	}
	return originSupplementalText(o, ctx)
}

func originSupplementalText(o *tele.MessageOrigin, ctx originDisplayContext) string {
	typ := normalizeOriginType(o.Type)
	var parts []string

	userFromO := originUserDisplay(o)
	chatFromO := ""
	if o.Chat != nil {
		chatFromO = formatChat(o.Chat)
	}
	senderChatStr := ""
	if o.SenderChat != nil {
		senderChatStr = formatChat(o.SenderChat)
	}

	switch typ {
	case "hidden_user":
		parts = append(parts, "отправитель со скрытым профилем")
	case "user", "channel", "chat":
		// тип очевиден из от/из; отдельную метку не дублируем
	default:
		if typ != "" {
			parts = append(parts, "тип источника: "+russianOriginTypeNoun(typ))
		}
	}

	// Пользователь из origin, если отличается от уже показанного
	if userFromO != "" && userFromO != ctx.userDisplay {
		parts = append(parts, "отправитель: "+userFromO)
	}

	// Чат канала vs чат отправителя — только если не совпадает с уже выведенным «из …»
	if chatFromO != "" && chatFromO != ctx.chatDisplay {
		parts = append(parts, "канал: "+chatFromO)
	}
	if senderChatStr != "" && senderChatStr != ctx.chatDisplay && senderChatStr != chatFromO {
		parts = append(parts, "чат отправителя: "+senderChatStr)
	}

	if o.MessageID != 0 && o.MessageID != ctx.messageID {
		parts = append(parts, "id сообщения: "+strconv.Itoa(o.MessageID))
	}
	if o.Signature != "" && o.Signature != ctx.signature {
		parts = append(parts, "подпись автора: "+o.Signature)
	}

	return strings.Join(parts, "; ")
}

func originUserDisplay(o *tele.MessageOrigin) string {
	if o == nil {
		return ""
	}
	if o.Sender != nil {
		return formatUser(o.Sender)
	}
	if o.SenderUsername != "" {
		return "@" + strings.TrimPrefix(o.SenderUsername, "@")
	}
	return ""
}

func formatExternalReply(ext *tele.ExternalReply, quoteUser, quoteChat string, replyToMsgID int) string {
	if ext == nil {
		return ""
	}
	var parts []string
	ctx := originDisplayContext{
		userDisplay: quoteUser,
		chatDisplay: quoteChat,
		messageID:   replyToMsgID,
	}
	if ext.Origin != nil {
		ctx.signature = ext.Origin.Signature
		if sup := originSupplementalText(ext.Origin, ctx); sup != "" {
			parts = append(parts, sup)
		}
	}
	if ext.Chat != nil {
		ch := formatChat(ext.Chat)
		if ch != quoteChat {
			parts = append(parts, "чат: "+ch)
		}
	}
	if ext.MessageID != 0 && ext.MessageID != replyToMsgID {
		parts = append(parts, "id сообщения: "+strconv.Itoa(ext.MessageID))
	}

	switch {
	case len(ext.Photo) > 0:
		parts = append(parts, "[фото]")
	case ext.Video != nil:
		parts = append(parts, "[видео]")
	case ext.Document != nil:
		parts = append(parts, "[документ]")
	case ext.Audio != nil:
		parts = append(parts, "[аудио]")
	case ext.Voice != nil:
		parts = append(parts, "[голосовое]")
	case ext.Sticker != nil:
		parts = append(parts, "[стикер]")
	case ext.Animation != nil:
		parts = append(parts, "[GIF]")
	case ext.Contact != nil:
		parts = append(parts, "[контакт]")
	case ext.Location != nil:
		parts = append(parts, "[локация]")
	case ext.Venue != nil:
		parts = append(parts, "[место]")
	case ext.Poll != nil:
		parts = append(parts, "[опрос]")
	}
	return strings.Join(parts, "; ")
}

func formatOriginPreview(origin *tele.MessageOrigin) string {
	if origin == nil {
		return ""
	}
	if origin.Sender != nil {
		return "ответ пользователю " + formatUser(origin.Sender)
	}
	if origin.SenderUsername != "" {
		return "ответ пользователю @" + strings.TrimPrefix(origin.SenderUsername, "@")
	}
	if origin.SenderChat != nil {
		return "ответ из чата " + formatChat(origin.SenderChat)
	}
	return "ответ из другого чата"
}

func formatChat(c *tele.Chat) string {
	if c == nil {
		return ""
	}
	if c.Username != "" {
		return "@" + c.Username
	}
	return c.Title + " (ид чата: " + strconv.FormatInt(c.ID, 10) + ")"
}

func formatStory(s *tele.Story) string {
	if s == nil {
		return ""
	}
	if s.Poster != nil {
		return "история (чат: " + formatChat(s.Poster) + ")"
	}
	return "история"
}

// String возвращает многострочное описание на русском.
func (s *MessageSummary) String() string {
	if s == nil {
		return ""
	}
	var lines []string

	if s.ReplyToPreview != "" {
		lines = append(lines, "Ответ на: "+s.ReplyToPreview)
		if s.ReplyToMessageID != 0 {
			lines = append(lines, "  id сообщения: "+strconv.Itoa(s.ReplyToMessageID))
		}
		if s.QuoteFromChat != "" {
			lines = append(lines, "  чат: "+s.QuoteFromChat)
		}
		if s.QuoteFromUser != "" {
			lines = append(lines, "  пользователь: "+s.QuoteFromUser)
		}
	}
	if s.QuoteText != "" {
		lines = append(lines, "Цитата: "+s.QuoteText)
	}
	if s.ExternalReplyInfo != "" {
		lines = append(lines, "Внешний ответ: "+s.ExternalReplyInfo)
	}
	if s.ViaBotUsername != "" {
		lines = append(lines, "Через бота: "+s.ViaBotUsername)
	}

	if s.ForwardedFromUser != "" || s.ForwardedFromChat != "" || s.ForwardedFromMessageID != 0 ||
		s.ForwardedSignature != "" || s.ForwardedSenderName != "" || s.ForwardOriginInfo != "" {
		var fwdParts []string
		if s.ForwardedFromUser != "" {
			fwdParts = append(fwdParts, "от "+s.ForwardedFromUser)
		}
		if s.ForwardedFromChat != "" {
			fwdParts = append(fwdParts, "из "+s.ForwardedFromChat)
		}
		if s.ForwardedFromMessageID != 0 {
			fwdParts = append(fwdParts, "id сообщения "+strconv.Itoa(s.ForwardedFromMessageID))
		}
		line := "Переслано"
		if len(fwdParts) > 0 {
			line += " " + strings.Join(fwdParts, ", ")
		}
		if s.ForwardedDate != "" {
			line += ", " + s.ForwardedDate
		}
		if s.ForwardedSignature != "" {
			line += ", подпись: " + s.ForwardedSignature
		}
		if s.ForwardedSenderName != "" {
			line += ", имя отправителя: " + s.ForwardedSenderName
		}
		lines = append(lines, line)
		if s.ForwardOriginInfo != "" {
			lines = append(lines, "  "+s.ForwardOriginInfo)
		}
	}

	if s.IsAutomaticForward {
		lines = append(lines, "Автоматическая пересылка")
	}
	if s.ReplyToStoryInfo != "" {
		lines = append(lines, "Ответ на историю: "+s.ReplyToStoryInfo)
	}
	if s.ProtectedContent {
		lines = append(lines, "Защищённый контент")
	}
	if s.FromOffline {
		lines = append(lines, "Отправлено офлайн")
	}

	return strings.Join(lines, "\n")
}
