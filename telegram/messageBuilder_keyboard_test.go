package telegram

import (
	"strings"
	"testing"

	"github.com/ChatDetectiveORG/shared/utils"
)

func TestParseKeyboardPageDelta(t *testing.T) {
	tests := []struct {
		name       string
		callback   string
		wantDelta  int
		wantOK     bool
	}{
		{name: "empty callback", callback: "", wantDelta: 0, wantOK: true},
		{name: "no query string", callback: "mirror_list", wantDelta: 0, wantOK: true},
		{name: "missing pageDelta", callback: "mirrors?page=1", wantDelta: 0, wantOK: true},
		{name: "forward", callback: "mirrors?pageDelta=1", wantDelta: 1, wantOK: true},
		{name: "back", callback: "mirrors?pageDelta=-1", wantDelta: -1, wantOK: true},
		{name: "invalid", callback: "mirrors?pageDelta=abc", wantDelta: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta, ok := parseKeyboardPageDelta(tt.callback)
			if ok != tt.wantOK || delta != tt.wantDelta {
				t.Fatalf("parseKeyboardPageDelta(%q) = (%d, %v), want (%d, %v)",
					tt.callback, delta, ok, tt.wantDelta, tt.wantOK)
			}
		})
	}
}

func TestNavigationCallbackDataFormat(t *testing.T) {
	data := utils.DumpCallbackData("my_page", map[string]any{"pageDelta": 1})
	if !strings.Contains(data, "pageDelta=1") {
		t.Fatalf("callback data = %q, want pageDelta=1", data)
	}
	if !strings.HasPrefix(data, "my_page?") {
		t.Fatalf("callback data = %q, want my_page? prefix", data)
	}

	delta, ok := parseKeyboardPageDelta(data)
	if !ok || delta != 1 {
		t.Fatalf("parsed delta = (%d, %v), want (1, true)", delta, ok)
	}
}
