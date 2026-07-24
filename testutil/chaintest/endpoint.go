package chaintest

import (
	"sync"
	"testing"

	e "github.com/ChatDetectiveORG/shared/errors"
	h "github.com/ChatDetectiveORG/shared/handlers"
	tele "gopkg.in/telebot.v4"
)

// RunEndpoint executes a handler endpoint synchronously with injected outgoing channels.
func RunEndpoint(t *testing.T, ep h.Endpoint, update tele.Update, capture *OutgoingCapture, mirrorID string) {
	t.Helper()
	if err := runEndpoint(ep, update, capture, mirrorID); !err.IsNil() {
		t.Fatalf("chaintest: endpoint %q failed: %s", ep.Name, err.JSON())
	}
}

// RunEndpointExpectError runs the endpoint and returns the error when the handler fails.
func RunEndpointExpectError(t *testing.T, ep h.Endpoint, update tele.Update, capture *OutgoingCapture, mirrorID string) *e.ErrorInfo {
	t.Helper()
	return runEndpoint(ep, update, capture, mirrorID)
}

func runEndpoint(ep h.Endpoint, update tele.Update, capture *OutgoingCapture, mirrorID string) *e.ErrorInfo {
	var wg sync.WaitGroup
	return ep.RunForTest(update, capture.Jobs, capture.Waiters, mirrorID, &wg)
}
