package types

import (
	"crypto/sha256"
	"fmt"
)

// InitialRegistrationMessages returns the two fixed messages a validator signs
// once to register an EVM key.
func InitialRegistrationMessages(operatorAddress string) (msgA, msgB string) {
	msgA = fmt.Sprintf("TellorLayer: Initial bridge signature A for operator %s", operatorAddress)
	msgB = fmt.Sprintf("TellorLayer: Initial bridge signature B for operator %s", operatorAddress)
	return msgA, msgB
}

// InitialRegistrationDigests is sha256 of each registration message. The local
// keyring signs these bytes; recovery hashes them once more (cosmos secp256k1
// hashes the payload again).
func InitialRegistrationDigests(operatorAddress string) (hashA, hashB [32]byte) {
	msgA, msgB := InitialRegistrationMessages(operatorAddress)
	return sha256.Sum256([]byte(msgA)), sha256.Sum256([]byte(msgB))
}
