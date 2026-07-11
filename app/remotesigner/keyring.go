// Package remotesigner provides a cosmos keyring.Keyring backed by the
// bridge-remote-signer over gRPC/mTLS. It lets the validator node sign
// vote-extension (bridge attestation) messages through the remote signer
// instead of a local private key in a file keyring.
//
// The Sign path computes sha256(msg) and calls the signer's SignRaw RPC,
// which is byte-for-byte identical to what a local cosmos secp256k1 keyring
// produces — so vote-extension signatures are unchanged.
package remotesigner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"

	signerv1 "github.com/tellor-io/bridge-remote-signer/api/gen/signer/v1"
	bridgetls "github.com/tellor-io/bridge-remote-signer/api/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmossecp "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// remoteSignerKeyring implements keyring.Keyring backed by a remote gRPC signer.
// Only Key()/List()/Sign() are functional; mutating methods return errors.
type remoteSignerKeyring struct {
	keyName    string
	pubKey     cryptotypes.PubKey
	signerConn signerv1.BridgeSignerClient
}

func newRemoteSignerKeyring(keyName string, pubKeyBytes []byte, signerConn signerv1.BridgeSignerClient) (*remoteSignerKeyring, error) {
	if len(pubKeyBytes) != 33 {
		return nil, fmt.Errorf("remotesigner: expected 33-byte compressed public key, got %d", len(pubKeyBytes))
	}
	return &remoteSignerKeyring{
		keyName:    keyName,
		pubKey:     &cosmossecp.PubKey{Key: pubKeyBytes},
		signerConn: signerConn,
	}, nil
}

func (r *remoteSignerKeyring) Backend() string { return "remote-signer" }

func (r *remoteSignerKeyring) List() ([]*keyring.Record, error) {
	rec, err := keyring.NewOfflineRecord(r.keyName, r.pubKey)
	if err != nil {
		return nil, err
	}
	return []*keyring.Record{rec}, nil
}

func (r *remoteSignerKeyring) SupportedAlgorithms() (keyring.SigningAlgoList, keyring.SigningAlgoList) {
	return keyring.SigningAlgoList{hd.Secp256k1}, keyring.SigningAlgoList{}
}

func (r *remoteSignerKeyring) Key(uid string) (*keyring.Record, error) {
	rec, err := keyring.NewOfflineRecord(uid, r.pubKey)
	if err != nil {
		return nil, fmt.Errorf("remotesigner.Key: %w", err)
	}
	return rec, nil
}

func (r *remoteSignerKeyring) KeyByAddress(address sdk.Address) (*keyring.Record, error) {
	if !sdk.AccAddress(r.pubKey.Address()).Equals(address) {
		return nil, fmt.Errorf("remotesigner.KeyByAddress: address not found")
	}
	return r.Key(r.keyName)
}

func (r *remoteSignerKeyring) Delete(_ string) error {
	return fmt.Errorf("remotesigner: Delete not supported")
}

func (r *remoteSignerKeyring) DeleteByAddress(_ sdk.Address) error {
	return fmt.Errorf("remotesigner: DeleteByAddress not supported")
}

func (r *remoteSignerKeyring) Rename(_, _ string) error {
	return fmt.Errorf("remotesigner: Rename not supported")
}

func (r *remoteSignerKeyring) NewMnemonic(_ string, _ keyring.Language, _, _ string, _ keyring.SignatureAlgo) (*keyring.Record, string, error) {
	return nil, "", fmt.Errorf("remotesigner: NewMnemonic not supported")
}

func (r *remoteSignerKeyring) NewAccount(_, _, _, _ string, _ keyring.SignatureAlgo) (*keyring.Record, error) {
	return nil, fmt.Errorf("remotesigner: NewAccount not supported")
}

func (r *remoteSignerKeyring) SaveLedgerKey(_ string, _ keyring.SignatureAlgo, _ string, _, _, _ uint32) (*keyring.Record, error) {
	return nil, fmt.Errorf("remotesigner: SaveLedgerKey not supported")
}

func (r *remoteSignerKeyring) SaveOfflineKey(_ string, _ cryptotypes.PubKey) (*keyring.Record, error) {
	return nil, fmt.Errorf("remotesigner: SaveOfflineKey not supported")
}

func (r *remoteSignerKeyring) SaveMultisig(_ string, _ cryptotypes.PubKey) (*keyring.Record, error) {
	return nil, fmt.Errorf("remotesigner: SaveMultisig not supported")
}

