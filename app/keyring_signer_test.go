package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
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

func TestKeyringSigner_SignInitialUsesCachedOperator(t *testing.T) {
	s := newTestKeyringSigner(t)
	cached, err := s.GetOperatorAddress(context.Background())
	require.NoError(t, err)

	sigA, sigB, err := s.SignInitial(context.Background())
	require.NoError(t, err)
	require.Len(t, sigA, 64)
	require.Len(t, sigB, 64)

	record, err := s.kr.Key(s.cfg.KeyName)
	require.NoError(t, err)
	pubKey, err := record.GetPubKey()
	require.NoError(t, err)

	hashA, hashB := bridgetypes.InitialRegistrationDigests(cached)
	require.True(t, pubKey.VerifySignature(hashA[:], sigA))
	require.True(t, pubKey.VerifySignature(hashB[:], sigB))

	foreignA, _ := bridgetypes.InitialRegistrationDigests("tellorvaloper1foreignoperator000000000000000")
	require.False(t, pubKey.VerifySignature(foreignA[:], sigA))
}

func TestKeyringSigner_SignInitialRequiresCachedOperator(t *testing.T) {
	s := newTestKeyringSigner(t)
	s.operatorAddress = ""
	_, _, err := s.SignInitial(context.Background())
	require.Error(t, err)
}

// failNthKeyring wraps a real keyring and fails the Nth Sign call (1-based).
type failNthKeyring struct {
	keyring.Keyring
	failOn int
	calls  int
	err    error
}

func (f *failNthKeyring) Sign(uid string, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
	f.calls++
	if f.calls == f.failOn {
		return nil, nil, f.err
	}
	return f.Keyring.Sign(uid, msg, signMode)
}

func TestKeyringSigner_SignInitialFailsMessageA(t *testing.T) {
	s := newTestKeyringSigner(t)
	s.kr = &failNthKeyring{Keyring: s.kr, failOn: 1, err: errors.New("boom")}
	_, _, err := s.SignInitial(context.Background())
	require.ErrorContains(t, err, "failed to sign message A")
}

func TestKeyringSigner_SignInitialFailsMessageB(t *testing.T) {
	s := newTestKeyringSigner(t)
	s.kr = &failNthKeyring{Keyring: s.kr, failOn: 2, err: errors.New("boom")}
	_, _, err := s.SignInitial(context.Background())
	require.ErrorContains(t, err, "failed to sign message B")
}
