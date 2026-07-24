package chaintest_test

import (
	"context"
	"testing"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	h "github.com/ChatDetectiveORG/shared/handlers"
	"github.com/ChatDetectiveORG/shared/testutil/chaintest"
	tele "gopkg.in/telebot.v4"
)

func TestOutgoingCaptureAutoReply(t *testing.T) {
	capture := chaintest.NewOutgoingCapture(t, 2)
	ep := h.Endpoint{}
	ep.Init(
		"echo",
		*h.HandlerChain{}.Init(time.Second, h.InitChainHandler(func(u tele.Update, hashe *h.HandlerChainHashe) *e.ErrorInfo {
			_ = u
			_, err := hashe.EmitWait(context.Background(), "telegram.message.send", &tele.Message{
				Chat: &tele.Chat{ID: 1},
				Text: "hello",
			})
			return err
		}, h.EndOnError)),
		nil,
	)

	chaintest.RunEndpoint(t, ep, tele.Update{ID: 1}, capture, "")
	requests := capture.Collect(200 * time.Millisecond)
	if len(requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(requests))
	}
	if requests[0].Message == nil || requests[0].Message.Text != "hello" {
		t.Fatalf("unexpected request: %#v", requests[0])
	}
}
