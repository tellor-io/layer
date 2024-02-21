package utils

import (
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func QueryIDFromData(queryData []byte) []byte {
	return crypto.Keccak256(queryData)
}

func QueryIDFromDataString(queryData string) ([]byte, error) {
	bz, err := hex.DecodeString(Remove0xPrefix(queryData))
	if err != nil {
		return nil, err
	}

	return QueryIDFromData(bz), nil
}

func QueryIDFromString(queryID string) ([]byte, error) {
	return hex.DecodeString(Remove0xPrefix(queryID))
}

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
