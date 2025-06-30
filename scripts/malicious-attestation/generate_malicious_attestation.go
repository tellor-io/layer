/*
Malicious attestation generator for testing validator slashing on testnet
WARNING: This script is for testing purposes only. It will slash and jail the validator whose private key was used to sign.

USAGE:
1. Get your validator's private key:
   echo y | ./layerd keys export validator --unarmored-hex --unsafe --keyring-backend test

2. Get the current validator checkpoint:
   ./layerd query bridge get-validator-checkpoint --node https://rpc-testnet.tellor.io:443

3. Modify the configuration section below:
   - paste your private key (without 0x prefix) into privateKeyHex
   - paste the checkpoint (without 0x prefix) into checkpointHex
   - set your account address in creatorAddress

4. Run the script:
   cd scripts && go run generate_malicious_attestation.go

5. Use the generated CLI command to submit the malicious attestation evidence

NOTES:
- script uses current timestamps automatically
- all other values are hardcoded for testing
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

	// paste the current validator checkpoint from the chain here (without 0x prefix)
	checkpointHex := "YOUR_CHECKPOINT_HERE"

	// creator address (your account address that will submit the evidence)
	creatorAddress := "YOUR_CREATOR_ADDRESS_HERE"

	// ===== HARDCODED ATTESTATION DATA =====

	// hardcoded query ID for testing
	queryId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"

	// malicious value (different from what was actually reported)
	maliciousValue := "000000000000000000000000000000000000000000000058528649cf90ee0000"

	// timestamps and other data
	currentTime := uint64(time.Now().UnixMilli())
	timestamp := currentTime - 1000       // report timestamp (slightly in past)
	attestationTimestamp := currentTime   // when attestation was made
	lastConsensusTimestamp := currentTime // last consensus time
	aggregatePower := uint64(1000000)     // voting power
	previousTimestamp := timestamp - 3000 // previous report timestamp
	nextTimestamp := uint64(0)            // next timestamp (0 if none)

	// ===== VALIDATION =====
	if privateKeyHex == "YOUR_PRIVATE_KEY_HERE" {
		log.Fatal("ERROR: Please set your private key in the script")
	}
	if checkpointHex == "YOUR_CHECKPOINT_HERE" {
		log.Fatal("ERROR: Please set the validator checkpoint in the script")
	}
	if creatorAddress == "YOUR_CREATOR_ADDRESS_HERE" {
		log.Fatal("ERROR: Please set the creator address in the script")
	}

	// ===== GENERATE ATTESTATION DATA =====

	fmt.Println("=== Generating Malicious Attestation Data ===")
	fmt.Printf("Query ID: %s\n", queryId)
	fmt.Printf("Malicious Value: %s\n", maliciousValue)
	fmt.Printf("Timestamp: %d\n", timestamp)
	fmt.Printf("Attestation Timestamp: %d\n", attestationTimestamp)
	fmt.Printf("Checkpoint: %s\n", checkpointHex)

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

	// convert inputs to bytes
	queryIdBytes, err := hex.DecodeString(queryId)
	if err != nil {
		log.Fatalf("Failed to decode query ID: %v", err)
	}

	checkpointBytes, err := hex.DecodeString(checkpointHex)
	if err != nil {
		log.Fatalf("Failed to decode checkpoint: %v", err)
	}

	// encode attestation data for signing
	snapshotBytes, err := encodeOracleAttestationData(
		queryIdBytes,
		maliciousValue,
		timestamp,
		aggregatePower,
		previousTimestamp,
		nextTimestamp,
		checkpointBytes,
		attestationTimestamp,
		lastConsensusTimestamp,
	)
	if err != nil {
		log.Fatalf("Failed to encode attestation data: %v", err)
	}

	// sign the snapshot
	msgHash := sha256.Sum256(snapshotBytes)
	signature, err := crypto.Sign(msgHash[:], privKey)
	if err != nil {
		log.Fatalf("Failed to sign attestation: %v", err)
	}

	// remove recovery ID (V) - bridge module expects only R || S (64 bytes)
	signature = signature[:64]
	sigHex := hex.EncodeToString(signature)

	fmt.Println("\n=== Signature Generated ===")
	fmt.Printf("Signature (64 bytes): %s\n", sigHex)

	// ===== GENERATE CLI COMMAND =====

	fmt.Println("\n=== CLI Command ===")
	fmt.Println("Run the following command to submit the malicious attestation evidence:")
	fmt.Println()

	cliCommand := fmt.Sprintf(
		"./layerd tx bridge submit-attestation-evidence %s %s %s %d %d %d %d %s %d %d %s --from %s --keyring-backend test --chain-id layertest-4 --fees 500loya",
		creatorAddress,
		queryId,
		maliciousValue,
		timestamp,
		aggregatePower,
		previousTimestamp,
		nextTimestamp,
		checkpointHex,
		attestationTimestamp,
		lastConsensusTimestamp,
		sigHex,
		creatorAddress,
	)

	fmt.Println(cliCommand)

	fmt.Println("\n=== Parameter Breakdown ===")
	fmt.Printf("Creator Address: %s\n", creatorAddress)
	fmt.Printf("Query ID: %s\n", queryId)
	fmt.Printf("Value: %s\n", maliciousValue)
	fmt.Printf("Timestamp: %d\n", timestamp)
	fmt.Printf("Aggregate Power: %d\n", aggregatePower)
	fmt.Printf("Previous Timestamp: %d\n", previousTimestamp)
	fmt.Printf("Next Timestamp: %d\n", nextTimestamp)
	fmt.Printf("Checkpoint: %s\n", checkpointHex)
	fmt.Printf("Attestation Timestamp: %d\n", attestationTimestamp)
	fmt.Printf("Last Consensus Timestamp: %d\n", lastConsensusTimestamp)
	fmt.Printf("Signature: %s\n", sigHex)
}

// encodeOracleAttestationData encodes attestation data for signing
// this must match the keeper.EncodeOracleAttestationData function exactly
func encodeOracleAttestationData(
	queryId []byte,
	value string,
	timestamp uint64,
	aggregatePower uint64,
	previousTimestamp uint64,
	nextTimestamp uint64,
	checkpoint []byte,
	attestationTimestamp uint64,
	lastConsensusTimestamp uint64,
) ([]byte, error) {
	// domain separator is bytes "tellorCurrentAttestation"
	NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR := []byte("tellorCurrentAttestation")
	// convert domain separator to bytes32
	var domainSepBytes32 [32]byte
	copy(domainSepBytes32[:], NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR)

	// convert queryId to bytes32
	var queryIdBytes32 [32]byte
	copy(queryIdBytes32[:], queryId)

	// convert value to bytes
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}

	// convert timestamps and power to big.Int
	timestampBig := new(big.Int).SetUint64(timestamp)
	aggregatePowerBig := new(big.Int).SetUint64(aggregatePower)
	previousTimestampBig := new(big.Int).SetUint64(previousTimestamp)
	nextTimestampBig := new(big.Int).SetUint64(nextTimestamp)
	attestationTimestampBig := new(big.Int).SetUint64(attestationTimestamp)
	lastConsensusTimestampBig := new(big.Int).SetUint64(lastConsensusTimestamp)

	// convert checkpoint to bytes32
	var checkpointBytes32 [32]byte
	copy(checkpointBytes32[:], checkpoint)

	// prepare ABI encoding types
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	arguments := abi.Arguments{
		{Type: bytes32Type}, // domain separator
		{Type: bytes32Type}, // queryId
		{Type: bytesType},   // value
		{Type: uint256Type}, // timestamp
		{Type: uint256Type}, // aggregatePower
		{Type: uint256Type}, // previousTimestamp
		{Type: uint256Type}, // nextTimestamp
		{Type: bytes32Type}, // checkpoint
		{Type: uint256Type}, // attestationTimestamp
		{Type: uint256Type}, // lastConsensusTimestamp
	}

	encodedData, err := arguments.Pack(
		domainSepBytes32,
		queryIdBytes32,
		valueBytes,
		timestampBig,
		aggregatePowerBig,
		previousTimestampBig,
		nextTimestampBig,
		checkpointBytes32,
		attestationTimestampBig,
		lastConsensusTimestampBig,
	)
	if err != nil {
		return nil, err
	}

	return crypto.Keccak256(encodedData), nil
}
