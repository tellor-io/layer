package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func CalculateCommitment(move, salt string) string {
	h := sha256.Sum256([]byte(move + ":" + salt))
	return hex.EncodeToString(h[:])
}

func Salt(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
