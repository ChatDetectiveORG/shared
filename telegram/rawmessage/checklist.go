package rawmessage

import (
	"encoding/json"
	"fmt"
)

// ChecklistOriginNotice is prepended to notification summaries and text fallbacks.
const ChecklistOriginNotice = "Оригинально сообщение было чеклистом."

// SendAttempt is one Telegram API call candidate for checklist notification replay.
type SendAttempt struct {
	Method  string
	Payload map[string]any
}

// HasChecklistContent reports whether raw JSON contains a checklist message body.
func HasChecklistContent(raw json.RawMessage) bool {
	m, err := decodeMap(raw)
	if err != nil {
		return false
	}
	return hasContent(m, "checklist")
}

// ChecklistTitleFromRaw extracts the checklist title with entities (e.g. custom emoji).
func ChecklistTitleFromRaw(raw json.RawMessage) (FormattedText, bool) {
	m, err := decodeMap(raw)
	if err != nil {
		return FormattedText{}, false
	}
	checklistObj, ok := m["checklist"].(map[string]any)
	if !ok {
		return FormattedText{}, false
	}
	return checklistTitleFormatted(checklistObj)
}

// ChecklistReplayAttempts builds ordered strategies for notification replay:
// copyMessage (optional) → sendPoll (multi-answer) → sendMessage (text).
func ChecklistReplayAttempts(raw json.RawMessage, opts ReplayOptions) ([]SendAttempt, error) {
	m, err := decodeMap(raw)
	if err != nil {
		return nil, err
	}
	if !hasContent(m, "checklist") {
		return nil, fmt.Errorf("rawmessage: not a checklist message")
	}

	attempts := make([]SendAttempt, 0, 3)
	if opts.AllowCopyMessage {
		if payload, ok := buildCopyMessagePayload(m, opts); ok {
			attempts = append(attempts, SendAttempt{Method: "copyMessage", Payload: payload})
		}
	}
	if payload, err := buildChecklistPollPayload(m, opts); err == nil {
		attempts = append(attempts, SendAttempt{Method: "sendPoll", Payload: payload})
	}
	if payload, err := buildChecklistTextNotificationPayload(m); err == nil {
		attempts = append(attempts, SendAttempt{Method: "sendMessage", Payload: payload})
	}
	if len(attempts) == 0 {
		return nil, fmt.Errorf("rawmessage: no checklist replay strategy available")
	}
	return attempts, nil
}

func checklistTitleFormatted(checklistObj map[string]any) (FormattedText, bool) {
	title, _ := checklistObj["title"].(string)
	if title == "" {
		return FormattedText{}, false
	}
	return FormattedText{
		Text:     title,
		Entities: rawEntitiesSlice(checklistObj["title_entities"]),
	}, true
}

func buildCopyMessagePayload(m map[string]any, opts ReplayOptions) (map[string]any, bool) {
	messageID := intFromAny(m["message_id"])
	fromChatID := chatIDFromMessageMap(m)
	if messageID == 0 || fromChatID == 0 {
		return nil, false
	}
	payload := map[string]any{
		"chat_id":      opts.TargetChatID,
		"from_chat_id": fromChatID,
		"message_id":   messageID,
	}
	if opts.ReplyToMessageID != 0 {
		payload["reply_to_message_id"] = opts.ReplyToMessageID
	}
	return payload, true
}

func buildChecklistPollPayload(m map[string]any, opts ReplayOptions) (map[string]any, error) {
	checklistObj, ok := m["checklist"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: checklist is missing or invalid")
	}
	tasks, err := inputChecklistTasksFromMap(checklistObj["tasks"])
	if err != nil {
		return nil, err
	}
	if len(tasks) < 2 || len(tasks) > 10 {
		return nil, fmt.Errorf("rawmessage: poll fallback requires 2-10 checklist tasks")
	}

	question, truncated := buildChecklistPollQuestion(checklistObj)
	options, err := buildChecklistPollOptions(tasks)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"options":                 options,
		"type":                    "regular",
		"is_closed":               true,
		"allows_multiple_answers": true,
	}
	if !truncated {
		question.ApplyPollQuestion(payload)
	} else {
		payload["question"] = question.Text
	}
	payload["chat_id"] = opts.TargetChatID
	if opts.ReplyToMessageID != 0 {
		payload["reply_to_message_id"] = opts.ReplyToMessageID
	}
	return payload, nil
}

