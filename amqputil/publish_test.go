package amqputil

import (
	"errors"
	"testing"
)

func TestIsRecoverableAMQPError(t *testing.T) {
	if !IsRecoverableAMQPError(errors.New(`Exception (504) Reason: "channel/connection is not open"`)) {
		t.Fatal("expected 504 to be recoverable")
	}
	if IsRecoverableAMQPError(errors.New("access refused")) {
		t.Fatal("expected access refused to be non-recoverable")
	}
}
