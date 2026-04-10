package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Makes a hash from any integer or string
func ToHash(id interface{}) string {
	data := []byte(fmt.Sprintf("%d", id))

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:32])
}
