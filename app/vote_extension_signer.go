package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/codec"
)

// VoteExtensionSigner is the interface VoteExtHandler uses for all bridge signing.
// Implemented by KeyringSigner or GRPCRemoteSigner.
type VoteExtensionSigner interface {
	// Sign signs a 32-byte message and returns a secp256k1 signature.
	Sign(ctx context.Context, msg []byte) ([]byte, error)

	// GetOperatorAddress returns the bech32 validator operator address.
	GetOperatorAddress(ctx context.Context) (string, error)
}

func RemoteVoteExtensionSigner(appCodec codec.Codec) (VoteExtensionSigner, error) {
	if viper.GetBool("remote-signer-enabled") {
		addr := viper.GetString("remote-signer-addr")
		if addr == "" {
			return nil, fmt.Errorf("--remote-signer-addr is required when --remote-signer-enabled is set")
		}
		return NewGRPCRemoteSigner(GRPCSignerConfig{
			Address:    addr,
			CACert:     viper.GetString("remote-signer-ca-cert"),
			ClientCert: viper.GetString("remote-signer-client-cert"),
			ClientKey:  viper.GetString("remote-signer-client-key"),
			ServerName: viper.GetString("remote-signer-server-name"),
		})
	}
	return nil, nil
}

// NewKeyringSignerFromViperIfSet is the lazy entry point used by the
// vote extension handler on first invocation. It reads --key-name from
// viper and tries to build a KeyringSigner if set.
func NewKeyringSignerFromViperIfSet(appCodec codec.Codec) (VoteExtensionSigner, error) {
	if viper.GetString("key-name") == "" {
		return nil, errors.New("no bridge signer configured; set --key-name or --remote-signer-enabled")
	}
	return NewKeyringSignerFromViper(appCodec)
}
