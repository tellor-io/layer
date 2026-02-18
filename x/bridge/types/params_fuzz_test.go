package types

import (
	"testing"

	"cosmossdk.io/math"
)

func FuzzParamsValidate(f *testing.F) {
	f.Add("0.01", "0.05", uint64(600000), uint64(600000), uint64(0), "tellor-1")
	f.Add("0", "0", uint64(1000), uint64(1000), uint64(0), "")
	f.Add("1", "1", uint64(1814400000), uint64(1814400000), ^uint64(0), "a]")
	f.Add("-1", "2", uint64(0), uint64(0), uint64(0), "tellor-1")
	f.Add("0.5", "0.5", uint64(999), uint64(999), uint64(0), "test")

	f.Fuzz(func(t *testing.T, attestSlash, valsetSlash string, attestWindow, valsetWindow, penaltyCutoff uint64, chainId string) {
		attestDec, err := math.LegacyNewDecFromStr(attestSlash)
		if err != nil {
			return
		}
		valsetDec, err := math.LegacyNewDecFromStr(valsetSlash)
		if err != nil {
			return
		}
		p := NewParams(attestDec, attestWindow, valsetDec, valsetWindow, penaltyCutoff, chainId)
		_ = p.Validate()
	})
}
