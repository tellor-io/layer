package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cosmossdk.io/log"
	cmttypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	daemontypes "github.com/tellor-io/layer/daemons/types"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

type Client struct {
	logger     log.Logger
	codec      codec.Codec
	checkpoint string
	cosmosCtx  client.Context
}

func NewClient(clientCtx client.Context, logger log.Logger) *Client {
	return &Client{
		cosmosCtx: clientCtx,
		logger:    logger,
	}
}

func (c *Client) Start(ctx context.Context, appCodec codec.Codec, grpcClient daemontypes.GrpcClient) {
	c.codec = appCodec
	c.logger.Info("Bridge daemon running")

	// Check and sign bridge message, for deriving and registering EVM address
	// err := c.CheckAndSignInitialMessage()
	// if err != nil {
	// 	c.logger.Error("Failed to check and sign initial message", "error", err)
	// 	return
	// }

	ticker := time.NewTicker(30 * time.Second) // Adjust the duration according to your needs
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkpoint, err := c.QueryLatestCheckpoint()
			if err != nil {
				c.logger.Error("Failed to query latest checkpoint", "error", err)
				continue
			}
			if c.isNewCheckpoint(checkpoint) {
				c.checkpoint = checkpoint
				c.logger.Info("New checkpoint", "checkpoint", checkpoint)
				// this is for initial eth address registration, update this to wait for node to be ready, then submit signature one time
				sig, err := c.EncodeAndSignMessage(checkpoint)
				if err != nil {
					c.logger.Error("Failed to encode and sign message", "error", err)
					continue
				}
				sigHex := hex.EncodeToString(sig)
				timestamp, err := c.QueryValsetTimestamp()
				if err != nil {
					c.logger.Error("Failed to query valset timestamp", "error", err)
					continue
				}
				c.logger.Info("Message sig successfully made it to daemon Start func", "signature", sigHex)
				err = c.SubmitSignature(context.Background(), sigHex, timestamp)
				if err != nil {
					c.logger.Error("Failed to submit signature to bridge module", "error", err)
					continue
				}
			}

			error := c.CheckAndSignInitialMessage()
			if error != nil {
				c.logger.Error("Failed to check and sign initial message", "error", error)
				return
			}
		}
	}
}

