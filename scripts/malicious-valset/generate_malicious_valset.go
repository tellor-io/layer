/*
Malicious valset signature generator for testing validator slashing on testnet
WARNING: This script is for testing purposes only. It will slash and jail the validator whose private key was used to sign.

USAGE:
1. Get your validator's private key:
   echo y | ./layerd keys export validator --unarmored-hex --unsafe --keyring-backend test

2. Modify the configuration section below:
   - paste your private key (without 0x prefix) into privateKeyHex
   - set your account address in creatorAddress

3. Run the script:
   cd scripts && go run generate_malicious_valset.go

4. Use the generated CLI command to submit the malicious valset signature evidence

NOTES:
- script uses current timestamp automatically
- fake valset hash and power threshold are hardcoded for testing
- this will slash and jail the validator whose private key was used to sign
*/

package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	// ===== CONFIGURATION - MODIFY THESE VALUES =====

	// paste your validator's private key here (without 0x prefix)
	privateKeyHex := "YOUR_PRIVATE_KEY_HERE"

	// creator address (your account address that will submit the evidence)
	creatorAddress := "YOUR_CREATOR_ADDRESS_HERE"

	// ===== HARDCODED VALSET DATA =====

	// fake valset hash (different from any actual valset)
	fakeValsetHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	// power threshold (total voting power)
	powerThreshold := uint64(10000000)

	// valset timestamp (current time)
	valsetTimestamp := uint64(time.Now().UnixMilli())

	// ===== VALIDATION =====
	if privateKeyHex == "YOUR_PRIVATE_KEY_HERE" {
		log.Fatal("ERROR: Please set your private key in the script")
	}
	if creatorAddress == "YOUR_CREATOR_ADDRESS_HERE" {
		log.Fatal("ERROR: Please set the creator address in the script")
	}

	// ===== GENERATE VALSET SIGNATURE DATA =====

	fmt.Println("=== Generating Malicious Valset Signature Data ===")
	fmt.Printf("Valset Timestamp: %d\n", valsetTimestamp)
	fmt.Printf("Fake Valset Hash: %s\n", fakeValsetHash)
	fmt.Printf("Power Threshold: %d\n", powerThreshold)

	// convert private key
	privKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		log.Fatalf("Failed to decode private key: %v", err)
	}

	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		log.Fatalf("Failed to create ECDSA private key: %v", err)
	}

	// get public address for verification
	pubKey := privKey.Public().(*ecdsa.PublicKey)
	evmAddress := crypto.PubkeyToAddress(*pubKey)
	fmt.Printf("EVM Address: %s\n", evmAddress.Hex())

	// encode the malicious valset checkpoint
	maliciousCheckpoint, err := encodeValsetCheckpoint(powerThreshold, valsetTimestamp, fakeValsetHash)
	if err != nil {
		log.Fatalf("Failed to encode valset checkpoint: %v", err)
	}

	// sign the checkpoint
	msgHash := sha256.Sum256(maliciousCheckpoint)
	signature, err := crypto.Sign(msgHash[:], privKey)
	if err != nil {
		log.Fatalf("Failed to sign valset checkpoint: %v", err)
	}

	// remove recovery ID (V) - bridge module expects only R || S (64 bytes)
	signature = signature[:64]
	sigHex := hex.EncodeToString(signature)

	fmt.Println("\n=== Signature Generated ===")
	fmt.Printf("Signature (64 bytes): %s\n", sigHex)

	// ===== GENERATE CLI COMMAND =====

	fmt.Println("\n=== CLI Command ===")
	fmt.Println("Run the following command to submit the malicious valset signature evidence:")
	fmt.Println()

	cliCommand := fmt.Sprintf(
		"./layerd tx bridge submit-valset-signature-evidence %s %d %s %d %s --from %s --keyring-backend test --chain-id layertest-4 --fees 500loya",
		creatorAddress,
		valsetTimestamp,
		fakeValsetHash,
		powerThreshold,
		sigHex,
		creatorAddress,
	)

	fmt.Println(cliCommand)

	fmt.Println("\n=== Parameter Breakdown ===")
	fmt.Printf("Creator Address: %s\n", creatorAddress)
	fmt.Printf("Valset Timestamp: %d\n", valsetTimestamp)
	fmt.Printf("Valset Hash: %s\n", fakeValsetHash)
	fmt.Printf("Power Threshold: %d\n", powerThreshold)
	fmt.Printf("Signature: %s\n", sigHex)
}

// encodeValsetCheckpoint replicates the keeper's EncodeValsetCheckpoint function
func encodeValsetCheckpoint(powerThreshold, validatorTimestamp uint64, validatorSetHash string) ([]byte, error) {
	// define the domain separator for the validator set hash, fixed size 32 bytes
	VALIDATOR_SET_HASH_DOMAIN_SEPARATOR := []byte("checkpoint")
	var domainSeparatorFixSize [32]byte
	copy(domainSeparatorFixSize[:], VALIDATOR_SET_HASH_DOMAIN_SEPARATOR)

	// convert validatorSetHash to bytes
	validatorSetHashBytes, err := hex.DecodeString(validatorSetHash)
	if err != nil {
		return nil, err
	}

	// convert validatorSetHash to a fixed size 32 bytes
	var validatorSetHashFixSize [32]byte
	copy(validatorSetHashFixSize[:], validatorSetHashBytes)

	// convert powerThreshold and validatorTimestamp to *big.Int for ABI encoding
	powerThresholdBigInt := new(big.Int).SetUint64(powerThreshold)
	validatorTimestampBigInt := new(big.Int).SetUint64(validatorTimestamp)

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	// prepare the types for encoding
	arguments := abi.Arguments{
		{Type: bytes32Type},
		{Type: uint256Type},
		{Type: uint256Type},
		{Type: bytes32Type},
	}

	// encode the arguments
	encodedCheckpointData, err := arguments.Pack(
		domainSeparatorFixSize,
		powerThresholdBigInt,
		validatorTimestampBigInt,
		validatorSetHashFixSize,
	)
	if err != nil {
		return nil, err
	}

	checkpoint := crypto.Keccak256(encodedCheckpointData)
	return checkpoint, nil
}
