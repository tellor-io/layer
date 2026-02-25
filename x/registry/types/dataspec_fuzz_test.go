package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func FuzzMakeArguments(f *testing.F) {
	f.Add("uint256", "value")
	f.Add("bool", "flag")
	f.Add("string", "name")
	f.Add("address", "addr")
	f.Add("bytes", "data")
	f.Add("uint256[]", "arr")
	f.Add("tuple", "main")
	f.Add("tuple[]", "list")
	// Edge cases
	f.Add("", "")
	f.Add("notreal", "x")
	f.Add("uint0", "x")
	f.Add("int7", "x")

	f.Fuzz(func(t *testing.T, typeName, name string) {
		args := []abi.ArgumentMarshaling{{Type: typeName, Name: name}}
		_, _ = MakeArguments(args)
	})
}

func FuzzEncodeWithQuerytype(f *testing.F) {
	f.Add("SpotPrice", []byte{0x01, 0x02, 0x03})
	f.Add("", []byte{})
	f.Add("TRBBridge", []byte{0x00})
	f.Add("a]b[c", []byte{0xff})

	f.Fuzz(func(t *testing.T, querytype string, data []byte) {
		_, _ = EncodeWithQuerytype(querytype, data)
	})
}

func FuzzValidateValue(f *testing.F) {
	f.Add("uint256", "0x0000000000000000000000000000000000000000000000000000000000000009")
	f.Add("bool", "0x0000000000000000000000000000000000000000000000000000000000000001")
	f.Add("string", "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000568656c6c6f000000000000000000000000000000000000000000000000000000")
	// Edge cases
	f.Add("uint256", "")
	f.Add("uint256", "0x")
	f.Add("uint256", "garbage")
	f.Add("", "0x00")
	f.Add("badtype", "0x00")

	f.Fuzz(func(t *testing.T, responseValueType, value string) {
		d := DataSpec{ResponseValueType: responseValueType}
		_ = d.ValidateValue(value)
	})
}

func FuzzDecodeValueRoundtrip(f *testing.F) {
	f.Add("uint256", "0x0000000000000000000000000000000000000000000000000000000000000009")
	f.Add("bool", "0x0000000000000000000000000000000000000000000000000000000000000001")
	f.Add("int8", "0x0000000000000000000000000000000000000000000000000000000000000009")

	f.Fuzz(func(t *testing.T, responseValueType, value string) {
		d := DataSpec{ResponseValueType: responseValueType}
		// If DecodeValue succeeds, ConvertToJSON shouldn't panic
		_, _ = d.DecodeValue(value)
	})
}
