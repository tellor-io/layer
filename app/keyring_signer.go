package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
	signerv1 "github.com/tellor-io/bridge-remote-signer/api/gen/signer/v1"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// KeyringSignerConfig holds resolved config for the keyring signer.
type KeyringSignerConfig struct {
	KeyName        string
	KeyringBackend string
	KeyringDir     string
}

// KeyringSigner implements VoteExtensionSigner using the Cosmos SDK keyring.
type KeyringSigner struct {
	cfg             KeyringSignerConfig
	codec           codec.Codec
	kr              keyring.Keyring
	operatorAddress string // resolved at construction, cached
}

var _ VoteExtensionSigner = (*KeyringSigner)(nil)

// NewKeyringSignerFromViper reads config from viper at startup.
func NewKeyringSignerFromViper(appCodec codec.Codec) (*KeyringSigner, error) {
	keyName := viper.GetString("key-name")
	if keyName == "" {
		return nil, errors.New("key-name not set, please use --key-name flag")
	}

	krBackend := viper.GetString("keyring-backend")
	if krBackend == "" {
		return nil, errors.New("keyring-backend not set, please use --keyring-backend flag")
	}

	krDir := viper.GetString("keyring-dir")
	if krDir == "" {
		krDir = viper.GetString("home")
	}
	if krDir == "" {
		return nil, errors.New("keyring directory not set, please use --home or --keyring-dir flag")
	}

	kr, err := keyring.New(sdk.KeyringServiceName(), krBackend, krDir, os.Stdin, appCodec)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keyring: %w", err)
	}

	s := &KeyringSigner{
		cfg: KeyringSignerConfig{
			KeyName:        keyName,
			KeyringBackend: krBackend,
			KeyringDir:     krDir,
		},
		codec: appCodec,
		kr:    kr,
	}

	addr, err := s.resolveOperatorAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator address: %w", err)
	}
	s.operatorAddress = addr

	return s, nil
}

// SignCheckpoint signs the expected checkpoint bytes with the local key.
func (s *KeyringSigner) SignCheckpoint(_ context.Context, req *signerv1.SignBridgeCheckpointRequest) ([]byte, error) {
	return s.sign(req.ExpectedCheckpoint)
}

// SignOracleAttestation signs the expected snapshot bytes with the local key.
func (s *KeyringSigner) SignOracleAttestation(_ context.Context, req *signerv1.SignOracleAttestationRequest) ([]byte, error) {
	return s.sign(req.ExpectedSnapshot)
}

// SignInitial signs the two registration messages for the cached operator.
func (s *KeyringSigner) SignInitial(_ context.Context) ([]byte, []byte, error) {
	if s.operatorAddress == "" {
		return nil, nil, errors.New("operator address not initialized")
	}
	hashA, hashB := bridgetypes.InitialRegistrationDigests(s.operatorAddress)
	sigA, err := s.sign(hashA[:])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign message A: %w", err)
	}
	sigB, err := s.sign(hashB[:])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign message B: %w", err)
	}
	return sigA, sigB, nil
}

// GetOperatorAddress returns the cached bech32 validator operator address.
func (s *KeyringSigner) GetOperatorAddress(_ context.Context) (string, error) {
	if s.operatorAddress == "" {
		return "", errors.New("operator address not initialized")
	}
	return s.operatorAddress, nil
}

// sign signs msg with the keyring key; cosmos secp256k1 keys sign sha256(msg).
func (s *KeyringSigner) sign(msg []byte) ([]byte, error) {
	const secp256k1SignMode = 1
	sig, _, err := s.kr.Sign(s.cfg.KeyName, msg, secp256k1SignMode)
	if err != nil {
		return nil, fmt.Errorf("keyring sign failed: %w", err)
	}
	return sig, nil
}

// resolveOperatorAddress derives the bech32 validator address from the keyring public key.
func (s *KeyringSigner) resolveOperatorAddress() (string, error) {
	record, err := s.kr.Key(s.cfg.KeyName)
	if err != nil {
		return "", fmt.Errorf("failed to get key %q from keyring: %w", s.cfg.KeyName, err)
	}

	pubKey, err := record.GetPubKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key for %q: %w", s.cfg.KeyName, err)
	}

	config := sdk.GetConfig()
	bech32ValAddr, err := sdk.Bech32ifyAddressBytes(
		config.GetBech32ValidatorAddrPrefix(),
		pubKey.Address().Bytes(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to bech32-encode validator address: %w", err)
	}

	return bech32ValAddr, nil
}
