package rawmessage

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChecklistReplayAttemptsCopyPollText(t *testing.T) {
	raw := json.RawMessage(`{
		"message_id": 99,
		"chat": {"id": 1001},
		"checklist": {
			"title": "Shopping",
			"tasks": [
				{"id": 1, "text": "Milk"},
				{"id": 2, "text": "Bread"}
			]
		}
	}`)

	attempts, err := ChecklistReplayAttempts(raw, ReplayOptions{
		TargetChatID:     42,
		AllowCopyMessage: true,
	})
	if err != nil {
		t.Fatalf("ChecklistReplayAttempts: %v", err)
	}
	if len(attempts) != 3 {
		t.Fatalf("attempts len = %d, want 3", len(attempts))
	}
	if attempts[0].Method != "copyMessage" {
		t.Fatalf("attempt[0] method = %q", attempts[0].Method)
	}
	if attempts[0].Payload["from_chat_id"] != int64(1001) || attempts[0].Payload["message_id"] != 99 {
		t.Fatalf("copy payload = %#v", attempts[0].Payload)
	}
	if attempts[1].Method != "sendPoll" {
		t.Fatalf("attempt[1] method = %q", attempts[1].Method)
	}
	if attempts[1].Payload["allows_multiple_answers"] != true {
		t.Fatalf("poll payload = %#v", attempts[1].Payload)
	}
	if attempts[2].Method != "sendMessage" {
		t.Fatalf("attempt[2] method = %q", attempts[2].Method)
	}
}

func TestChecklistReplayAttemptsSkipsCopyWhenDisabled(t *testing.T) {
	raw := json.RawMessage(`{
		"message_id": 99,
		"chat": {"id": 1001},
		"checklist": {
			"title": "Shopping",
			"tasks": [
				{"id": 1, "text": "Milk"},
				{"id": 2, "text": "Bread"}
			]
		}
	}`)

	attempts, err := ChecklistReplayAttempts(raw, ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("ChecklistReplayAttempts: %v", err)
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts len = %d, want 2", len(attempts))
	}
	if attempts[0].Method != "sendPoll" {
		t.Fatalf("attempt[0] method = %q, want sendPoll", attempts[0].Method)
	}
}

func TestChecklistReplayAttemptsSingleTaskTextOnly(t *testing.T) {
	raw := json.RawMessage(`{
		"checklist": {
			"title": "One",
			"tasks": [{"id": 1, "text": "Only task"}]
		}
	}`)

	attempts, err := ChecklistReplayAttempts(raw, ReplayOptions{TargetChatID: 42})
	if err != nil {
		t.Fatalf("ChecklistReplayAttempts: %v", err)
	}
	if len(attempts) != 1 || attempts[0].Method != "sendMessage" {
		t.Fatalf("attempts = %#v, want single sendMessage", attempts)
	}
	text, _ := attempts[0].Payload["text"].(string)
	if text == "" || !strings.Contains(text, ChecklistOriginNotice) {
		t.Fatalf("text = %#v", text)
	}
}
