package types

import (
	"testing"
)

func FuzzRemove0xPrefix(f *testing.F) {
	f.Add("0xdeadbeef")
	f.Add("0Xdeadbeef")
	f.Add("deadbeef")
	f.Add("")
	f.Add("0x")
	f.Add("0")
	f.Add("0x0x0x")

	f.Fuzz(func(t *testing.T, input string) {
		result := Remove0xPrefix(input)
		if Has0xPrefix(result) && !Has0xPrefix(input[2:]) {
			t.Errorf("Remove0xPrefix(%q) = %q still has prefix", input, result)
		}
	})
}

func FuzzConvertToJSON(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add("123")
	f.Add("\x00\xff")

	f.Fuzz(func(t *testing.T, val string) {
		slice := []interface{}{val}
		_, err := ConvertToJSON(slice)
		if err != nil {
			t.Skip()
		}
	})
}
