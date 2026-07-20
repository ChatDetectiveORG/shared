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

func Encrypt(plaintext []byte, key []byte) ([]byte, *e.ErrorInfo) {
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

func Decrypt(ciphertext []byte, key []byte) ([]byte, *e.ErrorInfo) {
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

func GetMasterkey() ([]byte, *e.ErrorInfo) {
	masterKey := []byte(os.Getenv("MASTER_KEY"))
	if err := ValidateMasterKey(masterKey); e.IsNonNil(err) {
		return nil, err
	}

	return masterKey, e.Nil()
}

// ValidateMasterKey checks that the master key is present and has a valid AES key length.
// Call it at service startup so misconfiguration fails fast instead of at first decrypt.
func ValidateMasterKey(masterKey []byte) *e.ErrorInfo {
	if len(masterKey) == 0 {
		return e.FromError(errors.New("master key is not set"), "master key is not set").WithSeverity(e.Critical)
	}
	switch len(masterKey) {
	case 16, 24, 32:
		return e.Nil()
	default:
		return e.FromError(
			errors.New("MASTER_KEY must be exactly 16, 24 or 32 bytes (AES-128/192/256)"),
			"master key has invalid length",
		).WithSeverity(e.Critical)
	}
}

// ValidateMasterKeyFromEnv validates the MASTER_KEY environment variable.
func ValidateMasterKeyFromEnv() *e.ErrorInfo {
	return ValidateMasterKey([]byte(os.Getenv("MASTER_KEY")))
}

func DecryptUserKey(key []byte) ([]byte, *e.ErrorInfo) {
	masterKey, err := GetMasterkey()
	if e.IsNonNil(err) {
		return nil, err
	}

	key, err = Decrypt(key, masterKey)
	if e.IsNonNil(err) {
		return nil, e.FromError(err, "failed to decrypt data encryption key").WithSeverity(e.Notice)
	}

	return key, e.Nil()
}
