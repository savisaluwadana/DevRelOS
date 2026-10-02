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

func TestAssociatedDataFormatIsStable(t *testing.T) {
	// Existing ciphertexts were sealed with this exact associated data. Changing
	// the format silently makes every stored secret undecryptable, and
	// KeyVersion is a rotation counter rather than a format version, so there is
	// no discriminator with which to migrate.
	got := string(AssociatedData("ws1", "github.issues", "token", 3))
	if want := "devrelos|ws1|github.issues|token|v3"; got != want {
		t.Fatalf("associated data format changed:\n got %q\nwant %q", got, want)
	}
}

func TestAssociatedDataBindsEveryField(t *testing.T) {
	base := AssociatedData("ws1", "github.issues", "token", 1)
	for _, variant := range [][]byte{
		AssociatedData("ws2", "github.issues", "token", 1),
		AssociatedData("ws1", "rss", "token", 1),
		AssociatedData("ws1", "github.issues", "other", 1),
		AssociatedData("ws1", "github.issues", "token", 2),
	} {
		if string(variant) == string(base) {
			t.Fatalf("associated data did not change: %q", variant)
		}
	}
}

func TestValidateAADFieldRejectsTheDelimiter(t *testing.T) {
	// Without this guard, provider "github.issues" + name "a|b" and provider
	// "github.issues|a" + name "b" produce identical associated data, so a
	// ciphertext sealed for one decrypts as the other.
	ambiguousA := AssociatedData("ws", "github.issues", "a|b", 1)
	ambiguousB := AssociatedData("ws", "github.issues|a", "b", 1)
	if string(ambiguousA) != string(ambiguousB) {
		t.Fatal("test premise is wrong: these were expected to collide")
	}

	if err := ValidateAADField("a|b"); err == nil {
		t.Fatal("a value containing the delimiter was accepted")
	}
	if err := ValidateAADField("github.issues|a"); err == nil {
		t.Fatal("a provider containing the delimiter was accepted")
	}
	for _, ok := range []string{"github.issues", "token", "my secret name", "a-b_c.d"} {
		if err := ValidateAADField(ok); err != nil {
			t.Errorf("ValidateAADField(%q) = %v, want nil", ok, err)
		}
	}
}

func TestDecryptRejectsMismatchedAssociatedData(t *testing.T) {
	box, err := NewSecretBox(base64.StdEncoding.EncodeToString(make([]byte, MasterKeyBytes)))
	if err != nil {
		t.Fatal(err)
	}
	aad := AssociatedData("ws1", "github.issues", "token", 1)
	ciphertext, nonce, err := box.Encrypt([]byte("ghp_secret"), aad)
	if err != nil {
		t.Fatal(err)
	}

	// The same ciphertext must not open under another workspace's binding.
	other := AssociatedData("ws2", "github.issues", "token", 1)
	if _, err := box.Decrypt(ciphertext, nonce, other); err == nil {
		t.Fatal("a ciphertext moved to another workspace decrypted")
	}
	plaintext, err := box.Decrypt(ciphertext, nonce, aad)
	if err != nil || string(plaintext) != "ghp_secret" {
		t.Fatalf("round trip failed: %v %q", err, plaintext)
	}
}
