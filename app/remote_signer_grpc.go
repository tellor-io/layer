package app

import (
	"bytes"
	"context"
	"fmt"
	"time"

	signerv1 "github.com/tellor-io/bridge-remote-signer/api/gen/signer/v1"
	bridgetls "github.com/tellor-io/bridge-remote-signer/api/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cosmossecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GRPCSignerConfig holds the connection config for the bridge-remote-signer.
type GRPCSignerConfig struct {
	// Address is the signer's gRPC address, e.g. "dns:///signer-host:9191"
	Address string

	// Insecure explicitly opts into a plaintext connection (local/test only).
	// Without it all three cert paths are required.
	Insecure bool

	// CACert is the path to the CA certificate used to verify the signer.
	CACert string

	// ClientCert is the path to the validator's client TLS certificate.
	ClientCert string

	// ClientKey is the path to the validator's client TLS private key.
	ClientKey string

	// ServerName must match the CN in the signer's server certificate.
	// Default: "bridge-signer".
	ServerName string

	// RequestTimeout is the per-RPC deadline.
	// Must be less than CometBFT's vote extension timeout.
	// Default: 2s
	RequestTimeout time.Duration
}

// GRPCRemoteSigner implements VoteExtensionSigner by delegating signing to the
// bridge-remote-signer over gRPC via its structured, fail-closed RPCs.
type GRPCRemoteSigner struct {
	cfg             GRPCSignerConfig
	conn            *grpc.ClientConn
	client          signerv1.BridgeSignerClient
	operatorAddress string // derived from the signer's public key at startup, cached
}

var _ VoteExtensionSigner = (*GRPCRemoteSigner)(nil)

// NewGRPCRemoteSigner dials the signer, fetches the public key, derives
// the operator address locally, and caches it.
// No private key ever exists on the validator node.
func NewGRPCRemoteSigner(cfg GRPCSignerConfig) (*GRPCRemoteSigner, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("bridge signer address is required")
	}
	if cfg.ServerName == "" {
		cfg.ServerName = "bridge-signer"
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 2 * time.Second
	}

	var dialOpt grpc.DialOption
	if cfg.Insecure {
		dialOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		if cfg.CACert == "" || cfg.ClientCert == "" || cfg.ClientKey == "" {
			return nil, fmt.Errorf("remote signer mTLS config incomplete: --remote-signer-ca-cert, --remote-signer-client-cert and --remote-signer-client-key are all required (or set --remote-signer-insecure for local testing)")
		}
		creds, err := bridgetls.NewClientCredentials(
			cfg.CACert,
			cfg.ClientCert,
			cfg.ClientKey,
			cfg.ServerName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build mTLS credentials: %w", err)
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	}

	// Dial the signer. grpc.NewClient does not block —
	// the actual TCP connection is established on first use.
	conn, err := grpc.NewClient(cfg.Address, dialOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for signer at %q: %w", cfg.Address, err)
	}

	s := &GRPCRemoteSigner{
		cfg:    cfg,
		conn:   conn,
		client: signerv1.NewBridgeSignerClient(conn),
	}

	// Fetch the public key from the signer at startup and derive the operator
	// address locally. Fails fast if the signer is unreachable or misconfigured.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pubKeyResp, err := s.client.GetPublicKey(ctx, &signerv1.GetPublicKeyRequest{})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get public key from signer at %q: %w", cfg.Address, err)
	}

	operatorAddress, err := deriveOperatorAddressFromPubKey(pubKeyResp.PublicKey)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to derive operator address from signer public key: %w", err)
	}
	s.operatorAddress = operatorAddress

	return s, nil
}

