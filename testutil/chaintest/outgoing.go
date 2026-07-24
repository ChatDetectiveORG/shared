package chaintest

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	h "github.com/ChatDetectiveORG/shared/handlers"
	"github.com/ChatDetectiveORG/shared/telegram"
	tele "gopkg.in/telebot.v4"
)

// OutgoingCapture records outgoing publish jobs and satisfies EmitWait replies.
type OutgoingCapture struct {
	Jobs    chan *h.PublishEnvelope
	Waiters *sync.Map

	t         *testing.T
	autoReply bool
	stop      chan struct{}
	done      chan struct{}

	mu       sync.Mutex
	recorded []telegram.OutgoingRequest
}

// NewOutgoingCapture creates a buffered capture with a background consumer loop.
func NewOutgoingCapture(t *testing.T, buffer int) *OutgoingCapture {
	t.Helper()
	if buffer <= 0 {
		buffer = 32
	}

	c := &OutgoingCapture{
		Jobs:      make(chan *h.PublishEnvelope, buffer),
		Waiters:   &sync.Map{},
		t:         t,
		autoReply: true,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	go c.loop()
	t.Cleanup(func() {
		close(c.stop)
		<-c.done
	})
	return c
}

func (c *OutgoingCapture) loop() {
	defer close(c.done)
	for {
		select {
		case <-c.stop:
			return
		case job, ok := <-c.Jobs:
			if !ok {
				return
			}
			if job == nil {
				continue
			}
			c.record(job)
			if c.autoReply {
				c.replyWaiter(job)
			}
		}
	}
}

func (c *OutgoingCapture) record(job *h.PublishEnvelope) {
	var request telegram.OutgoingRequest
	if err := json.Unmarshal(job.Body, &request); err != nil {
		c.t.Fatalf("chaintest: unmarshal outgoing request: %v", err)
	}
	c.mu.Lock()
	c.recorded = append(c.recorded, request)
	c.mu.Unlock()
}

func (c *OutgoingCapture) replyWaiter(job *h.PublishEnvelope) {
	value, ok := c.Waiters.Load(job.CorrelationID)
	if !ok {
		return
	}
	replyCh, ok := value.(chan *h.SendResult)
	if !ok {
		c.t.Fatalf("chaintest: unexpected waiter type %T", value)
		return
	}
	replyCh <- &h.SendResult{
		CorrelationID: job.CorrelationID,
		IsSuccess:     true,
		SentMessage:   &tele.Message{ID: 1, Chat: &tele.Chat{ID: 1}},
	}
}

// Snapshot returns recorded requests so far.
func (c *OutgoingCapture) Snapshot() []telegram.OutgoingRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]telegram.OutgoingRequest, len(c.recorded))
	copy(out, c.recorded)
	return out
}

// Collect waits until the recorded request count is stable, then returns a snapshot.
func (c *OutgoingCapture) Collect(idle time.Duration) []telegram.OutgoingRequest {
	c.t.Helper()
	if idle <= 0 {
		idle = 50 * time.Millisecond
	}
	last := -1
	stableSince := time.Now()
	for {
		n := len(c.Snapshot())
		if n != last {
			last = n
			stableSince = time.Now()
		} else if time.Since(stableSince) >= idle {
			return c.Snapshot()
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// DecodeOutgoing unmarshals a single publish envelope.
func DecodeOutgoing(t *testing.T, job *h.PublishEnvelope) telegram.OutgoingRequest {
	t.Helper()
	var request telegram.OutgoingRequest
	if err := json.Unmarshal(job.Body, &request); err != nil {
		t.Fatalf("chaintest: unmarshal outgoing request: %v", err)
	}
	return request
}
