package utils

import (
	"encoding/hex"
	"strings"

	"errors"

	"github.com/ethereum/go-ethereum/crypto"
)

func QueryIDFromData(queryData []byte) []byte {
	return crypto.Keccak256(queryData)
}

func QueryIDFromDataString(queryData string) ([]byte, error) {
	bz, err := QueryBytesFromString(queryData)
	if err != nil {
		return nil, err
	}

	return QueryIDFromData(bz), nil
}

// converts a hex string to bytes (query data or query ID)
func QueryBytesFromString(query string) ([]byte, error) {
	return hex.DecodeString(Remove0xPrefix(query))
}

// Remove0xPrefix removes the '0x' prefix from a hex string and returns the result in lower case.
func Remove0xPrefix(hexString string) string {
	if has0xPrefix(hexString) {
		hexString = hexString[2:]
	}
	return strings.ToLower(hexString)
}

// has0xPrefix validates str begins with '0x' or '0X'.
// From: https://github.com/ethereum/go-ethereum/blob/5c6f4b9f0d4270fcc56df681bf003e6a74f11a6b/common/bytes.go#L51
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func FormatUint256(hexStr string) (string, error) {
	// Remove "0x" prefix if present
	if has0xPrefix(hexStr) {
		hexStr = hexStr[2:]
	}

	// Ensure the length is at most 64
	if len(hexStr) > 64 {
		return "", errors.New("hex string is too long")
	}

	// Pad with leading zeros if less than 64 characters
	for len(hexStr) < 64 {
		hexStr = "0" + hexStr
	}

	return hexStr, nil
}
