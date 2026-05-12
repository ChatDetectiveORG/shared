package ratelimit

import (
	"github.com/ChatDetectiveORG/shared/telegram"
)

// PriorityFromOutgoingRequest maps an OutgoingRequest's Priority field to a ratelimit.Priority.
// Producers should set Priority to one of the documented levels (1..4); zero falls through to
// DefaultPriority. We resolve here so producers don't need to import this package.
func PriorityFromOutgoingRequest(r *telegram.OutgoingRequest) Priority {
	if r == nil {
		return DefaultPriority
	}
	return Resolve(r.Priority)
}
