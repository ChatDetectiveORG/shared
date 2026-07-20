package utils

import (
	"strings"
	"testing"
)

func TestValidateMasterKeyAcceptsAESKeyLengths(t *testing.T) {
	for _, size := range []int{16, 24, 32} {
		key := []byte(strings.Repeat("k", size))
		if err := ValidateMasterKey(key); !err.IsNil() {
			t.Fatalf("key of %d bytes should be valid: %s", size, err.JSON())
		}
	}
}

func TestValidateMasterKeyRejectsInvalidLengths(t *testing.T) {
	for _, size := range []int{0, 1, 15, 17, 31, 33, 64} {
		key := []byte(strings.Repeat("k", size))
		if err := ValidateMasterKey(key); err.IsNil() {
			t.Fatalf("key of %d bytes should be rejected", size)
		}
	}
}