// SignCheckpoint delegates to the fail-closed SignBridgeCheckpoint RPC; the
// signer recomputes the checkpoint and refuses to sign on mismatch.
func (s *GRPCRemoteSigner) SignCheckpoint(ctx context.Context, req *signerv1.SignBridgeCheckpointRequest) ([]byte, error) {
	rpcCtx, cancel := context.WithTimeout(ctx, s.cfg.RequestTimeout)
	defer cancel()

	resp, err := s.client.SignBridgeCheckpoint(rpcCtx, req)
	if err != nil {
		return nil, fmt.Errorf("SignBridgeCheckpoint RPC failed: %w", err)
	}
	if len(resp.Signature) != 64 {
		return nil, fmt.Errorf("SignBridgeCheckpoint: expected 64-byte signature, got %d", len(resp.Signature))
	}
	if !bytes.Equal(resp.Checkpoint, req.ExpectedCheckpoint) {
		return nil, fmt.Errorf("SignBridgeCheckpoint: signer checkpoint does not match expected checkpoint")
	}
	return resp.Signature, nil
}

// SignOracleAttestation delegates to the fail-closed SignOracleAttestation
// RPC; the signer recomputes the snapshot and refuses to sign on mismatch.
func (s *GRPCRemoteSigner) SignOracleAttestation(ctx context.Context, req *signerv1.SignOracleAttestationRequest) ([]byte, error) {
	rpcCtx, cancel := context.WithTimeout(ctx, s.cfg.RequestTimeout)
	defer cancel()

	resp, err := s.client.SignOracleAttestation(rpcCtx, req)
	if err != nil {
		return nil, fmt.Errorf("SignOracleAttestation RPC failed: %w", err)
	}
	if len(resp.Signature) != 64 {
		return nil, fmt.Errorf("SignOracleAttestation: expected 64-byte signature, got %d", len(resp.Signature))
	}
	if !bytes.Equal(resp.Snapshot, req.ExpectedSnapshot) {
		return nil, fmt.Errorf("SignOracleAttestation: signer snapshot does not match expected snapshot")
	}
	return resp.Signature, nil
}

// SignInitial calls the SignInitial RPC with the operator identity cached at construction.
func (s *GRPCRemoteSigner) SignInitial(ctx context.Context) ([]byte, []byte, error) {
	if s.operatorAddress == "" {
		return nil, nil, fmt.Errorf("operator address not initialized")
	}

	rpcCtx, cancel := context.WithTimeout(ctx, s.cfg.RequestTimeout)
	defer cancel()

	resp, err := s.client.SignInitial(rpcCtx, &signerv1.SignInitialRequest{
		OperatorAddress: s.operatorAddress,
		RequestId:       voteExtRequestID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("SignInitial RPC failed: %w", err)
	}
	if len(resp.SignatureA) != 64 || len(resp.SignatureB) != 64 {
		return nil, nil, fmt.Errorf("SignInitial: expected two 64 byte signatures, got %d and %d", len(resp.SignatureA), len(resp.SignatureB))
	}
	return resp.SignatureA, resp.SignatureB, nil
}

// GetOperatorAddress returns the cached operator address derived from the signer's public key.
func (s *GRPCRemoteSigner) GetOperatorAddress(_ context.Context) (string, error) {
	if s.operatorAddress == "" {
		return "", fmt.Errorf("operator address not initialized")
	}
	return s.operatorAddress, nil
}

// Close closes the underlying gRPC connection.
func (s *GRPCRemoteSigner) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// deriveOperatorAddressFromPubKey derives the bech32 validator operator address
// from a raw compressed secp256k1 public key using the Cosmos SDK.
// Uses SHA256 + RIPEMD160 (standard Cosmos address derivation).
func deriveOperatorAddressFromPubKey(compressedPubKey []byte) (string, error) {
	if len(compressedPubKey) != 33 {
		return "", fmt.Errorf("expected 33-byte compressed public key, got %d", len(compressedPubKey))
	}

	cosmosPubKey := &cosmossecp256k1.PubKey{Key: compressedPubKey}

	config := sdk.GetConfig()
	bech32ValAddr, err := sdk.Bech32ifyAddressBytes(
		config.GetBech32ValidatorAddrPrefix(),
		cosmosPubKey.Address().Bytes(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to bech32-encode validator address: %w", err)
	}

	return bech32ValAddr, nil
}
