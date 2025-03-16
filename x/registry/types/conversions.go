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

var (
	// Regular expression for parsing array types ...[size]
	reArray = regexp.MustCompile(`(.+)\[(\d*)\]$`)
	// Regular expression for parsing integer types, e.g. uint256,int8
	reInt = regexp.MustCompile(`(u?)int(\d*)`)
)

// ConvertTypeToReflectType converts ABI type strings to reflect.Type
func ConvertTypeToReflectType(abiType string) (reflect.Type, error) {
	// Handling arrays
	if matches := reArray.FindStringSubmatch(abiType); matches != nil {
		elemType, err := ConvertTypeToReflectType(matches[1])
		if err != nil {
			return nil, err
		}
		if matches[2] == "" {
			return reflect.SliceOf(elemType), nil
		} else {
			size, err := strconv.Atoi(matches[2])
			if err != nil {
				return nil, fmt.Errorf("invalid array size: %s", abiType)
			}

			return reflect.ArrayOf(size, elemType), nil
		}
	}
	// Handling integer types
	if matches := reInt.FindStringSubmatch(abiType); matches != nil {
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
	// handle array types
	if matches := reArray.FindStringSubmatch(dataType); matches != nil {
		arrayType := matches[1]
		arraySize := matches[2]
		dataField = strings.Trim(dataField, "[]")
		fieldlist := strings.Split(dataField, ",")
		var arraySizeInt int
		if arraySize != "" {
			var err error
			arraySizeInt, err = strconv.Atoi(arraySize)
			if err != nil {
				return nil, fmt.Errorf("failed to convert string to integer for array size: %w", err)
			}
			if len(fieldlist) != arraySizeInt {
				return nil, fmt.Errorf("array size mismatch: expected %d, got %d", arraySizeInt, len(fieldlist))
			}
		} else {
			arraySizeInt = len(fieldlist)
		}

		reflectype, err := ConvertTypeToReflectType(arrayType)
		if err != nil {
			return nil, fmt.Errorf("failed to convert ABI type to reflect type: %w", err)
		}
		slice := reflect.MakeSlice(reflect.SliceOf(reflectype), 0, arraySizeInt)

		for _, field := range fieldlist {
			value, err := ConvertStringToType(arrayType, field)
			if err != nil {
				return nil, fmt.Errorf("failed to convert string to type: %w", err)
			}
			slice = reflect.Append(slice, reflect.ValueOf(value))
		}

		return slice.Interface(), nil
	}

	// Handling integer types
	if matches := reInt.FindStringSubmatch(dataType); matches != nil {
		value, err := intValue(matches, dataField)
		if err != nil {
			return nil, fmt.Errorf("failed to convert string to integer, err: %w", err)
		}
		return value, nil
	}

	switch dataType {
	case "string":
		return dataField, nil
	case "bool":
		return strconv.ParseBool(dataField)
	case "address":
		if !common.IsHexAddress(dataField) {
			return nil, fmt.Errorf("invalid address: %s", dataField)
		}
		return common.HexToAddress(dataField), nil
	case "bytes":
		if strings.HasPrefix(dataField, "0x") {
			return hex.DecodeString(Remove0xPrefix(dataField))
		}
		return []byte(dataField), nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

func sizeOfType(match string) (int, error) {
	if match == "" {
		return 256, nil
	}
	num, err := strconv.Atoi(match)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string to integer for data type: %w", err)
	}

	return num, nil
}

func intValue(matches []string, fieldValue string) (interface{}, error) {
	size, err := sizeOfType(matches[2])
	if err != nil {
		return nil, fmt.Errorf("failed to convert string to integer, err: %w", err)
	}
	fieldValue = strings.TrimSpace(fieldValue)
	Uint := matches[1] == "u"
	switch size {
	case 8, 16, 32, 64:
		if Uint {
			// Use ParseUint for unsigned types
			value, err := strconv.ParseUint(fieldValue, 10, size)
			if err != nil {
				return nil, fmt.Errorf("failed to convert data field string to unsigned integer, err: %w", err)
			}
			switch size {
			case 8:
				return uint8(value), nil
			case 16:
				return uint16(value), nil
			case 32:
				return uint32(value), nil
			case 64:
				return value, nil
			}
		} else {
			// Use ParseInt for signed types
			value, err := strconv.ParseInt(fieldValue, 10, size)
			if err != nil {
				return nil, fmt.Errorf("failed to convert data field string to integer, err: %w", err)
			}
			switch size {
			case 8:
				return int8(value), nil
			case 16:
				return int16(value), nil
			case 32:
				return int32(value), nil
			case 64:
				return value, nil
			}
		}
	default:
		// Handle as big.Int for sizes not matching 8, 16, 32, or 64
		value := new(big.Int)
		value, success := value.SetString(fieldValue, 10)
		if !success {
			return nil, fmt.Errorf("could not set string to big.Int for value %s", fieldValue)
		}
		return value, nil
	}
	// This line is unreachable but included for completeness
	return nil, fmt.Errorf("unsupported data type size: %s", matches[2])
}
