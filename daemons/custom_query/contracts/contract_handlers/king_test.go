package contract_handlers

import (
	"context"
	"fmt"
	"testing"

	reader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
)

func TestKingContractCall(t *testing.T) {
	// Test with mainnet ETH RPC endpoint
	rpcURLs := []string{
		"https://eth.public-rpc.com",
		"https://ethereum.publicnode.com",
		"https://rpc.ankr.com/eth",
	}

	r, err := reader.NewReader(rpcURLs, 30)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer r.Close()

	ctx := context.Background()

	// Test 1: Try calling with the current signature and 1e18
	fmt.Println("Test 1: fairValueOf(uint256) with 1e18")
	result, err := r.ReadContract(ctx, KING_CONTRACT, "fairValueOf(uint256)", []string{"1000000000000000000"})
	if err != nil {
		t.Logf("Test 1 failed: %v", err)
	} else {
		fmt.Printf("Test 1 succeeded! Result length: %d bytes\n", len(result))
		fmt.Printf("Raw hex: 0x%x\n", result)
	}

	// Test 1b: Try with 0 input
	fmt.Println("\nTest 1b: fairValueOf(uint256) with 0")
	result, err = r.ReadContract(ctx, KING_CONTRACT, "fairValueOf(uint256)", []string{"0"})
	if err != nil {
		t.Logf("Test 1b failed: %v", err)
	} else {
		fmt.Printf("Test 1b succeeded! Result length: %d bytes\n", len(result))
		fmt.Printf("Raw hex: 0x%x\n", result)
	}

	// Test 1c: Try with 1 wei
	fmt.Println("\nTest 1c: fairValueOf(uint256) with 1")
	result, err = r.ReadContract(ctx, KING_CONTRACT, "fairValueOf(uint256)", []string{"1"})
	if err != nil {
		t.Logf("Test 1c failed: %v", err)
	} else {
		fmt.Printf("Test 1c succeeded! Result length: %d bytes\n", len(result))
		fmt.Printf("Raw hex: 0x%x\n", result)
	}

	// Test 2: Try with explicit returns in signature
	fmt.Println("\nTest 2: fairValueOf(uint256) returns (uint256,uint256)")
	result, err = r.ReadContract(ctx, KING_CONTRACT, "fairValueOf(uint256) returns (uint256,uint256)", []string{"1000000000000000000"})
	if err != nil {
		t.Logf("Test 2 failed: %v", err)
	} else {
		fmt.Printf("Test 2 succeeded! Result length: %d bytes\n", len(result))
		fmt.Printf("Raw hex: 0x%x\n", result)
	}

	// Test 3: Try without spaces in returns
	fmt.Println("\nTest 3: fairValueOf(uint256) returns (uint256, uint256)")
	result, err = r.ReadContract(ctx, KING_CONTRACT, "fairValueOf(uint256) returns (uint256, uint256)", []string{"1000000000000000000"})
	if err != nil {
		t.Logf("Test 3 failed: %v", err)
	} else {
		fmt.Printf("Test 3 succeeded! Result length: %d bytes\n", len(result))
		fmt.Printf("Raw hex: 0x%x\n", result)
	}

	// If any test succeeded, use the handler
	if len(result) > 0 {
		handler := &KingHandler{}
		value, err := handler.FetchValue(ctx, r, nil)
		if err != nil {
			t.Logf("Handler failed: %v", err)
		} else {
			fmt.Printf("Handler returned value: %f\n", value)
		}
	}
}
