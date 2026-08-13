package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func newTestKeyringSigner(t *testing.T) *KeyringSigner {
	t.Helper()

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	dir := t.TempDir()
	kr, err := keyring.New("test", keyring.BackendTest, dir, nil, cdc)
	require.NoError(t, err)

	_, _, err = kr.NewMnemonic("validator", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	require.NoError(t, err)

	s := &KeyringSigner{
		cfg: KeyringSignerConfig{
			KeyName:        "validator",
			KeyringBackend: keyring.BackendTest,
			KeyringDir:     dir,
		},
		codec: cdc,
		kr:    kr,
	}
	addr, err := s.resolveOperatorAddress()
	require.NoError(t, err)
	s.operatorAddress = addr
	return s
}

func recoverRegistration(sigA, sigB []byte, operatorAddress string) (common.Address, error) {
	msgA, msgB := bridgetypes.InitialRegistrationMessages(operatorAddress)
	hashA := sha256.Sum256([]byte(msgA))
	hashB := sha256.Sum256([]byte(msgB))
	digestA := sha256.Sum256(hashA[:])
	digestB := sha256.Sum256(hashB[:])
	addrsA, err := recoverBothIDs(sigA, digestA[:])
	if err != nil {
		return common.Address{}, err
	}
	addrsB, err := recoverBothIDs(sigB, digestB[:])
	if err != nil {
		return common.Address{}, err
	}
	if addrsA[0] == addrsB[0] || addrsA[0] == addrsB[1] {
		return addrsA[0], nil
	}
	if addrsA[1] == addrsB[0] || addrsA[1] == addrsB[1] {
		return addrsA[1], nil
	}
	return common.Address{}, errors.New("EVM addresses do not match")
}

func recoverBothIDs(sig, msgHash []byte) ([]common.Address, error) {
	addrs := make([]common.Address, 0, 2)
	for _, id := range []byte{0, 1} {
		sigWithID := append(append([]byte{}, sig[:64]...), id)
		pubKey, err := crypto.SigToPub(msgHash, sigWithID)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, crypto.PubkeyToAddress(*pubKey))
	}
	return addrs, nil
}

func TestKeyringSigner_SignInitialUsesCachedOperator(t *testing.T) {
	s := newTestKeyringSigner(t)
	cached, err := s.GetOperatorAddress(context.Background())
	require.NoError(t, err)

	sigA, sigB, err := s.SignInitial(context.Background())
	require.NoError(t, err)
	require.Len(t, sigA, 64)
	require.Len(t, sigB, 64)

	_, err = recoverRegistration(sigA, sigB, "tellorvaloper1foreignoperator000000000000000")
	require.Error(t, err)

	recovered, err := recoverRegistration(sigA, sigB, cached)
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, recovered)
}

func TestKeyringSigner_SignInitialRequiresCachedOperator(t *testing.T) {
	s := newTestKeyringSigner(t)
	s.operatorAddress = ""
	_, _, err := s.SignInitial(context.Background())
	require.Error(t, err)
}
