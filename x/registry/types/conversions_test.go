package types

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestConvertTypeToReflectType(t *testing.T) {
	tests := []struct {
		abiType    string
		expected   reflect.Type
		shouldFail bool
	}{
		{"uint8", reflect.TypeOf(uint8(0)), false},
		{"int8", reflect.TypeOf(int8(0)), false},
		{"uint256", reflect.TypeOf(new(big.Int)), false},
		{"int64", reflect.TypeOf(int64(0)), false},
		{"uint", reflect.TypeOf(new(big.Int)), false},
		{"string", reflect.TypeOf(""), false},
		{"bool", reflect.TypeOf(false), false},
		{"bytes", reflect.TypeOf([]byte{}), false},
		{"address", reflect.TypeOf(common.Address{}), false},
		{"uint1024", nil, true},
		{"int[3]", reflect.ArrayOf(3, reflect.PtrTo(reflect.TypeOf(big.Int{}))), false},
		{"uint8[]", reflect.SliceOf(reflect.TypeOf(uint8(0))), false},
	}

	for _, tt := range tests {
		t.Run(tt.abiType, func(t *testing.T) {
			got, err := ConvertTypeToReflectType(tt.abiType)
			if !tt.shouldFail {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)

			}
			if tt.shouldFail {
				require.Error(t, err)
			}
		})
	}
}

func TestConvertStringToType(t *testing.T) {
	bigNum, _ := new(big.Int).SetString("9223372036854775808", 10)
	tests := []struct {
		name       string
		dataType   string
		dataField  string
		expected   interface{}
		shouldFail bool
	}{
		{"Int8", "int8", "127", int8(127), false},
		{"Uint8", "uint8", "255", uint8(255), false},
		{"Int16", "int16", "32767", int16(32767), false},
		{"Uint16", "uint16", "65535", uint16(65535), false},
		{"Int32", "int32", "2147483647", int32(2147483647), false},
		{"Uint32", "uint32", "4294967295", uint32(4294967295), false},
		{"Int64", "int64", "-9223372036854775808", int64(-9223372036854775808), false},
		{"Uint64", "uint64", "18446744073709551615", uint64(18446744073709551615), false},
		{"BigInt", "int", "9223372036854775808", bigNum, false},
		{"Int", "int", "123", big.NewInt(123), false},
		{"Uint", "uint", "123", big.NewInt(123), false},
		{"BoolTrue", "bool", "true", true, false},
		{"BoolFalse", "bool", "false", false, false},
		{"String", "string", "test", "test", false},
		{"BytesHex", "bytes", "0xdeadbeef", hexutil.MustDecode("0xdeadbeef"), false},
		{"BytesPlain", "bytes", "hello", []byte("hello"), false},
		{"Address", "address", "0x1234567890ABCDEF1234567890ABCDEF12345678", common.HexToAddress("0x1234567890ABCDEF1234567890ABCDEF12345678"), false},
		{"IntArray", "int[3]", "[1,2,3]", []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}, false},
		{"Int8Overflow", "int8", "128", nil, true},
		{"Uint8Overflow", "uint8", "256", nil, true},
		{"Int16Overflow", "int16", "32768", nil, true},
		{"Uint16Overflow", "uint16", "65536", nil, true},
		{"Int32Overflow", "int32", "2147483648", nil, true},
		{"Uint32Overflow", "uint32", "4294967296", nil, true},
		{"Int64Underflow", "int64", "-9223372036854775809", nil, true},
		{"Uint64Overflow", "uint64", "18446744073709551616", nil, true},
		{"BoolInvalid", "bool", "notabool", nil, true},
		{"BytesInvalidHex", "bytes", "0xGHIJKL", nil, true},
		{"AddressInvalid", "address", "0xGHIJKL1234567890ABCDEF1234567890ABCDEF", nil, true},
		{"IntArrayInvalid", "int[3]", "[1,2,3,4]", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertStringToType(tt.dataType, tt.dataField)
			if !tt.shouldFail {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
			if tt.shouldFail {
				require.Error(t, err)
			}
		})
	}
}

func TestSizeOfType(t *testing.T) {
	res, err := sizeOfType("8")
	require.NoError(t, err)
	require.Equal(t, res, 8)
}

func TestIntValue(t *testing.T) {
	typeArray := []string{"[int8]", "", ""}
	res, err := intValue(typeArray, "123")
	require.NoError(t, err)
	require.Equal(t, res, big.NewInt(123))
}
