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

func TestEmitBuiltPublishesResolvedMessage(t *testing.T) {
	jobs := make(chan *publishEnvelope, 1)
	waiters := &sync.Map{}
	hashe := HandlerChainHashe{}.Init(jobs, waiters, "run-built")

	builder := &telegram.MessageBuilder{Mdv2Enabled: true}
	builder.WriteString("caption")
	builder.AddMirrorFile(telegram.MirrorFileAsset{
		PrimaryFileID: "primary-id",
		FallbackPath:  "static/photo.png",
		MimeType:      "image/png",
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := hashe.EmitBuilt(ctx, nil, "telegram.message.send", 42, builder); e.IsNonNil(err) {
		t.Fatalf("emit built: %v", err)
	}

	job := <-jobs
	var request telegram.OutgoingRequest
	if err := json.Unmarshal(job.body, &request); err != nil {
		t.Fatalf("unmarshal outgoing request: %v", err)
	}
	if request.Message == nil || request.Message.Photo == nil {
		t.Fatalf("expected photo message, got %#v", request.Message)
	}
	if request.Message.Photo.File.FileID != "primary-id" {
		t.Fatalf("file id = %q, want primary-id", request.Message.Photo.File.FileID)
	}
	if request.Message.Photo.File.FileLocal != "static/photo.png" {
		t.Fatalf("fallback path missing: %#v", request.Message.Photo.File)
	}
	if request.Message.Caption != "caption" {
		t.Fatalf("caption = %q", request.Message.Caption)
	}
}

func TestEmitBuiltMirrorUsesEmitWait(t *testing.T) {
	jobs := make(chan *publishEnvelope, 1)
	waiters := &sync.Map{}
	hashe := HandlerChainHashe{}.Init(jobs, waiters, "run-built-mirror", "1")

	builder := &telegram.MessageBuilder{}
	builder.AddMirrorFile(telegram.MirrorFileAsset{
		FallbackPath:  "static/setupInstruction.gif",
		MimeType:      "image/gif",
		MirrorFileKey: "installation_animation",
	})

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
			SentMessage: &tele.Message{
				Animation: &tele.Animation{File: tele.File{FileID: "fresh-id"}},
			},
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := hashe.EmitBuilt(ctx, nil, "telegram.message.send", 42, builder); e.IsNonNil(err) {
		t.Fatalf("emit built mirror: %v", err)
	}
}
