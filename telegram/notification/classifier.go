package notification

import (
	"encoding/json"

	"github.com/ChatDetectiveORG/shared/telegram/rawmessage"
	"github.com/ChatDetectiveORG/shared/utils"
)

type Strategy int

const (
	StrategyTextCombined Strategy = iota
	StrategyMediaGroup
	StrategyReplayWithNotice
)

const combinedTextThreshold = 3900

// ClassifyEdit chooses how to notify about an edited business message.
func ClassifyEdit(oldRaw, newRaw json.RawMessage, mediaGroupIDHash string) Strategy {
	if mediaGroupIDHash != "" {
		return StrategyMediaGroup
	}
	if rawmessage.IsPlainText(oldRaw) && rawmessage.IsPlainText(newRaw) {
		oldText, _, oldOK := rawmessage.ExtractTextFields(oldRaw)
		newText, _, newOK := rawmessage.ExtractTextFields(newRaw)
		if oldOK && newOK && utils.TgLen(oldText)+utils.TgLen(newText) <= combinedTextThreshold {
			return StrategyTextCombined
		}
	}
	return StrategyReplayWithNotice
}

// ClassifyDelete chooses how to notify about a deleted business message.
func ClassifyDelete(raw json.RawMessage, mediaGroupIDHash string) Strategy {
	if mediaGroupIDHash != "" {
		return StrategyMediaGroup
	}
	if rawmessage.IsPlainText(raw) {
		text, _, ok := rawmessage.ExtractTextFields(raw)
		if ok && utils.TgLen(text) <= combinedTextThreshold {
			return StrategyTextCombined
		}
	}
	return StrategyReplayWithNotice
}