// Sign computes sha256(msg) and signs it via the remote signer's SignRaw RPC,
// returning a 64-byte (r||s) secp256k1 signature.
func (r *remoteSignerKeyring) Sign(_ string, msg []byte, _ signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
	hash := sha256.Sum256(msg)
	resp, err := r.signerConn.SignRaw(context.Background(), &signerv1.SignRawRequest{
		Msg:       hash[:],
		RequestId: "layer-vote-ext",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("remotesigner.Sign: SignRaw RPC failed: %w", err)
	}
	return resp.Signature, r.pubKey, nil
}

func (r *remoteSignerKeyring) SignByAddress(address sdk.Address, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
	if !sdk.AccAddress(r.pubKey.Address()).Equals(address) {
		return nil, nil, fmt.Errorf("remotesigner.SignByAddress: address mismatch")
	}
	return r.Sign(r.keyName, msg, signMode)
}

func (r *remoteSignerKeyring) ImportPrivKey(_, _, _ string) error {
	return fmt.Errorf("remotesigner: ImportPrivKey not supported")
}

func (r *remoteSignerKeyring) ImportPrivKeyHex(_, _, _ string) error {
	return fmt.Errorf("remotesigner: ImportPrivKeyHex not supported")
}

func (r *remoteSignerKeyring) ImportPubKey(_, _ string) error {
	return fmt.Errorf("remotesigner: ImportPubKey not supported")
}

func (r *remoteSignerKeyring) MigrateAll() ([]*keyring.Record, error) {
	return nil, fmt.Errorf("remotesigner: MigrateAll not supported")
}

func (r *remoteSignerKeyring) ExportPubKeyArmor(_ string) (string, error) {
	return "", fmt.Errorf("remotesigner: ExportPubKeyArmor not supported")
}

func (r *remoteSignerKeyring) ExportPubKeyArmorByAddress(_ sdk.Address) (string, error) {
	return "", fmt.Errorf("remotesigner: ExportPubKeyArmorByAddress not supported")
}

func (r *remoteSignerKeyring) ExportPrivKeyArmor(_, _ string) (string, error) {
	return "", fmt.Errorf("remotesigner: ExportPrivKeyArmor not supported")
}

func (r *remoteSignerKeyring) ExportPrivKeyArmorByAddress(_ sdk.Address, _ string) (string, error) {
	return "", fmt.Errorf("remotesigner: ExportPrivKeyArmorByAddress not supported")
}

// BridgeRemoteSigner is the validated-signing surface of the remote signer.
// The app type-asserts a keyring.Keyring to this interface to decide whether to
// use the structured, fail-closed RPCs (SignBridgeCheckpoint /
// SignOracleAttestation) instead of the blind SignRaw path. Only the remote
// signer keyring implements it; a local file keyring does not, so the blind
// path is used there.
type BridgeRemoteSigner interface {
	SignBridgeCheckpoint(ctx context.Context, req *signerv1.SignBridgeCheckpointRequest) ([]byte, error)
	SignOracleAttestation(ctx context.Context, req *signerv1.SignOracleAttestationRequest) ([]byte, error)
}

// SignBridgeCheckpoint asks the remote signer to recompute the valset checkpoint
// from the structured inputs and sign it. The signer fails closed unless its
// recomputed checkpoint matches req.ExpectedCheckpoint, so a wrong input field
// yields an error here (the node simply does not sign), never a wrong signature.
func (r *remoteSignerKeyring) SignBridgeCheckpoint(ctx context.Context, req *signerv1.SignBridgeCheckpointRequest) ([]byte, error) {
	resp, err := r.signerConn.SignBridgeCheckpoint(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("remotesigner.SignBridgeCheckpoint: RPC failed: %w", err)
	}
	if len(resp.Signature) != 64 {
		return nil, fmt.Errorf("remotesigner.SignBridgeCheckpoint: expected 64-byte signature, got %d", len(resp.Signature))
	}
	if !bytes.Equal(resp.Checkpoint, req.ExpectedCheckpoint) {
		return nil, fmt.Errorf("remotesigner.SignBridgeCheckpoint: signer checkpoint does not match expected checkpoint")
	}
	return resp.Signature, nil
}

// SignOracleAttestation asks the remote signer to recompute the attestation
// snapshot from the structured inputs and sign it. The signer fails closed
// unless its recomputed snapshot matches req.ExpectedSnapshot.
func (r *remoteSignerKeyring) SignOracleAttestation(ctx context.Context, req *signerv1.SignOracleAttestationRequest) ([]byte, error) {
	resp, err := r.signerConn.SignOracleAttestation(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("remotesigner.SignOracleAttestation: RPC failed: %w", err)
	}
	if len(resp.Signature) != 64 {
		return nil, fmt.Errorf("remotesigner.SignOracleAttestation: expected 64-byte signature, got %d", len(resp.Signature))
	}
	if !bytes.Equal(resp.Snapshot, req.ExpectedSnapshot) {
		return nil, fmt.Errorf("remotesigner.SignOracleAttestation: signer snapshot does not match expected snapshot")
	}
	return resp.Signature, nil
}

var (
	_ keyring.Keyring    = (*remoteSignerKeyring)(nil)
	_ BridgeRemoteSigner = (*remoteSignerKeyring)(nil)
)

// NewKeyring dials the remote signer at addr, fetches its public key, and returns
// a keyring.Keyring backed by it. When caCert/clientCert/clientKey are all set,
// mTLS is used; otherwise the connection is insecure (local/test only).
// The underlying gRPC connection stays open for the lifetime of the keyring.
func NewKeyring(ctx context.Context, keyName, addr, caCert, clientCert, clientKey string) (keyring.Keyring, error) {
	var dialOpt grpc.DialOption
	if caCert != "" && clientCert != "" && clientKey != "" {
		creds, err := bridgetls.NewClientCredentials(caCert, clientCert, clientKey, "bridge-signer")
		if err != nil {
			return nil, fmt.Errorf("remotesigner: load mTLS credentials: %w", err)
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	conn, err := grpc.NewClient(addr, dialOpt)
	if err != nil {
		return nil, fmt.Errorf("remotesigner: dial %s: %w", addr, err)
	}

	signerClient := signerv1.NewBridgeSignerClient(conn)
	pubKeyResp, err := signerClient.GetPublicKey(ctx, &signerv1.GetPublicKeyRequest{})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("remotesigner: GetPublicKey: %w", err)
	}

	kr, err := newRemoteSignerKeyring(keyName, pubKeyResp.PublicKey, signerClient)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return kr, nil
}
