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

func TestEmitAlbumPublishesEnvelope(t *testing.T) {
	jobs := make(chan *publishEnvelope, 1)
	waiters := &sync.Map{}
	hashe := HandlerChainHashe{}.Init(jobs, waiters, "run-1")

	album := &telegram.MediaGroup{
		Chat: &tele.Chat{ID: 777},
		Messages: []*tele.Message{
			{Photo: &tele.Photo{}},
		},
	}

	if err := hashe.EmitAlbum("telegram.message.send", album); e.IsNonNil(err) {
		t.Fatalf("emit album: %v", err)
	}

	job := <-jobs
	if job.routingKey != "telegram.message.send" {
		t.Fatalf("expected routing key %q, got %q", "telegram.message.send", job.routingKey)
	}

	var request telegram.OutgoingRequest
	if err := json.Unmarshal(job.body, &request); err != nil {
		t.Fatalf("unmarshal outgoing request: %v", err)
	}
	if request.Kind != telegram.OutgoingRequestKindAlbum {
		t.Fatalf("expected kind %q, got %q", telegram.OutgoingRequestKindAlbum, request.Kind)
	}
	if request.Album == nil || len(request.Album.Messages) != 1 {
		t.Fatalf("expected 1 album message, got %#v", request.Album)
	}
}

func TestEmitAlbumWaitReturnsSentAlbum(t *testing.T) {
	jobs := make(chan *publishEnvelope, 1)
	waiters := &sync.Map{}
	hashe := HandlerChainHashe{}.Init(jobs, waiters, "run-2")

	want := []*tele.Message{
		{ID: 11},
		{ID: 12},
	}

	go func() {
		job := <-jobs
		value, ok := waiters.Load(job.correlationID)
		if !ok {
			t.Errorf("waiter for correlation %q not found", job.correlationID)
			return
		}
		replyCh, ok := value.(chan *SendResult)
		if !ok {
			t.Errorf("unexpected waiter type %T", value)
			return
		}
		replyCh <- &SendResult{
			CorrelationID: job.correlationID,
			IsSuccess:     true,
			SentAlbum:     want,
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := hashe.EmitAlbumWait(ctx, "telegram.message.send", &telegram.MediaGroup{
		Chat: &tele.Chat{ID: 777},
		Messages: []*tele.Message{
			{Photo: &tele.Photo{}},
		},
	})
	if e.IsNonNil(err) {
		t.Fatalf("emit album wait: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d sent messages, got %d", len(want), len(got))
	}
	if got[0].ID != want[0].ID || got[1].ID != want[1].ID {
		t.Fatalf("unexpected sent album: %#v", got)
	}
}
