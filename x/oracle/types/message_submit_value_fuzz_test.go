package types

import (
	"testing"
)

func FuzzNewMsgSubmitValue(f *testing.F) {
	// Valid hex query data
	f.Add("tellor1creator", "deadbeef", "0x0000000000000000000000000000000000000000000000000000000000000001", "salt")
	f.Add("tellor1creator", "0xdeadbeef", "value", "")
	// Edge cases
	f.Add("", "", "", "")
	f.Add("addr", "0x", "val", "salt")
	f.Add("addr", "zzzz", "val", "salt") // invalid hex - should panic

	f.Fuzz(func(t *testing.T, creator, queryData, value, salt string) {
		_, _ = NewMsgSubmitValue(creator, queryData, value, salt)
	})
}

func FuzzGetSignerAndValidateMsg(f *testing.F) {
	f.Add("tellor1fl48vsnmsdzcv85q5d2q4z5ajdha8yu3prylw4", []byte{0xde, 0xad}, "somevalue")
	f.Add("", []byte{}, "")
	f.Add("notanaddress", []byte{0x01}, "val")
	f.Add("tellor1fl48vsnmsdzcv85q5d2q4z5ajdha8yu3prylw4", []byte{}, "val")
	f.Add("tellor1fl48vsnmsdzcv85q5d2q4z5ajdha8yu3prylw4", []byte{0x01}, "")

	f.Fuzz(func(t *testing.T, creator string, queryData []byte, value string) {
		msg := &MsgSubmitValue{
			Creator:   creator,
			QueryData: queryData,
			Value:     value,
		}
		_, _ = msg.GetSignerAndValidateMsg()
	})
}
