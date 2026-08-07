package enrich_test

import (
	"testing"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/enrich"
)

// Known ABI-encoded vectors (abi.encode(string, abi.encode(bool, uint256))).
const (
	queryDataDeposit206 = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000b54524242726964676556320000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000ce"
	queryDataWithdraw1  = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000b5452424272696467655632000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"
	queryDataV1Deposit1 = "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000095452424272696467650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001"
)

func TestDecodeBridgeDepositQueryData(t *testing.T) {
	dep, ok := enrich.DecodeBridgeDepositQueryData(queryDataDeposit206)
	if !ok {
		t.Fatal("expected deposit 206 decode ok")
	}
	if dep.DepositID != 206 || dep.QueryType != "TRBBridgeV2" || !dep.IsBridgeDeposit() {
		t.Fatalf("unexpected deposit: %+v", dep)
	}

	// 0x prefix + mixed case should still work.
	dep, ok = enrich.DecodeBridgeDepositQueryData("0x" + queryDataDeposit206)
	if !ok || dep.DepositID != 206 {
		t.Fatalf("0x prefix decode failed: ok=%v dep=%+v", ok, dep)
	}

	if _, ok := enrich.DecodeBridgeDepositQueryData(queryDataWithdraw1); ok {
		t.Fatal("withdrawal must not be treated as deposit")
	}

	dep, ok = enrich.DecodeBridgeDepositQueryData(queryDataV1Deposit1)
	if !ok || dep.DepositID != 1 || dep.QueryType != "TRBBridge" {
		t.Fatalf("v1 deposit: ok=%v dep=%+v", ok, dep)
	}

	if _, ok := enrich.DecodeBridgeDepositQueryData(""); ok {
		t.Fatal("empty should fail")
	}
	if _, ok := enrich.DecodeBridgeDepositQueryData("deadbeef"); ok {
		t.Fatal("garbage should fail")
	}
}

func TestTipCommand(t *testing.T) {
	got := enrich.TipCommand(queryDataDeposit206, "layertest-4")
	want := "./layerd tx oracle tip " + queryDataDeposit206 + " 1500loya --chain-id layertest-4"
	if got != want {
		t.Fatalf("tip cmd =\n%q\nwant\n%q", got, want)
	}
	if enrich.TipCommand("0x"+queryDataDeposit206, "layertest-4") != want {
		t.Fatal("tip cmd should strip 0x")
	}
	if enrich.TipCommand(queryDataDeposit206, "") != "" {
		t.Fatal("empty chain id should yield empty tip")
	}
	if enrich.TipCommand("", "layertest-4") != "" {
		t.Fatal("empty query data should yield empty tip")
	}
}

func TestQueryTypeFromQueryData(t *testing.T) {
	if got := enrich.QueryTypeFromQueryData(queryDataDeposit206); got != "TRBBridgeV2" {
		t.Fatalf("deposit206 type = %q", got)
	}
	if got := enrich.QueryTypeFromQueryData("0x" + queryDataV1Deposit1); got != "TRBBridge" {
		t.Fatalf("v1 type = %q", got)
	}
	if enrich.QueryTypeFromQueryData("") != "" {
		t.Fatal("empty should be empty")
	}
	if enrich.QueryTypeFromQueryData("deadbeef") != "" {
		t.Fatal("garbage should be empty")
	}
}

func TestIsBridgeQueryType(t *testing.T) {
	if !enrich.IsBridgeQueryType("TRBBridgeV2") || !enrich.IsBridgeQueryType("trbbridge") {
		t.Fatal("expected bridge types")
	}
	if enrich.IsBridgeQueryType("SpotPrice") {
		t.Fatal("SpotPrice is not bridge")
	}
}
