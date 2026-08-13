package keeper_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	cosmossecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// TestGoldenVector_OracleAttestation produces a byte-exact golden vector for the
// Tellor oracle-attestation snapshot encoding using the node's REAL
// Keeper.EncodeOracleAttestationData method. This is the oracle the bridge-signer
// port must match byte-for-byte.
//
// It also proves the two signing paths are equivalent:
//   - node/checkpoint path: cosmos secp256k1 keyring Sign(snapshot), which
//     INTERNALLY does sha256(snapshot) then returns 64-byte R||S (lower-S),
//     stripping the leading compact-recovery byte.
//   - signer port path: geth crypto.Sign(sha256(snapshot), priv)[:64].
//
// Both must yield the IDENTICAL 64-byte R||S.
func TestGoldenVector_OracleAttestation(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	_ = ctx

	// ---- FIXED, HARDCODED INPUTS ----------------------------------------
	// queryId: 32 bytes. This is the SpotPrice(trb,usd) queryId used widely in
	// Tellor; documented as a literal 32-byte hex below so the signer test can
	// hardcode the exact same bytes.
	const queryIdHex = "83245f6a6a2f6458558a706270fbcc35ac3a81917602c1313d3bfa998dcc2d4b"
	queryId, err := hex.DecodeString(queryIdHex)
	require.NoError(t, err)
	require.Len(t, queryId, 32)

	// value: an ABI-encoded uint256 price (18 decimals). 0x...0de0b6b3a7640000
	// == 1e18 == "1.0". Passed WITH a 0x prefix to exercise Remove0xPrefix.
	const value = "0x0000000000000000000000000000000000000000000000000de0b6b3a7640000"

	// 6 uint256 scalars (timestamps are UNIX MILLISECONDS; power is a count).
	const timestamp uint64 = 1700000000000              // report aggregate timestamp (ms)
	const aggregatePower uint64 = 175                    // total reporter power
	const previousTimestamp uint64 = 1699999000000       // ms
	const nextTimestamp uint64 = 1700001000000           // ms
	const attestationTimestamp uint64 = 1700000500000    // ms
	const lastConsensusTimestamp uint64 = 1699998000000  // ms

	// valsetCheckpoint: 32 bytes. Fixed literal so signer can reuse verbatim.
	const valsetCheckpointHex = "5c3d8e1f0a9b7c6d4e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4f3a2b1c0d"
	valsetCheckpoint, err := hex.DecodeString(valsetCheckpointHex)
	require.NoError(t, err)
	require.Len(t, valsetCheckpoint, 32)

	// domain separator is FIXED inside the encoder:
	//   "tellorCurrentAttestation" ascii, copied into [32]byte (right zero-pad).
	domainSep := make([]byte, 32)
	copy(domainSep, []byte("tellorCurrentAttestation"))

	fmt.Println("=================== GOLDEN VECTOR INPUTS ===================")
	fmt.Printf("domainSeparator(ascii)=%q\n", "tellorCurrentAttestation")
	fmt.Printf("domainSeparator(bytes32 hex)=%s\n", hex.EncodeToString(domainSep))
	fmt.Printf("queryId(bytes32 hex)=%s\n", queryIdHex)
	fmt.Printf("value(string, 0x-prefixed)=%s\n", value)
	fmt.Printf("timestamp=%d\n", timestamp)
	fmt.Printf("aggregatePower=%d\n", aggregatePower)
	fmt.Printf("previousTimestamp=%d\n", previousTimestamp)
	fmt.Printf("nextTimestamp=%d\n", nextTimestamp)
	fmt.Printf("valsetCheckpoint(bytes32 hex)=%s\n", valsetCheckpointHex)
	fmt.Printf("attestationTimestamp=%d\n", attestationTimestamp)
	fmt.Printf("lastConsensusTimestamp=%d\n", lastConsensusTimestamp)

	// ---- 1. EncodeOracleAttestationData (REAL node method) --------------
	snapshot, err := k.EncodeOracleAttestationData(
		queryId,
		value,
		timestamp,
		aggregatePower,
		previousTimestamp,
		nextTimestamp,
		valsetCheckpoint,
		attestationTimestamp,
		lastConsensusTimestamp,
	)
	require.NoError(t, err)
	require.Len(t, snapshot, 32, "snapshot is keccak256(abi.Pack(...)) = 32 bytes")

	fmt.Println("=================== SNAPSHOT (keccak256 of ABI pack) =======")
	fmt.Printf("snapshotHash(hex)=%s\n", hex.EncodeToString(snapshot))

	// ---- 2. Signing digest -- what is actually signed -------------------
	// The keyring Sign() internally computes sha256(snapshot). Surface it so
	// the signer's sha256->SignRaw path can target the exact same 32 bytes.
	digest32 := sha256.Sum256(snapshot)
	signingDigest := digest32[:]
	fmt.Println("=================== SIGNING DIGEST (sha256 of snapshot) ====")
	fmt.Printf("signingDigest_sha256(hex)=%s\n", hex.EncodeToString(signingDigest))

	// ---- 3. FIXED private key: 32 bytes of 0x11 -------------------------
	priv := bytes.Repeat([]byte{0x11}, 32)
	fmt.Printf("privKey(hex)=%s\n", hex.EncodeToString(priv))

	// ---- 3a. SIGNER PORT PATH: geth crypto.Sign(sha256(snapshot)) -------
	gethPriv, err := crypto.ToECDSA(priv)
	require.NoError(t, err)
	gethSig65, err := crypto.Sign(signingDigest, gethPriv) // 65 bytes: R||S||V
	require.NoError(t, err)
	require.Len(t, gethSig65, 65)
	gethSig64 := gethSig65[:64] // R||S

	// ---- 3b. NODE/CHECKPOINT PATH: cosmos keyring Sign(snapshot) --------
	// PrivKey.Sign(msg) = ecdsa.SignCompact(priv, sha256(msg), false)[1:]
	// i.e. it sha256-hashes the message itself, then strips the compact
	// recovery byte, returning 64-byte R||S in lower-S form.
	cosmosPriv := &cosmossecp256k1.PrivKey{Key: priv}
	cosmosSig, err := cosmosPriv.Sign(snapshot) // pass UNHASHED snapshot
	require.NoError(t, err)
	require.Len(t, cosmosSig, 64)

	fmt.Println("=================== SIGNATURES (64-byte R||S) =============")
	fmt.Printf("sig64_geth(hex)        =%s\n", hex.EncodeToString(gethSig64))
	fmt.Printf("sig64_cosmosKeyring(hex)=%s\n", hex.EncodeToString(cosmosSig))

	// ---- 4. PARITY ASSERTION --------------------------------------------
	// Prove the signer's geth sha256->SignRaw[:64] path == node's keyring path.
	require.Equal(t, gethSig64, cosmosSig,
		"geth crypto.Sign(sha256(snapshot))[:64] must equal cosmos keyring Sign(snapshot)")

	// ---- 5. Sanity: verify the signature against the derived pubkey -----
	cosmosPub := cosmosPriv.PubKey()
	require.True(t, cosmosPub.VerifySignature(snapshot, cosmosSig),
		"cosmos pubkey must verify the 64-byte signature over the snapshot")

	fmt.Println("=================== GOLDEN VECTOR (COPY VERBATIM) =========")
	fmt.Printf("SNAPSHOT_HASH      = %s\n", hex.EncodeToString(snapshot))
	fmt.Printf("SIGNING_DIGEST_SHA = %s\n", hex.EncodeToString(signingDigest))
	fmt.Printf("SIG64_RS           = %s\n", hex.EncodeToString(gethSig64))
	fmt.Println("===========================================================")
}
