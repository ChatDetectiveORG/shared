package chaintest

import (
	"strings"
	"testing"

	"github.com/ChatDetectiveORG/shared/telegram"
)

// AssertAnyTextSubstr fails when no captured message contains all substrings.
func AssertAnyTextSubstr(t *testing.T, requests []telegram.OutgoingRequest, parts ...string) {
	t.Helper()
	for _, request := range requests {
		if request.Message == nil {
			continue
		}
		text := request.Message.Text
		if text == "" && request.Message.Caption != "" {
			text = request.Message.Caption
		}
		if text == "" {
			continue
		}
		ok := true
		for _, part := range parts {
			if !strings.Contains(text, part) {
				ok = false
				break
			}
		}
		if ok {
			return
		}
	}
	t.Fatalf("no outgoing message contains all parts %v in %#v", parts, requests)
}

// CountRawAPI returns the number of raw_api outgoing requests.
func CountRawAPI(requests []telegram.OutgoingRequest) int {
	n := 0
	for _, request := range requests {
		if request.Kind == telegram.OutgoingRequestKindRawAPI {
			n++
		}
	}
	return n
}

// CountRawMethod returns raw_api requests with the given Bot API method.
func CountRawMethod(requests []telegram.OutgoingRequest, method string) int {
	n := 0
	for _, request := range requests {
		if request.Kind == telegram.OutgoingRequestKindRawAPI && request.RawMethod == method {
			n++
		}
	}
	return n
}
