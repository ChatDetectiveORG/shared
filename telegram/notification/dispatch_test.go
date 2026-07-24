package notification

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/ChatDetectiveORG/shared/handlers"
	"github.com/ChatDetectiveORG/shared/telegram"
	"github.com/ChatDetectiveORG/shared/telegram/rawmessage"
	amqp "github.com/rabbitmq/amqp091-go"
	tele "gopkg.in/telebot.v4"
)

func TestDispatchEditReplaysOldAndNewAnimationRawEnvelopes(t *testing.T) {
	publisher, initErr := handlers.NewOutgoingPublisher(handlers.OutgoingConfig{
		Channel:    &amqp.Channel{},
		JobsBuffer: 4,
	})
	if e.IsNonNil(initErr) {
		t.Fatalf("NewOutgoingPublisher: %v", initErr)
	}
	hashe := publisher.NewHashe()

	oldRaw := json.RawMessage(`{
		"animation":{"file_id":"old-animation","file_unique_id":"old-unique"},
		"caption":"old caption"
	}`)
	newRaw := json.RawMessage(`{
		"animation":{"file_id":"new-animation","file_unique_id":"new-unique"},
		"caption":"new caption"
	}`)

	requests := make(chan telegram.OutgoingRequest, 2)
	consumeErr := make(chan error, 1)
	go func() {
		rawCount := 0
		for range 4 {
			job := <-publisher.Jobs()
			body := append([]byte(nil), job.Body...)
			correlationID := job.CorrelationID

			var request telegram.OutgoingRequest
			if err := json.Unmarshal(body, &request); err != nil {
				consumeErr <- err
				return
			}
			if request.Kind != telegram.OutgoingRequestKindRawAPI {
				continue
			}
			requests <- request
			rawCount++

			waiter, ok := publisher.Waiters().Load(correlationID)
			if !ok {
				consumeErr <- &testError{"raw request waiter not found"}
				return
			}
			replyCh, ok := waiter.(chan *handlers.SendResult)
			if !ok {
				consumeErr <- &testError{"raw request waiter has unexpected type"}
				return
			}
			replyCh <- &handlers.SendResult{
				CorrelationID: correlationID,
				IsSuccess:     true,
				SentMessage:   &tele.Message{ID: rawCount},
			}
		}
		close(requests)
		close(consumeErr)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	dispatchErr := DispatchEdit(ctx, hashe, EditDispatchInput{
		ReceiverID: 42,
		Actor:      Actor{Name: "Alice", ID: 7},
		OldRaw:     oldRaw,
		NewRaw:     newRaw,
		RoutingKey: "telegram.message.send",
	})
	if e.IsNonNil(dispatchErr) {
		t.Fatalf("DispatchEdit: %v", dispatchErr)
	}
	if err := <-consumeErr; err != nil {
		t.Fatalf("consume outgoing requests: %v", err)
	}

	got := make([]telegram.OutgoingRequest, 0, 2)
	for request := range requests {
		got = append(got, request)
	}
	if len(got) != 2 {
		t.Fatalf("raw request count = %d, want 2", len(got))
	}
	want := []struct {
		fileID  string
		caption string
	}{
		{fileID: "old-animation", caption: "old caption"},
		{fileID: "new-animation", caption: "new caption"},
	}
	for i, request := range got {
		if request.Kind != telegram.OutgoingRequestKindRawAPI || request.RawMethod != "sendAnimation" {
			t.Fatalf("request %d envelope = %#v", i, request)
		}
		var payload map[string]any
		if err := json.Unmarshal(request.RawPayload, &payload); err != nil {
			t.Fatalf("request %d payload: %v", i, err)
		}
		if payload["animation"] != want[i].fileID || payload["caption"] != want[i].caption {
			t.Fatalf("request %d payload = %#v", i, payload)
		}
		if payload["chat_id"] != float64(42) {
			t.Fatalf("request %d chat_id = %#v", i, payload["chat_id"])
		}
	}
}

func TestDispatchEditDoesNotPublishWhenReplayMediaHasNoFileID(t *testing.T) {
	publisher, initErr := handlers.NewOutgoingPublisher(handlers.OutgoingConfig{
		Channel:    &amqp.Channel{},
		JobsBuffer: 1,
	})
	if e.IsNonNil(initErr) {
		t.Fatalf("NewOutgoingPublisher: %v", initErr)
	}

	dispatchErr := DispatchEdit(context.Background(), publisher.NewHashe(), EditDispatchInput{
		ReceiverID: 42,
		Actor:      Actor{Name: "Alice", ID: 7},
		OldRaw:     json.RawMessage(`{"animation":{}}`),
		NewRaw:     json.RawMessage(`{"animation":{"file_id":"new-animation"}}`),
		RoutingKey: "telegram.message.send",
	})
	if !e.IsNonNil(dispatchErr) {
		t.Fatal("DispatchEdit must reject media without file_id")
	}
	select {
	case job := <-publisher.Jobs():
		t.Fatalf("unexpected outgoing publication: %#v", job)
	default:
	}
}

type testError struct {
	message string
}

func (err *testError) Error() string {
	return err.message
}

func TestDispatchEditReplaysOldAndNewPollRawEnvelopes(t *testing.T) {
	publisher, initErr := handlers.NewOutgoingPublisher(handlers.OutgoingConfig{
		Channel:    &amqp.Channel{},
		JobsBuffer: 4,
	})
	if e.IsNonNil(initErr) {
		t.Fatalf("NewOutgoingPublisher: %v", initErr)
	}
	hashe := publisher.NewHashe()

	oldRaw := json.RawMessage(`{
		"poll": {
			"question": "Old question?",
			"options": [{"text": "A"}, {"text": "B"}]
		}
	}`)
	newRaw := json.RawMessage(`{
		"poll": {
			"question": "New question?",
			"options": [{"text": "C"}, {"text": "D"}]
		}
	}`)

	requests := make(chan telegram.OutgoingRequest, 2)
	consumeErr := make(chan error, 1)
	go func() {
		rawCount := 0
		for range 4 {
			job := <-publisher.Jobs()
			body := append([]byte(nil), job.Body...)
			correlationID := job.CorrelationID

			var request telegram.OutgoingRequest
			if err := json.Unmarshal(body, &request); err != nil {
				consumeErr <- err
				return
			}
			if request.Kind != telegram.OutgoingRequestKindRawAPI {
				continue
			}
			requests <- request
			rawCount++

			waiter, ok := publisher.Waiters().Load(correlationID)
			if !ok {
				consumeErr <- &testError{"raw request waiter not found"}
				return
			}
			replyCh, ok := waiter.(chan *handlers.SendResult)
			if !ok {
				consumeErr <- &testError{"raw request waiter has unexpected type"}
				return
			}
			replyCh <- &handlers.SendResult{
				CorrelationID: correlationID,
				IsSuccess:     true,
				SentMessage:   &tele.Message{ID: rawCount},
			}
		}
		close(requests)
		close(consumeErr)
	}()

	dispatchErr := DispatchEdit(context.Background(), hashe, EditDispatchInput{
		ReceiverID: 42,
		Actor:      Actor{Name: "Alice", ID: 7},
		OldRaw:     oldRaw,
		NewRaw:     newRaw,
		RoutingKey: "telegram.message.send",
	})
	if e.IsNonNil(dispatchErr) {
		t.Fatalf("DispatchEdit: %v", dispatchErr)
	}
	if err := <-consumeErr; err != nil {
		t.Fatalf("consume outgoing requests: %v", err)
	}

	got := make([]telegram.OutgoingRequest, 0, 2)
	for request := range requests {
		got = append(got, request)
	}
	if len(got) != 2 {
		t.Fatalf("raw request count = %d, want 2", len(got))
	}
	wantQuestions := []string{"Old question?", "New question?"}
	for i, request := range got {
		if request.RawMethod != "sendPoll" {
			t.Fatalf("request %d method = %q, want sendPoll", i, request.RawMethod)
		}
		var payload map[string]any
		if err := json.Unmarshal(request.RawPayload, &payload); err != nil {
			t.Fatalf("request %d payload: %v", i, err)
		}
		if payload["question"] != wantQuestions[i] {
			t.Fatalf("request %d question = %#v", i, payload["question"])
		}
		if _, exists := payload["business_connection_id"]; exists {
			t.Fatalf("request %d must not include business_connection_id", i)
		}
	}
}

func TestDispatchEditReplaysOldAndNewChecklistRawEnvelopes(t *testing.T) {
	publisher, initErr := handlers.NewOutgoingPublisher(handlers.OutgoingConfig{
		Channel:    &amqp.Channel{},
		JobsBuffer: 4,
	})
	if e.IsNonNil(initErr) {
		t.Fatalf("NewOutgoingPublisher: %v", initErr)
	}
	hashe := publisher.NewHashe()

	oldRaw := json.RawMessage(`{
		"business_connection_id": "biz-conn",
		"checklist": {
			"title": "Old list",
			"tasks": [{"id": 1, "text": "Old task"}]
		}
	}`)
	newRaw := json.RawMessage(`{
		"business_connection_id": "biz-conn",
		"checklist": {
			"title": "New list",
			"tasks": [{"id": 1, "text": "New task"}]
		}
	}`)

	requests := make(chan telegram.OutgoingRequest, 2)
	consumeErr := make(chan error, 1)
	go func() {
		rawCount := 0
		for range 4 {
			job := <-publisher.Jobs()
			body := append([]byte(nil), job.Body...)
			correlationID := job.CorrelationID

			var request telegram.OutgoingRequest
			if err := json.Unmarshal(body, &request); err != nil {
				consumeErr <- err
				return
			}
			if request.Kind != telegram.OutgoingRequestKindRawAPI {
				continue
			}
			requests <- request
			rawCount++

			waiter, ok := publisher.Waiters().Load(correlationID)
			if !ok {
				consumeErr <- &testError{"raw request waiter not found"}
				return
			}
			replyCh, ok := waiter.(chan *handlers.SendResult)
			if !ok {
				consumeErr <- &testError{"raw request waiter has unexpected type"}
				return
			}
			replyCh <- &handlers.SendResult{
				CorrelationID: correlationID,
				IsSuccess:     true,
				SentMessage:   &tele.Message{ID: rawCount},
			}
		}
		close(requests)
		close(consumeErr)
	}()

	dispatchErr := DispatchEdit(context.Background(), hashe, EditDispatchInput{
		ReceiverID: 42,
		Actor:      Actor{Name: "Alice", ID: 7},
		OldRaw:     oldRaw,
		NewRaw:     newRaw,
		RoutingKey: "telegram.message.send",
	})
	if e.IsNonNil(dispatchErr) {
		t.Fatalf("DispatchEdit: %v", dispatchErr)
	}
	if err := <-consumeErr; err != nil {
		t.Fatalf("consume outgoing requests: %v", err)
	}

	got := make([]telegram.OutgoingRequest, 0, 2)
	for request := range requests {
		got = append(got, request)
	}
	if len(got) != 2 {
		t.Fatalf("raw request count = %d, want 2", len(got))
	}
	wantTexts := []string{"Old list", "New list"}
	for i, request := range got {
		if request.RawMethod != "sendMessage" {
			t.Fatalf("request %d method = %q, want sendMessage (notification checklist fallback)", i, request.RawMethod)
		}
		var payload map[string]any
		if err := json.Unmarshal(request.RawPayload, &payload); err != nil {
			t.Fatalf("request %d payload: %v", i, err)
		}
		text, _ := payload["text"].(string)
		if !strings.Contains(text, wantTexts[i]) {
			t.Fatalf("request %d text = %#v, want substring %q", i, text, wantTexts[i])
		}
		if !strings.Contains(text, rawmessage.ChecklistOriginNotice) {
			t.Fatalf("request %d text must mention checklist origin", i)
		}
		if _, exists := payload["business_connection_id"]; exists {
			t.Fatalf("request %d must not include business_connection_id", i)
		}
	}
}

func TestDispatchEditReplaysOldAndNewRichMessageRawEnvelopes(t *testing.T) {
	publisher, initErr := handlers.NewOutgoingPublisher(handlers.OutgoingConfig{
		Channel:    &amqp.Channel{},
		JobsBuffer: 4,
	})
	if e.IsNonNil(initErr) {
		t.Fatalf("NewOutgoingPublisher: %v", initErr)
	}
	hashe := publisher.NewHashe()

	oldRaw := json.RawMessage(`{
		"business_connection_id": "biz-conn",
		"rich_message": {
			"blocks": [{"type": "paragraph", "text": "Old rich"}]
		}
	}`)
	newRaw := json.RawMessage(`{
		"business_connection_id": "biz-conn",
		"rich_message": {
			"blocks": [{"type": "paragraph", "text": "New rich"}]
		}
	}`)

	requests := make(chan telegram.OutgoingRequest, 2)
	consumeErr := make(chan error, 1)
	go func() {
		rawCount := 0
		for range 4 {
			job := <-publisher.Jobs()
			body := append([]byte(nil), job.Body...)
			correlationID := job.CorrelationID

			var request telegram.OutgoingRequest
			if err := json.Unmarshal(body, &request); err != nil {
				consumeErr <- err
				return
			}
			if request.Kind != telegram.OutgoingRequestKindRawAPI {
				continue
			}
			requests <- request
			rawCount++

			waiter, ok := publisher.Waiters().Load(correlationID)
			if !ok {
				consumeErr <- &testError{"raw request waiter not found"}
				return
			}
			replyCh, ok := waiter.(chan *handlers.SendResult)
			if !ok {
				consumeErr <- &testError{"raw request waiter has unexpected type"}
				return
			}
			replyCh <- &handlers.SendResult{
				CorrelationID: correlationID,
				IsSuccess:     true,
				SentMessage:   &tele.Message{ID: rawCount},
			}
		}
		close(requests)
		close(consumeErr)
	}()

	dispatchErr := DispatchEdit(context.Background(), hashe, EditDispatchInput{
		ReceiverID: 42,
		Actor:      Actor{Name: "Alice", ID: 7},
		OldRaw:     oldRaw,
		NewRaw:     newRaw,
		RoutingKey: "telegram.message.send",
	})
	if e.IsNonNil(dispatchErr) {
		t.Fatalf("DispatchEdit: %v", dispatchErr)
	}
	if err := <-consumeErr; err != nil {
		t.Fatalf("consume outgoing requests: %v", err)
	}

	got := make([]telegram.OutgoingRequest, 0, 2)
	for request := range requests {
		got = append(got, request)
	}
	if len(got) != 2 {
		t.Fatalf("raw request count = %d, want 2", len(got))
	}
	wantTexts := []string{"Old rich", "New rich"}
	for i, request := range got {
		if request.RawMethod != "sendRichMessage" {
			t.Fatalf("request %d method = %q, want sendRichMessage", i, request.RawMethod)
		}
		var payload map[string]any
		if err := json.Unmarshal(request.RawPayload, &payload); err != nil {
			t.Fatalf("request %d payload: %v", i, err)
		}
		rich, ok := payload["rich_message"].(map[string]any)
		if !ok {
			t.Fatalf("request %d rich_message = %#v", i, payload["rich_message"])
		}
		blocks, ok := rich["blocks"].([]any)
		if !ok || len(blocks) == 0 {
			t.Fatalf("request %d blocks = %#v", i, rich["blocks"])
		}
		block := blocks[0].(map[string]any)
		if block["text"] != wantTexts[i] {
			t.Fatalf("request %d text = %#v, want %q", i, block["text"], wantTexts[i])
		}
	}
}

func TestDispatchDeleteReplaysChecklistRawEnvelope(t *testing.T) {
	publisher, initErr := handlers.NewOutgoingPublisher(handlers.OutgoingConfig{
		Channel:    &amqp.Channel{},
		JobsBuffer: 2,
	})
	if e.IsNonNil(initErr) {
		t.Fatalf("NewOutgoingPublisher: %v", initErr)
	}
	hashe := publisher.NewHashe()

	raw := json.RawMessage(`{
		"business_connection_id": "biz-conn",
		"checklist": {
			"title": "Deleted list",
			"tasks": [{"id": 1, "text": "Task"}]
		}
	}`)

	requests := make(chan telegram.OutgoingRequest, 1)
	consumeErr := make(chan error, 1)
	go func() {
		rawCount := 0
		for range 2 {
			job := <-publisher.Jobs()
			body := append([]byte(nil), job.Body...)
			correlationID := job.CorrelationID

			var request telegram.OutgoingRequest
			if err := json.Unmarshal(body, &request); err != nil {
				consumeErr <- err
				return
			}
			if request.Kind != telegram.OutgoingRequestKindRawAPI {
				continue
			}
			requests <- request
			rawCount++

			waiter, ok := publisher.Waiters().Load(correlationID)
			if !ok {
				consumeErr <- &testError{"raw request waiter not found"}
				return
			}
			replyCh, ok := waiter.(chan *handlers.SendResult)
			if !ok {
				consumeErr <- &testError{"raw request waiter has unexpected type"}
				return
			}
			replyCh <- &handlers.SendResult{
				CorrelationID: correlationID,
				IsSuccess:     true,
				SentMessage:   &tele.Message{ID: rawCount},
			}
		}
		close(requests)
		close(consumeErr)
	}()

	dispatchErr := DispatchDelete(context.Background(), hashe, DeleteDispatchInput{
		ReceiverID: 42,
		Actor:      Actor{Name: "Alice", ID: 7},
		Raw:        raw,
		RoutingKey: "telegram.message.send",
	})
	if e.IsNonNil(dispatchErr) {
		t.Fatalf("DispatchDelete: %v", dispatchErr)
	}
	if err := <-consumeErr; err != nil {
		t.Fatalf("consume outgoing requests: %v", err)
	}

	got := make([]telegram.OutgoingRequest, 0, 1)
	for request := range requests {
		got = append(got, request)
	}
	if len(got) != 1 {
		t.Fatalf("raw request count = %d, want 1", len(got))
	}
	if got[0].RawMethod != "sendMessage" {
		t.Fatalf("method = %q, want sendMessage (notification checklist fallback)", got[0].RawMethod)
	}
}

func TestDispatchDeleteReplaysPollRawEnvelope(t *testing.T) {
	publisher, initErr := handlers.NewOutgoingPublisher(handlers.OutgoingConfig{
		Channel:    &amqp.Channel{},
		JobsBuffer: 2,
	})
	if e.IsNonNil(initErr) {
		t.Fatalf("NewOutgoingPublisher: %v", initErr)
	}
	hashe := publisher.NewHashe()

	raw := json.RawMessage(`{
		"poll": {
			"question": "Deleted poll?",
			"options": [{"text": "Yes"}, {"text": "No"}]
		}
	}`)

	requests := make(chan telegram.OutgoingRequest, 1)
	consumeErr := make(chan error, 1)
	go func() {
		rawCount := 0
		for range 2 {
			job := <-publisher.Jobs()
			body := append([]byte(nil), job.Body...)
			correlationID := job.CorrelationID

			var request telegram.OutgoingRequest
			if err := json.Unmarshal(body, &request); err != nil {
				consumeErr <- err
				return
			}
			if request.Kind != telegram.OutgoingRequestKindRawAPI {
				continue
			}
			requests <- request
			rawCount++

			waiter, ok := publisher.Waiters().Load(correlationID)
			if !ok {
				consumeErr <- &testError{"raw request waiter not found"}
				return
			}
			replyCh, ok := waiter.(chan *handlers.SendResult)
			if !ok {
				consumeErr <- &testError{"raw request waiter has unexpected type"}
				return
			}
			replyCh <- &handlers.SendResult{
				CorrelationID: correlationID,
				IsSuccess:     true,
				SentMessage:   &tele.Message{ID: rawCount},
			}
		}
		close(requests)
		close(consumeErr)
	}()

	dispatchErr := DispatchDelete(context.Background(), hashe, DeleteDispatchInput{
		ReceiverID: 42,
		Actor:      Actor{Name: "Alice", ID: 7},
		Raw:        raw,
		RoutingKey: "telegram.message.send",
	})
	if e.IsNonNil(dispatchErr) {
		t.Fatalf("DispatchDelete: %v", dispatchErr)
	}
	if err := <-consumeErr; err != nil {
		t.Fatalf("consume outgoing requests: %v", err)
	}

	got := make([]telegram.OutgoingRequest, 0, 1)
	for request := range requests {
		got = append(got, request)
	}
	if len(got) != 1 {
		t.Fatalf("raw request count = %d, want 1", len(got))
	}
	if got[0].RawMethod != "sendPoll" {
		t.Fatalf("method = %q, want sendPoll", got[0].RawMethod)
	}
}
