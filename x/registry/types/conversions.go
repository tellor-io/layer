package types

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ConvertABIToReflectType converts ABI type strings to reflect.Type
func ConvertABIToReflectType(abiType string) (reflect.Type, error) {
	// Regular expression for parsing array types
	reArray := regexp.MustCompile(`(.+)\[(\d*)\]$`)
	// Regular expression for parsing integer types
	reInt := regexp.MustCompile(`(u?)int(\d*)`)
	fmt.Println("abiType", abiType)
	// Handling integer types
	if matches := reInt.FindStringSubmatch(abiType); matches != nil {
		fmt.Println("matches ", matches)
		size := 256 // default size
		if matches[2] != "" {
			var err error
			size, err = strconv.Atoi(matches[2])
			if err != nil || size <= 0 || size > 256 || size%8 != 0 {
				return nil, fmt.Errorf("invalid integer size: %s", abiType)
			}
		}
		switch size {
		case 8:
			if matches[1] == "u" {
				return reflect.TypeOf(uint8(0)), nil
			}
			return reflect.TypeOf(int8(0)), nil
		case 16:
			if matches[1] == "u" {
				return reflect.TypeOf(uint16(0)), nil
			}
			return reflect.TypeOf(int16(0)), nil
		case 32:
			if matches[1] == "u" {

				return reflect.TypeOf(uint32(0)), nil
			}
			return reflect.TypeOf(int32(0)), nil
		case 64:
			if matches[1] == "u" {

				return reflect.TypeOf(uint64(0)), nil
			}
			return reflect.TypeOf(int64(0)), nil
		default:
			return reflect.TypeOf(new(big.Int)), nil
		}
	}

	// Handling arrays
	if matches := reArray.FindStringSubmatch(abiType); matches != nil {
		elemType, err := ConvertABIToReflectType(matches[1])
		if err != nil {
			return nil, err
		}
		if matches[2] == "" { // dynamic array
			return reflect.SliceOf(elemType), nil
		} else { // fixed-size array
			size, err := strconv.Atoi(matches[2])
			if err != nil {
				return nil, fmt.Errorf("invalid array size: %s", abiType)
			}
			return reflect.ArrayOf(size, elemType), nil
		}
	}
	switch abiType {
	case "string":
		return reflect.TypeOf(""), nil
	case "address":
		return reflect.TypeOf(common.Address{}), nil
	case "bool":
		return reflect.TypeOf(false), nil
	case "bytes":
		return reflect.TypeOf([]byte{}), nil
	default:
		return nil, fmt.Errorf("unsupported ABI type: %s", abiType)
	}
}

func ConvertStringToType(dataType, dataField string) (interface{}, error) {
	// TODO: make more robust and handle multidimensional arrays
	if strings.Contains(dataType, "int") {
		if strings.HasSuffix(dataType, "[]") {
			dataType = "int[]"
		} else {
			dataType = "int"
		}
	}

	switch dataType {
	case "string":
		return dataField, nil
	case "string[]":
		dataField = strings.Trim(dataField, "[]")
		return []string{dataField}, nil
	case "bool":
		return strconv.ParseBool(dataField)
	case "bool[]":
		dataField = strings.Trim(dataField, "[]")
		// Bool
		boolStrings := strings.Split(dataField, ",")
		boolSlice := make([]bool, 0, len(boolStrings))
		for _, boolStr := range boolStrings {
			boolVal, err := strconv.ParseBool(boolStr)
			if err != nil {
				return nil, fmt.Errorf("could not parse bool string %s", boolStr)
			}
			boolSlice = append(boolSlice, boolVal)
		}
		return boolSlice, nil
	case "address":
		return common.HexToAddress(dataField), nil
	case "address[]":
		dataField = strings.Trim(dataField, "[]")
		// Address
		addressStrings := strings.Split(dataField, ",")
		addressSlice := make([]common.Address, 0, len(addressStrings))
		for _, addressStr := range addressStrings {
			addressSlice = append(addressSlice, common.HexToAddress(addressStr))
		}
		return addressSlice, nil
	case "bytes", "bytes[]":
		if strings.HasPrefix(dataField, "0x") {
			dataField = dataField[2:]
			return hex.DecodeString(dataField)
		}
		return []byte(dataField), nil
	case "bytes32", "bytes32[]":
		var b [32]byte
		if dataType == "bytes32" {
			byt := []byte(dataField)
			copy(b[:], byt)
			return b, nil
		} else {
			copy(b[:], []byte(dataField))
			return [][32]byte{b}, nil
		}
	case "int":
		// https://docs.soliditylang.org/en/latest/types.html#integers
		value := new(big.Int)
		value, success := value.SetString(dataField, 10)
		if !success {
			return nil, fmt.Errorf("could not set string to big.Int for value %s", dataField)
		}
		return value, nil
	case "int[]":
		// Remove the brackets
		dataField = strings.Trim(dataField, "[]")

		// Split the string by commas
		numberStrings := strings.Split(dataField, ",")

		// Convert each string number to a big.Int
		bigIntSlice := make([]*big.Int, 0, len(numberStrings))
		for _, numberStr := range numberStrings {
			numberStr = strings.TrimSpace(numberStr) // Remove any whitespace
			num := new(big.Int)
			_, success := num.SetString(numberStr, 10) // Base 10 for decimal
			if !success {
				return nil, fmt.Errorf("Error converting '%s' to big.Int\n", numberStr)
			}
			bigIntSlice = append(bigIntSlice, num)
		}
		return bigIntSlice, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}