func (c *Client) QueryLatestCheckpoint() (string, error) {
	resp, err := http.Get("http://localhost:1317/tellor-io/layer/bridge/get_validator_checkpoint")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Checkpoint string `json:"validatorCheckpoint"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Checkpoint, nil
}

func (c *Client) QueryValsetTimestamp() (string, error) {
	resp, err := http.Get("http://localhost:1317/tellor-io/layer/bridge/get_validator_timestamp_by_index?index=0")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Timestamp, nil
}

func (c *Client) isNewCheckpoint(checkpoint string) bool {
	return checkpoint != c.checkpoint
}

func (c *Client) EncodeAndSignMessage(checkpointString string) ([]byte, error) {
	// Encode the checkpoint string to bytes
	checkpoint, err := hex.DecodeString(checkpointString)
	if err != nil {
		c.logger.Error("Failed to decode checkpoint", "error", err)
		return nil, err
	}
	signature, err := c.SignMessage(checkpoint)
	if err != nil {
		c.logger.Error("Failed to sign message", "error", err)
		return nil, err
	}
	return signature, nil
}

// CheckAndSignInitialMessage checks for the existence of "bridgeSig.txt".
// If it doesn't exist, it signs a predefined message, creates the file, and writes the signature.
func (c *Client) CheckAndSignInitialMessage() error {
	c.logger.Info("Checking for initial signature file")
	// Resolve the home directory.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		c.logger.Error("Failed to get user home directory", "error", err)
		return err
	}

	// Construct the full path for the file.
	filePath := filepath.Join(homeDir, ".layer", "bridgeSig.txt")
	// Check if "bridgeSig.txt" exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File does not exist, proceed to sign a message
		message := "TellorLayer: Initial bridge daemon signature"
		// convert message to bytes
		msgBytes := []byte(message)
		// hash message
		msgHashBytes32 := sha256.Sum256(msgBytes)
		// convert [32]byte to []byte
		msgHashBytes := msgHashBytes32[:]
		// sign message
		sig, err := c.SignMessage(msgHashBytes)
		if err != nil {
			c.logger.Error("Failed to sign message", "error", err)
			return err
		}
		// append 00 to the end of the signature
		sig = append(sig, 0)
		sigHex := hex.EncodeToString(sig)

		c.logger.Info("Submitting pubkey to bridge module via transaction")
		err = c.SubmitPubkey(context.Background(), sigHex)
		if err != nil {
			c.logger.Error("Failed to submit pubkey to bridge module", "error", err)
			return err
		}

		// Write the signature to "bridgeSig.txt"
		err = os.WriteFile(filePath, []byte(sigHex), 0644)
		if err != nil {
			c.logger.Error("Failed to write signature file", "error", err, "path", filePath)
			return err
		}
		c.logger.Info("Signature file created", "signature", sigHex, "path", filePath)

	} else if err != nil {
		// An error occurred checking the file, not related to the file not existing
		c.logger.Error("Failed to check signature file", "error", err, "path", filePath)
		return err
	} else {
		c.logger.Info("Signature file already exists", "path", filePath)
	}
	return nil
}

func (c *Client) SubmitPubkey(ctx context.Context, signature string) error {
	// Submit the signature to the bridge module using "SubmitReport" below as a guide
	accountName := "alice"
	c.cosmosCtx = c.cosmosCtx.WithChainID("layer")
	fromAddr, fromName, _, err := client.GetFromFields(c.cosmosCtx, c.cosmosCtx.Keyring, accountName)
	if err != nil {
		return fmt.Errorf("error getting address from keyring: %v", err)
	}
	c.cosmosCtx = c.cosmosCtx.WithFrom(accountName).WithFromAddress(fromAddr).WithFromName(fromName)
	msgSubmitSig := &bridgetypes.MsgRegisterOperatorPubkey{
		Creator:        fromAddr.String(),
		OperatorPubkey: signature,
	}
	_, seq, err := c.cosmosCtx.AccountRetriever.GetAccountNumberSequence(c.cosmosCtx, fromAddr)
	if err != nil {
		return fmt.Errorf("error getting account number sequence for 'MsgSubmitBridgeValsetSignature': %v", err)
	}

	txf := tx.Factory{}.
		WithChainID(c.cosmosCtx.ChainID).
		WithKeybase(c.cosmosCtx.Keyring).
		WithGasAdjustment(1.1).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithAccountRetriever(c.cosmosCtx.AccountRetriever).
		WithTxConfig(c.cosmosCtx.TxConfig)
	// txf := newFactory(c.cosmosCtx)
	txf = txf.WithSequence(seq)
	txf, err = txf.Prepare(c.cosmosCtx)
	if err != nil {
		return fmt.Errorf("error preparing transaction: %v", err)
	}
	gas := uint64(200000)
	txf = txf.WithGas(gas)

	txn, err := txf.BuildUnsignedTx(msgSubmitSig)
	if err != nil {
		return fmt.Errorf("error building 'MsgSubmitBridgeValsetSignature' unsigned transaction: %v", err)

	}
	if err = tx.Sign(c.cosmosCtx.CmdContext, txf, c.cosmosCtx.FromName, txn, true); err != nil {
		return fmt.Errorf("error when signing 'MsgSubmitBridgeValsetSignature' transaction: %v", err)
	}

	txBytes, err := c.cosmosCtx.TxConfig.TxEncoder()(txn.GetTx())
	if err != nil {
		return fmt.Errorf("error encoding 'MsgSubmitBridgeValsetSignature' transaction: %v", err)
	}
	res, err := c.cosmosCtx.BroadcastTx(txBytes)
	if err := handleBroadcastResult(res, err); err != nil {
		return fmt.Errorf("error broadcasting 'MsgSubmitBridgeValsetSignature' transaction after 'handleBroadcastResult': %v", err)
	}
	txnResult, err := c.WaitForTx(ctx, res.TxHash)
	if err != nil {
		return fmt.Errorf("error waiting for 'MsgSubmitBridgeValsetSignature' transaction: %v", err)
	}
	c.logger.Info("SubmitSignatureTxResult", "TxResult", txnResult)
	return nil
}

func (c *Client) SubmitSignature(ctx context.Context, signature string, timestamp string) error {
	// Submit the signature to the bridge module using "SubmitReport" below as a guide
	accountName := "alice"
	c.cosmosCtx = c.cosmosCtx.WithChainID("layer")
	fromAddr, fromName, _, err := client.GetFromFields(c.cosmosCtx, c.cosmosCtx.Keyring, accountName)
	if err != nil {
		return fmt.Errorf("error getting address from keyring: %v", err)
	}
	c.cosmosCtx = c.cosmosCtx.WithFrom(accountName).WithFromAddress(fromAddr).WithFromName(fromName)
	msgSubmitSig := &bridgetypes.MsgSubmitBridgeValsetSignature{
		Creator:   fromAddr.String(),
		Signature: signature,
		Timestamp: timestamp,
	}
	_, seq, err := c.cosmosCtx.AccountRetriever.GetAccountNumberSequence(c.cosmosCtx, fromAddr)
	if err != nil {
		return fmt.Errorf("error getting account number sequence for 'MsgSubmitBridgeValsetSignature': %v", err)
	}

	txf := tx.Factory{}.
		WithChainID(c.cosmosCtx.ChainID).
		WithKeybase(c.cosmosCtx.Keyring).
		WithGasAdjustment(1.1).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithAccountRetriever(c.cosmosCtx.AccountRetriever).
		WithTxConfig(c.cosmosCtx.TxConfig)
	// txf := newFactory(c.cosmosCtx)
	txf = txf.WithSequence(seq)
	txf, err = txf.Prepare(c.cosmosCtx)
	if err != nil {
		return fmt.Errorf("error preparing transaction: %v", err)
	}
	gas := uint64(100000)
	txf = txf.WithGas(gas)

	txn, err := txf.BuildUnsignedTx(msgSubmitSig)
	if err != nil {
		return fmt.Errorf("error building 'MsgSubmitBridgeValsetSignature' unsigned transaction: %v", err)

	}
	if err = tx.Sign(c.cosmosCtx.CmdContext, txf, c.cosmosCtx.FromName, txn, true); err != nil {
		return fmt.Errorf("error when signing 'MsgSubmitBridgeValsetSignature' transaction: %v", err)
	}

	txBytes, err := c.cosmosCtx.TxConfig.TxEncoder()(txn.GetTx())
	if err != nil {
		return fmt.Errorf("error encoding 'MsgSubmitBridgeValsetSignature' transaction: %v", err)
	}
	res, err := c.cosmosCtx.BroadcastTx(txBytes)
	if err := handleBroadcastResult(res, err); err != nil {
		return fmt.Errorf("error broadcasting 'MsgSubmitBridgeValsetSignature' transaction after 'handleBroadcastResult': %v", err)
	}
	txnResult, err := c.WaitForTx(ctx, res.TxHash)
	if err != nil {
		return fmt.Errorf("error waiting for 'MsgSubmitBridgeValsetSignature' transaction: %v", err)
	}
	c.logger.Info("SubmitSignatureTxResult", "TxResult", txnResult)
	return nil
}

func handleBroadcastResult(resp *sdk.TxResponse, err error) error {
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("make sure that your account has enough balance")
		}
		return err
	}

	if resp.Code > 0 {
		return fmt.Errorf("error code: '%d' msg: '%s'", resp.Code, resp.RawLog)
	}
	return nil
}

func (c *Client) WaitForTx(ctx context.Context, hash string) (*cmttypes.ResultTx, error) {
	bz, err := hex.DecodeString(hash)
	if err != nil {
		return nil, fmt.Errorf("unable to decode tx hash '%s'; err: %v", hash, err)
	}
	for {
		resp, err := c.cosmosCtx.Client.Tx(ctx, bz, false)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				// Tx not found, wait for next block and try again
				err := c.WaitForNextBlock(ctx)
				if err != nil {
					return nil, fmt.Errorf("waiting for next block: err: %v", err)
				}
				continue
			}
			return nil, fmt.Errorf("fetching tx '%s'; err: %v", hash, err)
		}
		// Tx found
		return resp, nil
	}
}

func (c Client) WaitForNextBlock(ctx context.Context) error {
	return c.WaitForNBlocks(ctx, 1)
}
func (c Client) WaitForNBlocks(ctx context.Context, n int64) error {
	start, err := c.LatestBlockHeight(ctx)
	if err != nil {
		return err
	}
	return c.WaitForBlockHeight(ctx, start+n)
}
func (c Client) LatestBlockHeight(ctx context.Context) (int64, error) {
	resp, err := c.Status(ctx)
	if err != nil {
		return 0, err
	}
	return resp.SyncInfo.LatestBlockHeight, nil
}

func (c Client) Status(ctx context.Context) (*cmttypes.ResultStatus, error) {
	return c.cosmosCtx.Client.Status(ctx)
}
func (c Client) WaitForBlockHeight(ctx context.Context, h int64) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		latestHeight, err := c.LatestBlockHeight(ctx)
		if err != nil {
			return err
		}
		if latestHeight >= h {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout exceeded waiting for block, err: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}
