package types

import (
	"encoding/json"
)

func Remove0xPrefix(hexString string) string {
	if Has0xPrefix(hexString) {
		hexString = hexString[2:]
	}
	return hexString
}

// has0xPrefix validates str begins with '0x' or '0X'.
// From: https://github.com/ethereum/go-ethereum/blob/5c6f4b9f0d4270fcc56df681bf003e6a74f11a6b/common/bytes.go#L51
func Has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// convertToJSON converts a slice of interfaces into a JSON string.
func ConvertToJSON(slice []interface{}) (string, error) {
	jsonResult, err := json.Marshal(slice)
	if err != nil {
		return "", err
	}

	return string(jsonResult), nil
}

// check queryId is valid ie 32 bytes
func IsQueryId64chars(queryId string) bool {
	queryId = Remove0xPrefix(queryId)
	return len(queryId) == 64
}
