package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/ChatDetectiveORG/shared/telegram"

	tele "gopkg.in/telebot.v4"
)

func TestOutgoingPublisherNewHasheEmit(t *testing.T) {
	jobs := make(chan *PublishEnvelope, 1)
	waiters := &sync.Map{}
	pub := &OutgoingPublisher{
		jobs:    jobs,
		waiters: waiters,
		podID:   "test-pod",
	}

	hashe := pub.NewHashe()
	if err := hashe.Emit("telegram.message.send", &tele.Message{
		Chat: &tele.Chat{ID: 42},
		Text: "hello",
	}); e.IsNonNil(err) {
		t.Fatalf("emit: %v", err)
	}

	job := <-jobs
	var request telegram.OutgoingRequest
	if err := json.Unmarshal(job.Body, &request); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if request.Message == nil || request.Message.Chat.ID != 42 {
		t.Fatalf("unexpected message: %#v", request.Message)
	}
}

func TestOutgoingPublisherEmitWait(t *testing.T) {
	jobs := make(chan *PublishEnvelope, 1)
	waiters := &sync.Map{}
	pub := &OutgoingPublisher{
		jobs:    jobs,
		waiters: waiters,
		podID:   "test-pod",
	}

	hashe := pub.NewHashe()
	want := &tele.Message{ID: 99, Chat: &tele.Chat{ID: 42}, Text: "sent"}

	go func() {
		job := <-jobs
		value, ok := waiters.Load(job.CorrelationID)
		if !ok {
			t.Errorf("waiter missing for %q", job.CorrelationID)
			return
		}
		replyCh := value.(chan *SendResult)
		replyCh <- &SendResult{
			CorrelationID: job.CorrelationID,
			IsSuccess:     true,
			SentMessage:   want,
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := hashe.EmitWait(ctx, "telegram.message.send", &tele.Message{
		Chat: &tele.Chat{ID: 42},
		Text: "hello",
	})
	if e.IsNonNil(err) {
		t.Fatalf("emit wait: %v", err)
	}
	if got == nil || got.ID != want.ID {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
