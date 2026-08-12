package main

import (
	"encoding/hex"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashBcrypt(t *testing.T) {
	hash, err := hashBcrypt("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Errorf("hash prefix: %s", hash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("s3cret")); err != nil {
		t.Errorf("bcrypt mismatch: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrong")); err == nil {
		t.Error("wrong password validated")
	}
}

func TestGenerateRandomSecret(t *testing.T) {
	a := generateRandomSecret()
	b := generateRandomSecret()
	if len(a) != 64 {
		t.Errorf("length %d, want 64 hex chars", len(a))
	}
	if _, err := hex.DecodeString(a); err != nil {
		t.Errorf("not hex: %v", err)
	}
	if a == b {
		t.Error("two secrets equal")
	}
}

func TestGenerateAgentSecret(t *testing.T) {
	a := generateAgentSecret()
	b := generateAgentSecret()
	if a == b || len(a) != 44 {
		t.Errorf("agent secret wrong: %q", a)
	}
}

func TestSecureCompare(t *testing.T) {
	if !secureCompare("abc", "abc") {
		t.Error("equal strings rejected")
	}
	if secureCompare("abc", "abd") {
		t.Error("different strings accepted")
	}
}
