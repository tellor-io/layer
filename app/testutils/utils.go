package testutils

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/testutil/sample"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TestAccount struct {
	Name    string
	Address sdk.AccAddress
}

func CreateTestContext(t *testing.T) sdk.Context {
	t.Helper()

	key := storetypes.NewKVStoreKey(oracletypes.StoreKey)

	testCtx := testutil.DefaultContextWithDB(
		t,
		key,
		storetypes.NewTransientStoreKey("test"),
	)

	// set vote ext enabled height
	testCtx.Ctx = testCtx.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 0,
		},
	})

	// set chain id
	testCtx.Ctx = testCtx.Ctx.WithChainID("layer")

	return testCtx.Ctx
}

func CreateKeyringAccounts(t *testing.T, kr keyring.Keyring, num int) []TestAccount {
	t.Helper()
	require := require.New(t)

	accounts := make([]TestAccount, num)
	for i := range accounts {
		record, _, err := kr.NewMnemonic(
			fmt.Sprintf("key-%d", i),
			keyring.English,
			sdk.FullFundraiserPath,
			keyring.DefaultBIP39Passphrase,
			hd.Secp256k1)
		require.NoError(err)

		addr, err := record.GetAddress()
		require.NoError(err)
		err = kr.Delete(fmt.Sprintf("key-%d", i))
		require.NoError(err)

		accounts[i] = TestAccount{Name: record.Name, Address: addr}
	}

	return accounts
}

func ClearKeyring(t *testing.T, kr keyring.Keyring) {
	t.Helper()

	records, err := kr.List()
	require.NoError(t, err)

	for _, record := range records {
		err = kr.Delete(record.Name)
		require.NoError(t, err)
	}
}

func GenerateSignatures(t *testing.T) (sigA, sigB []byte, addressExpected common.Address) {
	t.Helper()

	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	require.NotNil(t, privateKey)
	require.NoError(t, err)

	pkCoord := &ecdsa.PublicKey{
		X: privateKey.X,
		Y: privateKey.Y,
	}
	addressExpected = crypto.PubkeyToAddress(*pkCoord)

	msgA := "TellorLayer: Initial bridge signature A"
	msgB := "TellorLayer: Initial bridge signature B"
	msgBytesA := []byte(msgA)
	msgBytesB := []byte(msgB)

	// hash messages
	msgHashBytes32A := sha256.Sum256(msgBytesA)
	msgHashBytesA := msgHashBytes32A[:]

	msgHashBytes32B := sha256.Sum256(msgBytesB)
	msgHashBytesB := msgHashBytes32B[:]

	// hash the hash, since the keyring signer automatically hashes the message
	msgDoubleHashBytes32A := sha256.Sum256(msgHashBytesA)
	msgDoubleHashBytesA := msgDoubleHashBytes32A[:]

	msgDoubleHashBytes32B := sha256.Sum256(msgHashBytesB)
	msgDoubleHashBytesB := msgDoubleHashBytes32B[:]

	sigA, err = crypto.Sign(msgDoubleHashBytesA, privateKey)
	require.NoError(t, err)
	require.NotNil(t, sigA)

	sigB, err = crypto.Sign(msgDoubleHashBytesB, privateKey)
	require.NoError(t, err)
	require.NotNil(t, sigB)

	return sigA, sigB, addressExpected
}

func GenerateCommit(t *testing.T, ctx sdk.Context) (abcitypes.ExtendedCommitInfo, app.BridgeVoteExtension, common.Address, sdk.AccAddress, sdk.ConsAddress) {
	t.Helper()

	accAddr := sample.AccAddressBytes()
	consAddr := sdk.ConsAddress(accAddr)

	sigA, sigB, addressExpected := GenerateSignatures(t)

	voteExt := app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			{
				Attestation: []byte("attestation"),
				Snapshot:    []byte("snapshot"),
			},
		},
		InitialSignature: app.InitialSignature{
			SignatureA: sigA,
			SignatureB: sigB,
		},
		ValsetSignature: app.BridgeValsetSignature{
			Signature: []byte("signature"),
			Timestamp: uint64(ctx.BlockTime().Unix()),
		},
	}
	voteExtBytes, err := json.Marshal(voteExt)
	require.NoError(t, err)

	commit := abcitypes.ExtendedCommitInfo{
		Round: 1,
		Votes: []abcitypes.ExtendedVoteInfo{
			{
				Validator: abcitypes.Validator{
					Address: consAddr,
					Power:   1000,
				},
				VoteExtension:      voteExtBytes,
				ExtensionSignature: []byte("extensionSignature"),
				BlockIdFlag:        cmtproto.BlockIDFlagCommit,
			},
		},
	}

	return commit, voteExt, addressExpected, accAddr, consAddr
}
