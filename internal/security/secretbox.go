package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const MasterKeyBytes = 32

type SecretBox struct {
	aead cipher.AEAD
}

func NewSecretBox(encodedKey string) (*SecretBox, error) {
	encodedKey = strings.TrimSpace(encodedKey)
	if encodedKey == "" {
		return nil, errors.New("secret encryption key is not configured")
	}
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(encodedKey)
	}
	if err != nil {
		return nil, fmt.Errorf("decode secret encryption key: %w", err)
	}
	if len(key) != MasterKeyBytes {
		return nil, fmt.Errorf("secret encryption key must decode to %d bytes", MasterKeyBytes)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}
	return &SecretBox{aead: aead}, nil
}

func (b *SecretBox) Encrypt(plaintext []byte, associatedData []byte) (ciphertext, nonce []byte, err error) {
	if b == nil || b.aead == nil {
		return nil, nil, errors.New("secret box is unavailable")
	}
	if len(plaintext) == 0 {
		return nil, nil, errors.New("secret value is empty")
	}
	nonce = make([]byte, b.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext = b.aead.Seal(nil, nonce, plaintext, associatedData)
	return ciphertext, nonce, nil
}

func (b *SecretBox) Decrypt(ciphertext, nonce, associatedData []byte) ([]byte, error) {
	if b == nil || b.aead == nil {
		return nil, errors.New("secret box is unavailable")
	}
	if len(nonce) != b.aead.NonceSize() {
		return nil, errors.New("invalid secret nonce")
	}
	plaintext, err := b.aead.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		return nil, errors.New("secret decryption failed")
	}
	return plaintext, nil
}

func AssociatedData(workspaceID, provider, name string, keyVersion int) []byte {
	return []byte(fmt.Sprintf("devrelos|%s|%s|%s|v%d", workspaceID, provider, name, keyVersion))
}
