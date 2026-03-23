package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"
)

// MessageSummary contains structured metadata about a message for display/audit.
type MessageSummary struct {
	// Reply/Quote info
	ReplyToPreview    string `json:"reply_to_preview,omitempty"`    // First 100 chars of replied message or type description
	ReplyToMessageID  int    `json:"reply_to_message_id,omitempty"`
	ReplyToChatID     int64  `json:"reply_to_chat_id,omitempty"`
	QuoteText         string `json:"quote_text,omitempty"`
	QuoteFromChat     string `json:"quote_from_chat,omitempty"`
	QuoteFromUser     string `json:"quote_from_user,omitempty"`
	ExternalReplyInfo string `json:"external_reply_info,omitempty"`

	// Via bot
	ViaBotUsername string `json:"via_bot_username,omitempty"`
	ViaBotID       int64  `json:"via_bot_id,omitempty"`

	// Forward info
	ForwardedFromUser     string `json:"forwarded_from_user,omitempty"`
	ForwardedFromChat     string `json:"forwarded_from_chat,omitempty"`
	ForwardedFromMessageID int   `json:"forwarded_from_message_id,omitempty"`
	ForwardedDate         string `json:"forwarded_date,omitempty"`
	ForwardedSignature    string `json:"forwarded_signature,omitempty"`
	ForwardedSenderName   string `json:"forwarded_sender_name,omitempty"`
	ForwardOriginInfo     string `json:"forward_origin_info,omitempty"`
	IsAutomaticForward    bool   `json:"is_automatic_forward,omitempty"`

	// Story reply
	ReplyToStoryInfo string `json:"reply_to_story_info,omitempty"`

	// Other
	ProtectedContent bool   `json:"protected_content,omitempty"`
	FromOffline      bool   `json:"from_offline,omitempty"`
	ForumTopicID     int    `json:"forum_topic_id,omitempty"`
}

// BuildMessageSummary creates a structured summary with all available metadata.
func BuildMessageSummary(msg *tele.Message) *MessageSummary {
	if msg == nil {
		return nil
	}

	s := &MessageSummary{
		ProtectedContent: msg.Protected,
		FromOffline:      msg.FromOffline,
		ForumTopicID:     msg.ThreadID,
	}

	// Via bot
	if msg.Via != nil {
		s.ViaBotID = msg.Via.ID
		s.ViaBotUsername = msg.Via.Username
		if s.ViaBotUsername != "" {
			s.ViaBotUsername = "@" + s.ViaBotUsername
		}
	}

	// Reply to message (in same chat)
	if msg.ReplyTo != nil {
		s.ReplyToMessageID = msg.ReplyTo.ID
		s.ReplyToChatID = msg.ReplyTo.Chat.ID
		s.ReplyToPreview = getMessageTextPreview(msg.ReplyTo, 100)
	}

	// Quote (reply with quoted text)
	if msg.Quote != nil {
		s.QuoteText = msg.Quote.Text
		if len(s.QuoteText) > 100 {
			s.QuoteText = s.QuoteText[:100] + "..."
		}
	}

	// External reply (reply from another chat/forum)
	if msg.ExternalReply != nil {
		s.ExternalReplyInfo = formatExternalReply(msg.ExternalReply)
		if msg.ExternalReply.Chat != nil {
			s.QuoteFromChat = formatChat(msg.ExternalReply.Chat)
		}
		if msg.ExternalReply.Origin != nil && msg.ExternalReply.Origin.Sender != nil {
			s.QuoteFromUser = formatUser(msg.ExternalReply.Origin.Sender)
		}
		s.ReplyToMessageID = msg.ExternalReply.MessageID
		if s.ReplyToPreview == "" && msg.ExternalReply.Origin != nil {
			s.ReplyToPreview = formatOriginPreview(msg.ExternalReply.Origin)
		}
	}

	// Forward info (legacy: forward_from, forward_from_chat)
	if msg.OriginalSender != nil {
		s.ForwardedFromUser = formatUser(msg.OriginalSender)
	}
	if msg.OriginalChat != nil {
		s.ForwardedFromChat = formatChat(msg.OriginalChat)
		s.ForwardedFromMessageID = msg.OriginalMessageID
	}
	if msg.OriginalUnixtime != 0 {
		s.ForwardedDate = fmt.Sprintf("(date: %d)", msg.OriginalUnixtime)
	}
	s.ForwardedSignature = msg.OriginalSignature
	s.ForwardedSenderName = msg.OriginalSenderName
	s.IsAutomaticForward = msg.AutomaticForward

	// Forward origin (new API)
	if msg.Origin != nil {
		s.ForwardOriginInfo = formatMessageOrigin(msg.Origin)
		if s.ForwardedFromChat == "" && msg.Origin.Chat != nil {
			s.ForwardedFromChat = formatChat(msg.Origin.Chat)
		}
		if s.ForwardedFromMessageID == 0 {
			s.ForwardedFromMessageID = msg.Origin.MessageID
		}
		if msg.Origin.DateUnixtime != 0 {
			s.ForwardedDate = fmt.Sprintf("(date: %d)", msg.Origin.DateUnixtime)
		}
	}

	// Story reply
	if msg.Story != nil {
		s.ReplyToStoryInfo = formatStory(msg.Story)
	}
	if msg.ReplyToStory != nil {
		s.ReplyToStoryInfo = formatStory(msg.ReplyToStory)
	}

	return s
}

