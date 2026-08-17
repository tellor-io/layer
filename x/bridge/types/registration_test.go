package types

import (
	"crypto/sha256"
	"testing"
)

func TestInitialRegistrationMessages(t *testing.T) {
	msgA, msgB := InitialRegistrationMessages("tellorvaloper1test")
	if msgA != "TellorLayer: Initial bridge signature A for operator tellorvaloper1test" {
		t.Fatalf("msgA = %q", msgA)
	}
	if msgB != "TellorLayer: Initial bridge signature B for operator tellorvaloper1test" {
		t.Fatalf("msgB = %q", msgB)
	}
}

func TestInitialRegistrationDigests(t *testing.T) {
	msgA, msgB := InitialRegistrationMessages("tellorvaloper1test")
	hashA, hashB := InitialRegistrationDigests("tellorvaloper1test")
	wantA := sha256.Sum256([]byte(msgA))
	wantB := sha256.Sum256([]byte(msgB))
	if hashA != wantA {
		t.Fatalf("hashA = %x, want %x", hashA, wantA)
	}
	if hashB != wantB {
		t.Fatalf("hashB = %x, want %x", hashB, wantB)
	}
}
