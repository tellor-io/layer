// bridge_deposit_tips.go
//
// This script loops through bridge deposit IDs from 1 to 130, encodes the query data
// correctly for each deposit ID, and submits a tip transaction to the chain with
// chain ID "layer-internal".
//
// Usage:
//
//	go run scripts/tips/bridge_deposit_tips.go
//
// Configuration (edit the variables in main()):
//   - chainID: Chain ID to submit tips to (default: "layer-internal")
//   - from: Key name to use for signing transactions (default: "alice")
//   - keyringBackend: Keyring backend (default: "test")
//   - keyringDir: Path to keyring directory (default: "$HOME/.layer/alice")
//   - tipAmount: Amount to tip for each deposit (default: "10000loya")
//   - fees: Transaction fees (default: "500loya")
//
// Environment variables:
//   - LAYERD_PATH: Path to layerd binary (default: "./layerd")
//   - HOME: Home directory (required)
package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// encodeBridgeDepositQueryData encodes query data for a bridge deposit
// Format: abi.encode(string "TRBBridge", abi.encode(bool true, uint256 depositId))
func encodeBridgeDepositQueryData(depositId uint64) (string, error) {
	queryTypeString := "TRBBridge"
	toLayerBool := true
	depositIdUint64 := new(big.Int).SetUint64(depositId)

	// prepare encoding
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create string type: %w", err)
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create uint256 type: %w", err)
	}
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create bool type: %w", err)
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create bytes type: %w", err)
	}

	// encode query data arguments first
	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}
	queryDataArgsEncoded, err := queryDataArgs.Pack(toLayerBool, depositIdUint64)
	if err != nil {
		return "", fmt.Errorf("failed to pack query data args: %w", err)
	}

	// encode query data
	finalArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryDataEncoded, err := finalArgs.Pack(queryTypeString, queryDataArgsEncoded)
	if err != nil {
		return "", fmt.Errorf("failed to pack final args: %w", err)
	}

	// convert to hex string (without 0x prefix)
	queryDataHex := hex.EncodeToString(queryDataEncoded)
	// Ensure no "0x" prefix (strip it if present)
	queryDataHex = strings.TrimPrefix(queryDataHex, "0x")
	queryDataHex = strings.TrimPrefix(queryDataHex, "0X")
	return queryDataHex, nil
}

// submitTip submits a tip transaction using the layerd CLI
func submitTip(queryDataHex, tipAmount, layerdPath string) error {
	cmd := exec.Command(layerdPath, "tx", "oracle", "tip",
		queryDataHex,
		tipAmount,
		"--chain-id", "layer-internal",
		"--from", "chelsea",
		"--keyring-backend", "test",
		"--keyring-dir", "/home/ubuntu/.layer/chelsea",
		"--fees", "10loya",
		"--unordered",
		"--timeout-duration", "30s",
		"--yes",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to submit tip: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Tip submitted successfully. Output: %s\n", string(output))
	return nil
}

func main() {
	// Configuration
	chainID := "layer-internal"

	// Get layerd command path (default to ./layerd, can be overridden with LAYERD_PATH env var)
	layerdPath := "/home/ubuntu/layer/layerd"

	tipAmount := "10000loya" // Change this to your desired tip amount
	fees := "10loya"         // Transaction fees

	// Loop from 1 to 130
	fmt.Printf("Starting to submit tips for bridge deposits 1-130 on chain %s\n", chainID)
	fmt.Printf("Tip amount: %s, Fees: %s\n", tipAmount, fees)
	fmt.Println()

	for depositId := uint64(1); depositId <= 130; depositId++ {
		fmt.Printf("Processing deposit ID %d...\n", depositId)

		// Encode query data
		queryDataHex, err := encodeBridgeDepositQueryData(depositId)
		if err != nil {
			log.Printf("Error encoding query data for deposit ID %d: %v\n", depositId, err)
			continue
		}

		fmt.Printf("  Encoded query data: %s\n", queryDataHex)

		// Submit tip
		err = submitTip(queryDataHex, tipAmount, layerdPath)
		if err != nil {
			log.Printf("Error submitting tip for deposit ID %d: %v\n", depositId, err)
			// Continue with next deposit even if this one fails
			continue
		}

		fmt.Printf("  ✓ Successfully submitted tip for deposit ID %d\n", depositId)
		fmt.Println()

		// Add a small delay between transactions to avoid overwhelming the chain
		time.Sleep(1 * time.Second)
	}

	fmt.Println("Completed processing all bridge deposits (1-130)")
}
