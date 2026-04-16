package app

import (
	"context"
	"fmt"
	"time"

	signerv1 "github.com/tellor-io/bridge-remote-signer/api/gen/signer/v1"
	bridgetls "github.com/tellor-io/bridge-remote-signer/api/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	cosmossecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GRPCSignerConfig holds the connection config for the remote signing sidecar.
type GRPCSignerConfig struct {
	// Address is the sidecar's gRPC address, e.g. "dns:///sidecar-host:9191"
	Address string

	// CACert is the path to the CA certificate
	CACert string

	// ClientCert is the path to the validator's client TLS certificate
	ClientCert string

	// ClientKey is the path to the validator's client TLS private key
	ClientKey string

	// ServerName must match the CN in the sidecar's server certificate
	ServerName string

	// RequestTimeout is the per-RPC deadline.
	// Must be less than CometBFT's vote extension timeout.
	// Default: 2s
	RequestTimeout time.Duration
}

// GRPCRemoteSigner implements RemoteSigner by delegating Sign to the sidecar
// and deriving GetOperatorAddress from the sidecar's public key locally.
type GRPCRemoteSigner struct {
	cfg             GRPCSignerConfig
	conn            *grpc.ClientConn
	client          signerv1.BridgeSignerClient
	operatorAddress string // derived from sidecar's public key at startup, cached
}

// NewGRPCRemoteSigner dials the sidecar, fetches the public key, derives
// the operator address locally, and caches it.
// No private key ever exists on the validator node.
func NewGRPCRemoteSigner(cfg GRPCSignerConfig) (*GRPCRemoteSigner, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("bridge signer address is required")
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 2 * time.Second
	}

	// Build mTLS credentials.
	creds, err := bridgetls.NewClientCredentials(
		cfg.CACert,
		cfg.ClientCert,
		cfg.ClientKey,
		cfg.ServerName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build mTLS credentials: %w", err)
	}

	// Dial the sidecar. grpc.NewClient does not block —
	// the actual TCP connection is established on first use.
	conn, err := grpc.NewClient(
		cfg.Address,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for sidecar at %q: %w", cfg.Address, err)
	}

	s := &GRPCRemoteSigner{
		cfg:    cfg,
		conn:   conn,
		client: signerv1.NewBridgeSignerClient(conn),
	}

	// Fetch the public key from the sidecar at startup.
	// Derive the operator address locally
	// Fails fast if sidecar is unreachable or key is misconfigured.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pubKeyResp, err := s.client.GetPublicKey(ctx, &signerv1.GetPublicKeyRequest{})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get public key from sidecar at %q: %w", cfg.Address, err)
	}

	// Derive bech32 operator address from the public key using the Cosmos SDK.
	// No private key needed — only the compressed public key.
	operatorAddress, err := deriveOperatorAddressFromPubKey(pubKeyResp.PublicKey)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to derive operator address from sidecar public key: %w", err)
	}

	s.operatorAddress = operatorAddress

	return s, nil
}

// Sign implements RemoteSigner.
// Delegates to the sidecar over mTLS-secured gRPC.
func (s *GRPCRemoteSigner) Sign(ctx context.Context, msg []byte) ([]byte, error) {
	if len(msg) != 32 {
		return nil, fmt.Errorf("Sign: msg must be exactly 32 bytes, got %d", len(msg))
	}

	rpcCtx, cancel := context.WithTimeout(ctx, s.cfg.RequestTimeout)
	defer cancel()

	resp, err := s.client.Sign(rpcCtx, &signerv1.SignRequest{Msg: msg})
	if err != nil {
		return nil, fmt.Errorf("Sign RPC failed: %w", err)
	}

	if len(resp.Signature) != 65 {
		return nil, fmt.Errorf("Sign: sidecar returned invalid signature length %d, expected 65", len(resp.Signature))
	}

	return resp.Signature[:64], nil
}

// GetOperatorAddress implements RemoteSigner.
// Returns the cached operator address derived from the sidecar's public key.
// No network call — derived once at startup and cached.
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

// IsReady reports whether the gRPC connection to the sidecar is healthy.
func (s *GRPCRemoteSigner) IsReady() bool {
	state := s.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
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
