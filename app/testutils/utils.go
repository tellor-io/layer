package testutils

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	protoio "github.com/cosmos/gogoproto/io"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/testutil/sample"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/api/tendermint/abci"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OperatorAndEVM struct {
	OperatorAddresses []string `json:"operator_addresses"`
	EVMAddresses      []string `json:"evm_addresses"`
}

type ValsetSignatures struct {
	OperatorAddresses []string `json:"operator_addresses"`
	Timestamps        []int64  `json:"timestamps"`
	Signatures        []string `json:"signatures"`
}

type OracleAttestations struct {
	OperatorAddresses []string `json:"operator_addresses"`
	Attestations      [][]byte `json:"attestations"`
	Snapshots         [][]byte `json:"snapshots"`
}

type VoteExtTx struct {
	BlockHeight        int64                    `json:"block_height"`
	OpAndEVMAddrs      OperatorAndEVM           `json:"op_and_evm_addrs"`
	ValsetSigs         ValsetSignatures         `json:"valset_sigs"`
	OracleAttestations OracleAttestations       `json:"oracle_attestations"`
	ExtendedCommitInfo *abci.ExtendedCommitInfo `json:"extended_commit_info"`
}

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
			VoteExtensionsEnableHeight: 1,
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

func GenerateSignatures(t *testing.T) (sigA, sigB []byte, evmAddr common.Address) {
	t.Helper()

	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	require.NotNil(t, privateKey)
	require.NoError(t, err)

	pkCoord := &ecdsa.PublicKey{
		X: privateKey.X,
		Y: privateKey.Y,
	}
	evmAddr = crypto.PubkeyToAddress(*pkCoord)

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

	return sigA, sigB, evmAddr
}

func GenerateCommit(t *testing.T, ctx sdk.Context) (abcitypes.ExtendedCommitInfo, app.BridgeVoteExtension, common.Address, sdk.AccAddress, sdk.ConsAddress, []byte) {
	t.Helper()

	accAddr := sample.AccAddressBytes()
	consAddr := sdk.ConsAddress(accAddr)

	sigA, sigB, evmAddr := GenerateSignatures(t)

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

	extCommit := abcitypes.ExtendedCommitInfo{
		Round: 2,
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

	return extCommit, voteExt, evmAddr, accAddr, consAddr, voteExtBytes
}

// type extendedVoteInfos []abci.ExtendedVoteInfo

// func extendedCommitToLastCommit(commit abci.ExtendedCommitInfo) (abci.ExtendedCommitInfo, comet.BlockInfo) {
// 	// sort the extended commit info
// 	sort.Sort(extendedVoteInfos(commit.Votes))

// 	// convert the extended commit info to last commit info
// 	lastCommit := abci.CommitInfo{
// 		Round: ec.Round,
// 		Votes: make([]abci.VoteInfo, len(ec.Votes)),
// 	}

// 	for i, vote := range ec.Votes {
// 		lastCommit.Votes[i] = abci.VoteInfo{
// 			Validator: abci.Validator{
// 				Address: vote.Validator.Address,
// 				Power:   vote.Validator.Power,
// 			},
// 			BlockIdFlag: vote.BlockIdFlag,
// 		}
// 	}

//		return ec, baseapp.NewBlockInfo(
//			nil,
//			nil,
//			nil,
//			lastCommit,
//		)
//	}
func GenerateProposer(t *testing.T) (pubKey ed25519.PublicKey, privKey ed25519.PrivateKey, consAddr sdk.ConsAddress, accAddr sdk.AccAddress) {
	t.Helper()

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	accAddr = sdk.AccAddress(pubKey)
	consAddr = sdk.ConsAddress(pubKey)

	return pubKey, privKey, consAddr, accAddr
}

func GenerateVoteExtBz(t *testing.T, voteExt VoteExtTx, consAddr sdk.ConsAddress) ([]byte, OperatorAndEVM, ValsetSignatures, OracleAttestations, error) {
	t.Helper()

	sigA, sigB, evmAddr := GenerateSignatures(t)

	operatorAndEvm := OperatorAndEVM{
		OperatorAddresses: []string{consAddr.String()},
		EVMAddresses:      []string{evmAddr.String()},
	}
	voteExt.OpAndEVMAddrs = operatorAndEvm

	valsetSigs := ValsetSignatures{
		OperatorAddresses: []string{consAddr.String()},
		Timestamps:        []int64{1},
		Signatures:        []string{string(sigA), string(sigB)},
	}
	voteExt.ValsetSigs = valsetSigs

	oracleAttestations := OracleAttestations{
		OperatorAddresses: []string{consAddr.String()},
		Attestations:      [][]byte{[]byte("attestation1")},
		Snapshots:         [][]byte{[]byte("snapshot1")},
	}
	voteExt.OracleAttestations = oracleAttestations

	bz, err := json.Marshal(voteExt)
	if err != nil {
		return nil, operatorAndEvm, valsetSigs, oracleAttestations, err
	}

	return bz, operatorAndEvm, valsetSigs, oracleAttestations, nil
}

func SignCVE(veBz []byte, height, round int64, privKey ed25519.PrivateKey) (cmtproto.CanonicalVoteExtension, []byte, []byte, error) {
	cve := cmtproto.CanonicalVoteExtension{
		Extension: veBz,
		Height:    height,
		Round:     round,
		ChainId:   "layer",
	}

	marshalDelimitedFn := func(msg proto.Message) ([]byte, error) {
		var buf bytes.Buffer
		if err := protoio.NewDelimitedWriter(&buf).WriteMsg(msg); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	extSignBytes, err := marshalDelimitedFn(&cve)
	if err != nil {
		return cmtproto.CanonicalVoteExtension{}, nil, nil, err
	}

	// sign extSignBytes
	sig := ed25519.Sign(privKey, extSignBytes)

	return cve, extSignBytes, sig, nil
}

// func signInitialMessage() ([]byte, []byte, error) {
// 	messageA := "TellorLayer: Initial bridge signature A"
// 	messageB := "TellorLayer: Initial bridge signature B"

// 	// convert message to bytes
// 	msgBytesA := []byte(messageA)
// 	msgBytesB := []byte(messageB)

// 	// hash message
// 	msgHashABytes32 := sha256.Sum256(msgBytesA)
// 	msgHashBBytes32 := sha256.Sum256(msgBytesB)

// 	// convert [32]byte to []byte
// 	msgHashABytes := msgHashABytes32[:]
// 	msgHashBBytes := msgHashBBytes32[:]

// 	// sign message
// 	sigA, err := h.SignMessage(msgHashABytes)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	sigB, err := h.SignMessage(msgHashBBytes)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	return sigA, sigB, nil
// }

// func signMessage(msg []byte) ([]byte, error) {
// 	kr, err := h.GetKeyring()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get keyring: %w", err)
// 	}
// 	keyName := viper.GetString("key-name")
// 	if keyName == "" {
// 		return nil, fmt.Errorf("key name not found, please set --key-name flag")
// 	}
// 	sig, _, err := kr.Sign(keyName, msg, 1)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to sign message: %w", err)
// 	}
// 	return sig, nil
// }
