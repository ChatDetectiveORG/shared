package notification

import (
	"encoding/json"

	"github.com/ChatDetectiveORG/shared/telegram"
	"github.com/ChatDetectiveORG/shared/telegram/rawmessage"
	tele "gopkg.in/telebot.v4"
)

func buildNoticeSummary(msg *tele.Message, raw json.RawMessage) (string, *tele.SendOptions) {
	baseText, baseOpts := telegram.BuildMessageSummary(msg)
	summary := rawmessage.FormattedTextFromTeleSummary(baseText, baseOpts)

	if !rawmessage.HasChecklistContent(raw) {
		return summary.ToTeleSendOptions(baseOpts)
	}

	parts := []rawmessage.FormattedText{{Text: rawmessage.ChecklistOriginNotice}}
	if title, ok := rawmessage.ChecklistTitleFromRaw(raw); ok {
		parts = append(parts, title)
	}
	if !summary.IsEmpty() {
		parts = append(parts, summary)
	}

	combined := rawmessage.JoinFormatted("\n", parts...)
	return combined.ToTeleSendOptions(baseOpts)
}
