package types

import "testing"

func TestInitialRegistrationMessages(t *testing.T) {
	msgA, msgB := InitialRegistrationMessages("tellorvaloper1test")
	if msgA != "TellorLayer: Initial bridge signature A for operator tellorvaloper1test" {
		t.Fatalf("msgA = %q", msgA)
	}
	if msgB != "TellorLayer: Initial bridge signature B for operator tellorvaloper1test" {
		t.Fatalf("msgB = %q", msgB)
	}
}
