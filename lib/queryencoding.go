package lib

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// https://github.com/ethereum/go-ethereum/blob/master/accounts/abi/argument.go
func EncodeArguments(dataTypes []string, dataFields []string) ([]byte, error) {
	var arguments abi.Arguments

	interfaceFields := make([]interface{}, len(dataFields))
	for i, dataType := range dataTypes {
		argType, err := abi.NewType(dataType, dataType, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create new ABI type: %v", err)
		}

		interfaceFields[i], err = ConvertStringToType(dataType, dataFields[i])
		if err != nil {
			return nil, err
		}

		arguments = append(arguments, abi.Argument{
			Name:    "",
			Type:    argType,
			Indexed: false,
		})
	}

	return arguments.Pack(interfaceFields...)
}

func ConvertStringToType(dataType, dataField string) (interface{}, error) {
	switch dataType {
	case "string":
		return dataField, nil
	case "bool":
		return strconv.ParseBool(dataField)
	case "address":
		// TODO: Validate address, maybe?
		return dataField, nil
	case "bytes":
		return []byte(dataField), nil
	case "int8", "int16", "int32", "int64", "int128", "int256", "uint8", "uint16", "uint32", "uint64", "uint128", "uint256":
		// https://docs.soliditylang.org/en/latest/types.html#integers
		value := new(big.Int)
		value, success := value.SetString(dataField, 10)
		if !success {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("could not set string to big.Int for value %s", dataField))
		}
		return value, nil
	default:
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported data type: %s", dataType))
	}
}

func VerifyDataTypeFields(queryType string, datafields map[string]string) error {
	// check if data fields is empty
	if len(datafields) == 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("data field mapping is empty"))
	}
	// encode query data params
	encodedDatafields, err := EncodeData(datafields)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to encode data field mapping, data_type_to_field_name: %v", err))
	}
	querydata := map[string]string{
		"string": queryType,
		"bytes":  string(encodedDatafields),
	}
	// encode query data with query type
	_, err = EncodeData(querydata)
	if err != nil {
		return fmt.Errorf("failed to encode query data: %v", err)
	}
	return nil
}

func EncodeData(data map[string]string) ([]byte, error) {
	var arguments abi.Arguments
	i := 0
	interfaceFields := make([]interface{}, len(data))
	for datatype, datafield := range data {
		argType, err := abi.NewType(datatype, datatype, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create new ABI type: %v", err)
		}

		interfaceFields[i], err = ConvertStringToType(datatype, datafield)
		if err != nil {
			return nil, err
		}

		arguments = append(arguments, abi.Argument{
			Name:    "",
			Type:    argType,
			Indexed: false,
		})
		i++
	}

	return arguments.Pack(interfaceFields...)
}

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
