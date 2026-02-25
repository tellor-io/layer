package types

import (
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func FuzzNewMsgTip(f *testing.F) {
	f.Add("tellor1tipper", "deadbeef")
	f.Add("tellor1tipper", "0xdeadbeef")
	f.Add("", "")
	f.Add("addr", "0x")
	f.Add("addr", "zzzz")

	f.Fuzz(func(t *testing.T, tipper, queryData string) {
		coin := sdk.Coin{Denom: "loya", Amount: math.NewInt(1000)}
		_, _ = NewMsgTip(tipper, queryData, coin)
	})
}
