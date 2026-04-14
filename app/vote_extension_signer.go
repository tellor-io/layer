package app

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/viper"
)

// VoteExtensionSigner is the interface VoteExtHandler uses for all bridge signing.
// Implemented by KeyringSigner or GRPCRemoteSigner.
type VoteExtensionSigner interface {
	// Sign signs a 32-byte message and returns a secp256k1 signature.
	Sign(ctx context.Context, msg []byte) ([]byte, error)

	// GetOperatorAddress returns the bech32 validator operator address.
	GetOperatorAddress(ctx context.Context) (string, error)
}

func NewVoteExtensionSigner(appCodec codec.Codec) (VoteExtensionSigner, error) {

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

	keyName := viper.GetString("key-name")
	if keyName == "" {
		return nil, nil
	}

	return NewKeyringSignerFromViper(appCodec)
}
