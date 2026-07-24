package rawmessage

import (
	"github.com/ChatDetectiveORG/shared/utils"
	tele "gopkg.in/telebot.v4"
)

// FormattedText is Telegram message text with optional entity annotations (raw API shape).
type FormattedText struct {
	Text     string
	Entities []any
}

func (f FormattedText) IsEmpty() bool {
	return f.Text == "" && len(f.Entities) == 0
}

func (f FormattedText) WithLiteralPrefix(prefix string) FormattedText {
	if prefix == "" {
		return f
	}
	return FormattedText{
		Text:     prefix + f.Text,
		Entities: appendOffsetRawEntities(nil, f.Entities, utils.TgLen(prefix)),
	}
}

// JoinFormatted concatenates parts; entity offsets are adjusted for preceding text and separators.
func JoinFormatted(separator string, parts ...FormattedText) FormattedText {
	var out FormattedText
	for _, part := range parts {
		if part.IsEmpty() {
			continue
		}
		if out.Text != "" {
			out.Text += separator
		}
		offset := utils.TgLen(out.Text)
		out.Text += part.Text
		out.Entities = appendOffsetRawEntities(out.Entities, part.Entities, offset)
	}
	return out
}

func (f FormattedText) Truncate(maxUnits int) (FormattedText, bool) {
	if utils.TgLen(f.Text) <= maxUnits {
		return f, false
	}
	return FormattedText{Text: truncateUTF16(f.Text, maxUnits)}, true
}

func (f FormattedText) ApplyTextFields(payload map[string]any) {
	if f.Text != "" {
		payload["text"] = f.Text
	}
	if len(f.Entities) > 0 {
		payload["entities"] = f.Entities
	}
}

func (f FormattedText) ApplyPollQuestion(payload map[string]any) {
	payload["question"] = f.Text
	if len(f.Entities) > 0 {
		payload["question_entities"] = f.Entities
	}
}

func FormattedTextFromTeleSummary(text string, opts *tele.SendOptions) FormattedText {
	f := FormattedText{Text: text}
	if opts != nil && len(opts.Entities) > 0 {
		f.Entities = TeleEntitiesToRaw(opts.Entities)
	}
	return f
}

func (f FormattedText) ToTeleSendOptions(base *tele.SendOptions) (string, *tele.SendOptions) {
	opts := base
	if opts == nil {
		opts = &tele.SendOptions{DisableWebPagePreview: true}
	} else {
		copy := *opts
		opts = &copy
	}
	if len(f.Entities) > 0 {
		opts.Entities = TeleEntitiesFromRaw(f.Entities)
	}
	return f.Text, opts
}
