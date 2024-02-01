package lib

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// func VerifyDataTypeFields(querytype string, datatypes, datafields []string) error {
// 	// check if data fields is empty
// 	if len(datafields) == 0 {
// 		return status.Error(codes.InvalidArgument, fmt.Sprintf("data field mapping is empty"))
// 	}
// 	// encode query data params
// 	encodedDatafields, err := EncodeArguments(datatypes, datafields)
// 	if err != nil {
// 		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to encode data field mapping, data_type_to_field_name: %v", err))
// 	}
// 	// encode query data with query type
// 	_, err = EncodeWithQuerytype(querytype, encodedDatafields)

// 	if err != nil {
// 		return fmt.Errorf("failed to encode query data: %v", err)
// 	}
// 	return nil
// }

// has0xPrefix validates str begins with '0x' or '0X'.
// From: https://github.com/ethereum/go-ethereum/blob/5c6f4b9f0d4270fcc56df681bf003e6a74f11a6b/common/bytes.go#L51
func Has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// check queryId is valid ie 32 bytes
func IsQueryId64chars(queryId string) bool {
	hasPrefix := Has0xPrefix(queryId)
	if hasPrefix {
		queryId = queryId[2:]
	}
	if len(queryId) != 64 {
		return false
	}
	return true
}

func RemoveHexPrefix(hexString string) string {
	if Has0xPrefix(hexString) {
		hexString = hexString[2:]
	}
	return hexString
}

func EncodeWithQuerytype(querytype string, databytes []byte) ([]byte, error) {
	strArg, err := abi.NewType("string", "string", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ABI type when encoding query type: %v", err)
	}
	bytesArg, err := abi.NewType("bytes", "bytes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ABI type when encoding with query type: %v", err)
	}
	args := abi.Arguments{
		abi.Argument{Type: strArg},
		abi.Argument{Type: bytesArg},
	}
	return args.Pack(querytype, databytes)
}
