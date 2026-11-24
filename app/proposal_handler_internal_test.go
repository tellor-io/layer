package app

import "testing"

func TestTrimTxsToFit(t *testing.T) {
	tx := func(n int) []byte { return make([]byte, n) }
	cases := []struct {
		name   string
		txs    [][]byte
		budget int64
		want   int
	}{
		{"empty", nil, 1000, 0},
		{"fits exactly with overhead", [][]byte{tx(96), tx(96)}, 200, 2}, // 2*(96+4)=200
		{"cuts one over", [][]byte{tx(96), tx(97)}, 200, 1},              // 96+4+97+4=201>200
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
	}
}
