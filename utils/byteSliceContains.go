package utils

import "bytes"

func ByteSliceContains(slice [][]byte, item []byte) bool {
	for _, v := range slice {
		if bytes.Equal(v, item) {
			return true
		}
	}

	return false
}