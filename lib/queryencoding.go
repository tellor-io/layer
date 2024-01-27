package lib

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

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
		if !strings.HasPrefix(dataField, "0x") {
			return nil, fmt.Errorf("invalid address format")
		}
		addrBytes, err := hex.DecodeString(dataField[2:])
		if err != nil {
			return nil, err
		}
		if len(addrBytes) != 20 {
			return nil, fmt.Errorf("invalid address length")
		}
		var addr [20]byte
		copy(addr[:], addrBytes)
		return addr, nil
	case "bytes":
		// TODO: decode bytes properly
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

func VerifyDataTypeFields(querytype string, datatypes, datafields []string) error {
	// check if data fields is empty
	if len(datafields) == 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("data field mapping is empty"))
	}
	// encode query data params
	encodedDatafields, err := EncodeArguments(datatypes, datafields)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to encode data field mapping, data_type_to_field_name: %v", err))
	}
	// encode query data with query type
	_, err = EncodeArguments([]string{"string", "bytes"}, []string{querytype, string(encodedDatafields)})

	if err != nil {
		return fmt.Errorf("failed to encode query data: %v", err)
	}
	return nil
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

func RemoveHexPrefix(hexString string) string {
	if Has0xPrefix(hexString) {
		hexString = hexString[2:]
	}
	return hexString
}

// Decodes query data bytes to query type and data fields
func DecodeQueryType(data []byte) (string, []byte, error) {
	// Create an ABI arguments object based on the types
	strArg, err := abi.NewType("string", "string", nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create new ABI type when decoding query type: %v", err)
	}
	bytesArg, err := abi.NewType("bytes", "bytes", nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create new ABI type when decoding query type: %v", err)
	}
	args := abi.Arguments{
		abi.Argument{Type: strArg},
		abi.Argument{Type: bytesArg},
	}
	result, err := args.UnpackValues(data)
	if err != nil {
		return "", nil, fmt.Errorf("failed to unpack query type: %v", err)
	}
	return result[0].(string), result[1].([]byte), nil
}

// Decodes query data bytes to query type and data fields
func DecodeParamtypes(data []byte, types []string) (string, error) {
	var args abi.Arguments
	for _, t := range types {
		argType, err := abi.NewType(t, t, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create new ABI type: %v", err)
		}
		args = append(args, abi.Argument{Type: argType})
	}

	result, err := args.UnpackValues(data)
	if err != nil {
		return "", fmt.Errorf("failed to unpack query type: %v", err)
	}

	return convertToJSON(result, types)
}

// convertToJSON converts a slice of interfaces into a JSON string.
func convertToJSON(slice []interface{}, types []string) (string, error) {
	var items []map[string]interface{}
	for i, item := range slice {
		itemType := types[i]

		itemMap := map[string]interface{}{
			"type":  itemType,
			"value": item,
		}
		items = append(items, itemMap)
	}

	jsonResult, err := json.Marshal(items)
	if err != nil {
		return "", err
	}

	return string(jsonResult), nil
}

func GenerateQuerydata(querytype string, parameters []string, queryParameterTypes []string) (string, error) {
	if len(parameters) == 0 {
		return "", status.Error(codes.InvalidArgument, fmt.Sprintf("data field mapping is empty"))
	}
	// encode query data params
	encodedDatafields, err := EncodeArguments(queryParameterTypes, parameters)
	if err != nil {
		return "", status.Error(codes.InvalidArgument, fmt.Sprintf("failed to encode data field mapping, data_type_to_field_name: %v", err))
	}

	// encode query data with query type
	encodedQuerydata, err := EncodeArguments([]string{"string", "bytes"}, []string{querytype, string(encodedDatafields)})
	if err != nil {
		return "", fmt.Errorf("failed to encode query data: %v", err)
	}
	return hex.EncodeToString(encodedQuerydata), nil
}

func IsValueDecodable(value, datatype string) error {
	argType, err := abi.NewType(datatype, datatype, nil)
	if err != nil {
		return fmt.Errorf("failed to create new ABI type when decoding value: %v", err)
	}
	arg := abi.Argument{
		Type: argType,
	}
	args := abi.Arguments{arg}
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode value string: %v", err))
	}
	var result []interface{}
	result, err = args.Unpack(valueBytes)
	if err != nil {
		return fmt.Errorf("failed to unpack value: %v", err)
	}
	fmt.Println("Decoded value: ", result[0])
	return nil
}
