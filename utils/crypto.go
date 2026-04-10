package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"

	e "github.com/ChatDetectiveORG/shared/errors"
)

func Encrypt(plaintext []byte, key []byte) ([]byte, e.ErrorInfo) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, e.FromError(err, "failed to create new cipher").WithSeverity(e.Critical)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, e.FromError(err, "failed to create new GCM").WithSeverity(e.Critical)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, e.FromError(err, "failed to read full random reader").WithSeverity(e.Critical)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), e.Nil()
}

func Decrypt(ciphertext []byte, key []byte) ([]byte, e.ErrorInfo) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, e.FromError(err, "failed to create new cipher").WithSeverity(e.Critical)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, e.FromError(err, "failed to create new GCM").WithSeverity(e.Critical)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, e.FromError(errors.New("ciphertext too short"), "ciphertext too short").WithSeverity(e.Critical)
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	res, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, e.FromError(err, "failed to open GCM").WithSeverity(e.Critical)
	}
	return res, e.Nil()
}

func GetMasterkey() ([]byte, e.ErrorInfo) {
	masterKey := []byte(os.Getenv("MASTER_KEY"))
	if len(masterKey) == 0 {
		return nil, e.FromError(errors.New("master key is not set"), "master key is not set").WithSeverity(e.Critical)
	}

	return masterKey, e.Nil()
}
