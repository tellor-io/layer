package app

import (
	"testing"

	cmttypes "github.com/cometbft/cometbft/types"
)

func TestTxBlockSize(t *testing.T) {
	// The extra bytes per tx grow with its size: 2, 3, 4, then 5 from 2 MiB up.
	cases := []struct {
		txLen int
		want  int64
	}{
		{0, 2},
		{127, 129},
		{128, 131},
		{16383, 16386},
		{16384, 16388},
		{2097151, 2097155},
		{2097152, 2097157},
	}
	for _, c := range cases {
		if got := txBlockSize(make([]byte, c.txLen)); got != c.want {
			t.Errorf("tx of %d bytes: got %d, want %d", c.txLen, got, c.want)
		}
	}
}

func TestTrimTxsToFit(t *testing.T) {
	tx := func(n int) []byte { return make([]byte, n) }
	cases := []struct {
		name   string
		txs    [][]byte
		budget int64
		want   int
	}{
		{"empty", nil, 1000, 0},
		{"fits exactly with overhead", [][]byte{tx(96), tx(96)}, 196, 2}, // 2*(96+2)=196
		{"cuts one over", [][]byte{tx(96), tx(97)}, 196, 1},              // 96+2+97+2=197>196
		{"none fit", [][]byte{tx(500)}, 100, 0},
		{"non-positive budget drops everything", [][]byte{tx(1)}, 0, 0},
		// A tx that doesn't fit stops the scan: smaller later txs are NOT kept,
		// so mempool priority order is preserved (baseapp preparer semantics).
		{"keeps prefix order", [][]byte{tx(10), tx(10), tx(1000), tx(10)}, 30, 2},
	}
	for _, c := range cases {
		got := trimTxsToFit(c.txs, c.budget)
		if len(got) != c.want {
			t.Errorf("%s: got %d txs, want %d", c.name, len(got), c.want)
		}
		// Whatever is kept must pass CometBFT's own size check.
		if err := cmttypes.ToTxs(got).Validate(c.budget); err != nil {
			t.Errorf("%s: kept txs fail CometBFT's size check: %v", c.name, err)
		}
	}
}
