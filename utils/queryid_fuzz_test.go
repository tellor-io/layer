package utils

import (
	"math/big"
	"testing"
)

func FuzzQueryBytesFromString(f *testing.F) {
	f.Add("0xdeadbeef")
	f.Add("0Xdeadbeef")
	f.Add("deadbeef")
	f.Add("")
	f.Add("0x")
	f.Add("0x0")
	f.Add("zzzz")
	f.Add("0xZZZZ")
	f.Add("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080")

	f.Fuzz(func(t *testing.T, query string) {
		_, _ = QueryBytesFromString(query)
	})
}

func FuzzFormatUint256(f *testing.F) {
	f.Add("0x0000000000000000000000000000000000000000000000000000000000000001")
	f.Add("0x01")
	f.Add("ff")
	f.Add("")
	f.Add("0x")
	f.Add("0")
	// Exactly 64 chars
	f.Add("0000000000000000000000000000000000000000000000000000000000000001")
	// 65 chars - should error
	f.Add("00000000000000000000000000000000000000000000000000000000000000001")
	f.Add("zzzz")

	f.Fuzz(func(t *testing.T, hexStr string) {
		_, _ = FormatUint256(hexStr)
	})
}

func FuzzRemove0xPrefix(f *testing.F) {
	f.Add("0xdeadbeef")
	f.Add("0Xdeadbeef")
	f.Add("deadbeef")
	f.Add("")
	f.Add("0x")
	f.Add("0")
	f.Add("0x0x0x")

	f.Fuzz(func(t *testing.T, input string) {
		_ = Remove0xPrefix(input)
	})
}

func FuzzQueryIDFromData(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte("hello"))

	f.Fuzz(func(t *testing.T, data []byte) {
		result := QueryIDFromData(data)
		if len(result) != 32 {
			t.Errorf("expected 32 byte hash, got %d", len(result))
		}
	})
}

func FuzzFormatBigInt(f *testing.F) {
	f.Add(int64(0), 18)
	f.Add(int64(1000000000000000000), 18)
	f.Add(int64(-1), 6)
	f.Add(int64(999), 0)
	f.Add(int64(1), 100)

	f.Fuzz(func(t *testing.T, val int64, decimals int) {
		// Limit decimals to avoid extremely long computation
		if decimals < 0 || decimals > 200 {
			return
		}
		v := big.NewInt(val)
		FormatBigInt(v, decimals)
	})
}
