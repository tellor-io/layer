package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/tellor-io/layer/x/bridge/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// TestGoldenVector_BridgeCheckpoint produces a byte-exact golden vector for the
// bridge checkpoint encoding using the node's REAL EncodeAndHashValidatorSet and
// EncodeValsetCheckpoint methods. This is the oracle the signer port must match.
func TestGoldenVector_BridgeCheckpoint(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)

	// ---- FIXED, HARDCODED INPUTS ----------------------------------------
	// Three validators with explicit 20-byte ethereum addresses + powers.
	// NOTE: these are provided ALREADY in the node's canonical sort order
	// (power descending; ties broken by ascending address byte compare).
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	addr3 := common.HexToAddress("0x3333333333333333333333333333333333333333")

	vs := &types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{EthereumAddress: addr1.Bytes(), Power: 100},
			{EthereumAddress: addr2.Bytes(), Power: 50},
			{EthereumAddress: addr3.Bytes(), Power: 25},
		},
	}

	const powerThreshold uint64 = 116          // (100+50+25)*2/3 = 116
	const validatorTimestamp uint64 = 1700000000000 // UNIX MILLISECONDS

	fmt.Println("=================== GOLDEN VECTOR INPUTS ===================")
	for i, v := range vs.BridgeValidatorSet {
		fmt.Printf("validator[%d] addr=0x%s power=%d\n", i, hex.EncodeToString(v.EthereumAddress), v.Power)
	}
	fmt.Printf("powerThreshold=%d\n", powerThreshold)
	fmt.Printf("validatorTimestamp=%d (UNIX MILLISECONDS)\n", validatorTimestamp)

	// ---- 1. EncodeAndHashValidatorSet (REAL node method) ----------------
	encodedValset, valsetHash, err := k.EncodeAndHashValidatorSet(ctx, vs)
	if err != nil {
		t.Fatalf("EncodeAndHashValidatorSet: %v", err)
	}
	fmt.Println("=================== VALIDATOR SET ENCODING =================")
	fmt.Printf("encodedValidatorSet(hex)=%s\n", hex.EncodeToString(encodedValset))
	fmt.Printf("validatorSetHash(hex)=%s\n", hex.EncodeToString(valsetHash))

	// ---- 2a. MAINNET domain separator -----------------------------------
	// Mainnet fixed constant: "checkpoint" ascii, right-padded to 32 bytes.
	domainSepMainnet := make([]byte, 32)
	copy(domainSepMainnet, []byte("checkpoint"))
	if err := k.ValsetCheckpointDomainSeparator.Set(ctx, domainSepMainnet); err != nil {
		t.Fatalf("set mainnet domsep: %v", err)
	}
	checkpointMainnet, err := k.EncodeValsetCheckpoint(ctx, powerThreshold, validatorTimestamp, valsetHash)
	if err != nil {
		t.Fatalf("EncodeValsetCheckpoint mainnet: %v", err)
	}

	// ---- 2b. NON-MAINNET domain separator -------------------------------
	// keccak256(abi.encode("checkpoint", chainID)) with chainID = "layertest-4"
	const nonMainnetChainID = "layertest-4"
	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		t.Fatalf("abi string type: %v", err)
	}
	dsArgs := abi.Arguments{{Type: stringType}, {Type: stringType}}
	dsEncoded, err := dsArgs.Pack("checkpoint", nonMainnetChainID)
	if err != nil {
		t.Fatalf("pack domsep: %v", err)
	}
	domainSepNonMainnet := crypto.Keccak256(dsEncoded)
	if err := k.ValsetCheckpointDomainSeparator.Set(ctx, domainSepNonMainnet); err != nil {
		t.Fatalf("set non-mainnet domsep: %v", err)
	}
	checkpointNonMainnet, err := k.EncodeValsetCheckpoint(ctx, powerThreshold, validatorTimestamp, valsetHash)
	if err != nil {
		t.Fatalf("EncodeValsetCheckpoint non-mainnet: %v", err)
	}

	fmt.Println("=================== DOMAIN SEPARATORS ======================")
	fmt.Printf("domainSeparator_MAINNET(hex)=%s\n", hex.EncodeToString(domainSepMainnet))
	fmt.Printf("nonMainnetChainID=%q\n", nonMainnetChainID)
	fmt.Printf("domainSeparator_nonMainnet_encoded(hex)=%s\n", hex.EncodeToString(dsEncoded))
	fmt.Printf("domainSeparator_nonMainnet(hex)=%s\n", hex.EncodeToString(domainSepNonMainnet))

	fmt.Println("=================== CHECKPOINTS ============================")
	fmt.Printf("checkpoint_MAINNET(hex)=%s\n", hex.EncodeToString(checkpointMainnet))
	fmt.Printf("checkpoint_nonMainnet(hex)=%s\n", hex.EncodeToString(checkpointNonMainnet))

	// ---- 3. Signing digest -- what gets signed (sha256 of checkpoint) ---
	// The node signs via kr.Sign(checkpoint) which sha256-hashes the message
	// internally then produces 64-byte r||s. Surface sha256(checkpoint).
	digestMainnet := sha256sum(checkpointMainnet)
	digestNonMainnet := sha256sum(checkpointNonMainnet)
	fmt.Println("=================== SIGNING DIGESTS (sha256 of checkpoint) =")
	fmt.Printf("sha256(checkpoint_MAINNET)(hex)=%s\n", hex.EncodeToString(digestMainnet))
	fmt.Printf("sha256(checkpoint_nonMainnet)(hex)=%s\n", hex.EncodeToString(digestNonMainnet))
	fmt.Println("===========================================================")
}

func sha256sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}
