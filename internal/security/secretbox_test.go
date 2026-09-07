package security

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestSecretBoxRoundTripAndAAD(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, MasterKeyBytes))
	box, err := NewSecretBox(key)
	if err != nil {
		t.Fatalf("new secret box: %v", err)
	}
	aad := AssociatedData("workspace", "github", "primary", 1)
	ciphertext, nonce, err := box.Encrypt([]byte("super-secret-token"), aad)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Contains(ciphertext, []byte("super-secret-token")) {
		t.Fatal("ciphertext contains plaintext")
	}
	plaintext, err := box.Decrypt(ciphertext, nonce, aad)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plaintext) != "super-secret-token" {
		t.Fatalf("unexpected plaintext %q", string(plaintext))
	}
	if _, err := box.Decrypt(ciphertext, nonce, AssociatedData("other", "github", "primary", 1)); err == nil {
		t.Fatal("expected associated-data mismatch to fail")
	}
}

func TestSecretBoxRejectsWrongKeySize(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("too-short"))
	if _, err := NewSecretBox(key); err == nil {
		t.Fatal("expected invalid key size error")
	}
}
