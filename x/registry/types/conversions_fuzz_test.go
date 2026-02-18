package types

import (
	"testing"
)

func FuzzConvertTypeToReflectType(f *testing.F) {
	f.Add("uint256")
	f.Add("int8")
	f.Add("uint64")
	f.Add("bool")
	f.Add("string")
	f.Add("address")
	f.Add("bytes")
	f.Add("uint256[]")
	f.Add("address[]")
	f.Add("bytes32")
	f.Add("int128[]")
	// Edge cases
	f.Add("")
	f.Add("uint0")
	f.Add("uint999")
	f.Add("int7")
	f.Add("[]")
	f.Add("[0]")
	f.Add("uint256[0]")
	f.Add("uint256[-1]")
	f.Add("uint256[99999999999999999999]")
	f.Add("a]")
	f.Add("a[")
	f.Add("a[b]")
	f.Add("a[][]")
	f.Add("a[][][][][][][][][]")

	f.Fuzz(func(t *testing.T, abiType string) {
		_, _ = ConvertTypeToReflectType(abiType)
	})
}

func FuzzConvertStringToType(f *testing.F) {
	// Valid cases
	f.Add("uint256", "12345")
	f.Add("uint8", "255")
	f.Add("int8", "-128")
	f.Add("int64", "9999999")
	f.Add("bool", "true")
	f.Add("bool", "false")
	f.Add("string", "hello world")
	f.Add("address", "0x88df592f8eb5d7bd38bfef7deb0fbc02cf3778a0")
	f.Add("bytes", "0xdeadbeef")
	f.Add("bytes", "raw bytes here")
	f.Add("uint256[]", "1,2,3")
	f.Add("int8[]", "-1,0,1")
	// Edge cases
	f.Add("uint256", "")
	f.Add("uint256", "-1")
	f.Add("uint8", "256")
	f.Add("int8", "999")
	f.Add("bool", "maybe")
	f.Add("address", "not-an-address")
	f.Add("address", "0x")
	f.Add("bytes", "0xZZZZ")
	f.Add("", "anything")
	f.Add("uint256[]", "")
	f.Add("uint256[2]", "1")
	f.Add("uint256[1]", "1,2,3")
	f.Add("notreal", "data")

	f.Fuzz(func(t *testing.T, dataType, dataField string) {
		_, _ = ConvertStringToType(dataType, dataField)
	})
}
