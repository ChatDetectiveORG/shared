package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	e "github.com/ChatDetectiveORG/shared/errors"
)

// Makes a hash from any integer or string
func ToHash(id interface{}) string {
	data := []byte(fmt.Sprintf("%d", id))

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:32])
}

func ToSecureHash(id interface{}) (string, *e.ErrorInfo) {
	masterKey, err := GetMasterkey()
	if e.IsNonNil(err) {
		return "", e.FromError(err, "failed to get master key")
	}

	return ToHash(string(masterKey) + "|" + fmt.Sprintf("%d", id)), e.Nil()
}