func formatExternalReply(ext *tele.ExternalReply) string {
	if ext == nil {
		return ""
	}
	var parts []string
	if ext.Origin != nil {
		parts = append(parts, formatMessageOrigin(ext.Origin))
	}
	if ext.Chat != nil {
		parts = append(parts, "чат: "+formatChat(ext.Chat))
	}
	if ext.MessageID != 0 {
		parts = append(parts, "msg_id: "+strconv.Itoa(ext.MessageID))
	}
	// Preview of quoted content from ExternalReply
	if len(ext.Photo) > 0 {
		parts = append(parts, "[фото]")
	} else if ext.Video != nil {
		parts = append(parts, "[видео]")
	} else if ext.Document != nil {
		parts = append(parts, "[документ]")
	} else if ext.Audio != nil {
		parts = append(parts, "[аудио]")
	} else if ext.Voice != nil {
		parts = append(parts, "[голосовое]")
	} else if ext.Sticker != nil {
		parts = append(parts, "[стикер]")
	} else if ext.Animation != nil {
		parts = append(parts, "[GIF]")
	} else if ext.Contact != nil {
		parts = append(parts, "[контакт]")
	} else if ext.Location != nil {
		parts = append(parts, "[локация]")
	} else if ext.Venue != nil {
		parts = append(parts, "[место]")
	} else if ext.Poll != nil {
		parts = append(parts, "[опрос]")
	}
	return strings.Join(parts, "; ")
}

func formatMessageOrigin(origin *tele.MessageOrigin) string {
	if origin == nil {
		return ""
	}
	var parts []string
	parts = append(parts, "origin_type: "+origin.Type)
	if origin.Sender != nil {
		parts = append(parts, "from: "+formatUser(origin.Sender))
	}
	if origin.SenderUsername != "" {
		parts = append(parts, "username: @"+origin.SenderUsername)
	}
	if origin.SenderChat != nil {
		parts = append(parts, "chat: "+formatChat(origin.SenderChat))
	}
	if origin.Chat != nil {
		parts = append(parts, "channel: "+formatChat(origin.Chat))
	}
	if origin.MessageID != 0 {
		parts = append(parts, "msg_id: "+strconv.Itoa(origin.MessageID))
	}
	if origin.Signature != "" {
		parts = append(parts, "signature: "+origin.Signature)
	}
	return strings.Join(parts, ", ")
}

func formatOriginPreview(origin *tele.MessageOrigin) string {
	if origin == nil {
		return ""
	}
	if origin.Sender != nil {
		return "ответ для " + formatUser(origin.Sender)
	}
	if origin.SenderChat != nil {
		return "ответ из " + formatChat(origin.SenderChat)
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
	return c.Title + " (id: " + strconv.FormatInt(c.ID, 10) + ")"
}

func formatStory(s *tele.Story) string {
	if s == nil {
		return ""
	}
	if s.Poster != nil {
		return "история (chat: " + formatChat(s.Poster) + ")"
	}
	return "история"
}

// String returns a human-readable multi-line summary.
func (s *MessageSummary) String() string {
	if s == nil {
		return ""
	}
	var lines []string

	if s.ReplyToPreview != "" {
		lines = append(lines, "↩ Ответ на: "+s.ReplyToPreview)
		if s.ReplyToMessageID != 0 {
			lines = append(lines, "  msg_id: "+strconv.Itoa(s.ReplyToMessageID))
		}
		if s.QuoteFromChat != "" {
			lines = append(lines, "  чат: "+s.QuoteFromChat)
		}
	}
	if s.QuoteText != "" {
		lines = append(lines, "📎 Цитата: "+s.QuoteText)
	}
	if s.ExternalReplyInfo != "" {
		lines = append(lines, "🔗 Внешний ответ: "+s.ExternalReplyInfo)
	}
	if s.ViaBotUsername != "" {
		lines = append(lines, "🤖 Через бота: "+s.ViaBotUsername)
	}
	if s.ForwardedFromUser != "" || s.ForwardedFromChat != "" {
		fwd := "📤 Переслано"
		if s.ForwardedFromUser != "" {
			fwd += " от " + s.ForwardedFromUser
		}
		if s.ForwardedFromChat != "" {
			fwd += " из " + s.ForwardedFromChat
		}
		if s.ForwardedFromMessageID != 0 {
			fwd += " (msg_id: " + strconv.Itoa(s.ForwardedFromMessageID) + ")"
		}
		if s.ForwardedSignature != "" {
			fwd += " [" + s.ForwardedSignature + "]"
		}
		if s.ForwardedSenderName != "" {
			fwd += " (" + s.ForwardedSenderName + ")"
		}
		if s.ForwardOriginInfo != "" {
			fwd += " | " + s.ForwardOriginInfo
		}
		lines = append(lines, fwd)
	}
	if s.IsAutomaticForward {
		lines = append(lines, "  (авто-пересылка)")
	}
	if s.ReplyToStoryInfo != "" {
		lines = append(lines, "📖 Ответ на историю: "+s.ReplyToStoryInfo)
	}
	if s.ProtectedContent {
		lines = append(lines, "🔒 Защищённый контент")
	}
	if s.FromOffline {
		lines = append(lines, "⏰ Отправлено офлайн")
	}

	return strings.Join(lines, "\n")
}