func buildChecklistPollQuestion(checklistObj map[string]any) (FormattedText, bool) {
	notice := FormattedText{Text: ChecklistOriginNotice}
	title, hasTitle := checklistTitleFormatted(checklistObj)
	var question FormattedText
	if hasTitle {
		question = JoinFormatted("\n", notice, title)
	} else {
		question = notice
	}
	return question.Truncate(300)
}

func buildChecklistPollOptions(tasks []map[string]any) ([]map[string]any, error) {
	options := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		text, _ := task["text"].(string)
		if text == "" {
			return nil, fmt.Errorf("rawmessage: poll option text is required")
		}
		option := FormattedText{Text: text, Entities: rawEntitiesSlice(task["text_entities"])}
		truncated, wasTruncated := option.Truncate(100)
		opt := map[string]any{"text": truncated.Text}
		if !wasTruncated && len(truncated.Entities) > 0 {
			opt["text_entities"] = truncated.Entities
		}
		options = append(options, opt)
	}
	return options, nil
}

func buildChecklistBody(checklistObj map[string]any) (FormattedText, error) {
	var lines []FormattedText
	if title, ok := checklistTitleFormatted(checklistObj); ok {
		lines = append(lines, title)
	}

	items, ok := checklistObj["tasks"].([]any)
	if !ok || len(items) == 0 {
		if len(lines) == 0 {
			return FormattedText{}, fmt.Errorf("rawmessage: checklist has no displayable text")
		}
		return JoinFormatted("\n", lines...), nil
	}

	for _, item := range items {
		taskMap, mapOK := item.(map[string]any)
		if !mapOK {
			continue
		}
		taskText, _ := taskMap["text"].(string)
		if taskText == "" {
			continue
		}
		task := FormattedText{
			Text:     taskText,
			Entities: rawEntitiesSlice(taskMap["text_entities"]),
		}
		lines = append(lines, task.WithLiteralPrefix("- "))
	}

	body := JoinFormatted("\n", lines...)
	if body.IsEmpty() {
		return FormattedText{}, fmt.Errorf("rawmessage: checklist has no displayable text")
	}
	return body, nil
}

func buildChecklistTextNotificationPayload(m map[string]any) (map[string]any, error) {
	checklistObj, ok := m["checklist"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: checklist is missing or invalid")
	}
	body, err := buildChecklistBody(checklistObj)
	if err != nil {
		return nil, err
	}
	message := JoinFormatted("\n\n", FormattedText{Text: ChecklistOriginNotice}, body)
	payload := map[string]any{}
	message.ApplyTextFields(payload)
	return payload, nil
}

func buildSendChecklistTextPayload(m map[string]any) (map[string]any, error) {
	checklistObj, ok := m["checklist"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: checklist is missing or invalid")
	}
	body, err := buildChecklistBody(checklistObj)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	body.ApplyTextFields(payload)
	return payload, nil
}

func buildChecklistNotificationPayload(m map[string]any, opts ReplayOptions) (string, map[string]any, error) {
	raw, err := json.Marshal(m)
	if err != nil {
		return "", nil, err
	}
	attempts, err := ChecklistReplayAttempts(raw, opts)
	if err != nil {
		return "", nil, err
	}
	for _, attempt := range attempts {
		if attempt.Method == "copyMessage" {
			continue
		}
		return attempt.Method, attempt.Payload, nil
	}
	last := attempts[len(attempts)-1]
	return last.Method, last.Payload, nil
}

func chatIDFromMessageMap(m map[string]any) int64 {
	chatObj, ok := m["chat"].(map[string]any)
	if !ok {
		return 0
	}
	switch id := chatObj["id"].(type) {
	case float64:
		return int64(id)
	case int:
		return int64(id)
	case int64:
		return id
	case json.Number:
		i, _ := id.Int64()
		return i
	default:
		return 0
	}
}

func truncateUTF16(s string, maxUnits int) string {
	if maxUnits <= 0 {
		return ""
	}
	units := 0
	for i, r := range s {
		runeLen := 1
		if r >= 0x10000 {
			runeLen = 2
		}
		if units+runeLen > maxUnits {
			return s[:i]
		}
		units += runeLen
	}
	return s
}
