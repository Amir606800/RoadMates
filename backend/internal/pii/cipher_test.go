package pii

import (
	"bytes"
	"strings"
	"testing"
)

func TestPINEncryption(t *testing.T) {
	cipher, err := New(bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal("could not initialize cipher")
	}
	first, err := cipher.EncryptPIN("ABC1234", "user-1", "application-1")
	if err != nil {
		t.Fatal("encryption failed")
	}
	second, _ := cipher.EncryptPIN("ABC1234", "user-1", "application-1")
	if bytes.Equal(first, second) || bytes.Contains(first, []byte("ABC1234")) {
		t.Fatal("ciphertext is not randomized or contains plaintext")
	}
	plain, err := cipher.DecryptPIN(first, "user-1", "application-1")
	if err != nil || plain != "ABC1234" {
		t.Fatal("PIN roundtrip failed")
	}
	for _, ids := range [][2]string{{"user-2", "application-1"}, {"user-1", "application-2"}} {
		if _, err := cipher.DecryptPIN(first, ids[0], ids[1]); err == nil {
			t.Fatal("ciphertext was accepted for another application")
		}
	}
	tampered := append([]byte(nil), first...)
	tampered[len(tampered)-1] ^= 1
	for _, invalid := range [][]byte{nil, first[:4], tampered, append([]byte{2}, first[1:]...)} {
		if _, err := cipher.DecryptPIN(invalid, "user-1", "application-1"); err == nil {
			t.Fatal("invalid ciphertext accepted")
		}
	}
	other, _ := New(bytes.Repeat([]byte{2}, 32))
	if _, err := other.DecryptPIN(first, "user-1", "application-1"); err == nil {
		t.Fatal("wrong encryption key accepted")
	}
	if _, err := New([]byte(strings.Repeat("k", 31))); err == nil {
		t.Fatal("short key accepted")
	}
}
