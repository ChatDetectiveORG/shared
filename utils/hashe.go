package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Unique, non-decodable int64 hash generator
func Int64ToHash(id int64) string {
	data := []byte(fmt.Sprintf("%d", id))

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:32])
}
