package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	cosmossecp "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestSigCheck_CosmosVsGeth(t *testing.T) {
	keyBytes, _ := hex.DecodeString("1111111111111111111111111111111111111111111111111111111111111111")
	checkpoint, _ := hex.DecodeString("ab090e41f0bc98246cce8eb74603375dcc3721bc212a6704f2e26980d04ee0f1")

	cosmosPriv := &cosmossecp.PrivKey{Key: keyBytes}
	cosmosSig, err := cosmosPriv.Sign(checkpoint)
	if err != nil { t.Fatal(err) }

	digest := sha256.Sum256(checkpoint)
	gethPriv, err := gethcrypto.ToECDSA(keyBytes)
	if err != nil { t.Fatal(err) }
	gethSig, err := gethcrypto.Sign(digest[:], gethPriv)
	if err != nil { t.Fatal(err) }
	gethRS := gethSig[:64]

	t.Logf("digest sha256(checkpoint) = %s", hex.EncodeToString(digest[:]))
	t.Logf("cosmos kr.Sign (R||S)     = %s  len=%d", hex.EncodeToString(cosmosSig), len(cosmosSig))
	t.Logf("geth Sign(digest)[:64]    = %s  len=%d", hex.EncodeToString(gethRS), len(gethRS))
	t.Logf("geth recovery V byte      = %02x", gethSig[64])
	if hex.EncodeToString(cosmosSig) != hex.EncodeToString(gethRS) {
		t.Fatalf("MISMATCH: cosmos and geth produced different R||S")
	}
	t.Logf("MATCH byte-for-byte       = true")
}
