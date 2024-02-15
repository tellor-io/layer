package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
)

type Client struct {
	logger     log.Logger
	codec      codec.Codec
	checkpoint string
}

func NewClient(logger log.Logger) *Client {
	return &Client{
		logger: logger,
	}
}

func (c *Client) Start(ctx context.Context, appCodec codec.Codec) {
	c.codec = appCodec
	c.logger.Info("Bridge daemon running")

	// Check and sign bridge message, for deriving and registering EVM address
	err := c.CheckAndSignInitialMessage()
	if err != nil {
		c.logger.Error("Failed to check and sign initial message", "error", err)
		return
	}

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
				sig, err := c.EncodeAndSignMessage(checkpoint)
				if err != nil {
					c.logger.Error("Failed to encode and sign message", "error", err)
					continue
				}
				sigHex := hex.EncodeToString(sig)
				c.logger.Info("Message sig successfully made it to daemon Start func", "signature", sigHex)
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

	body, err := ioutil.ReadAll(resp.Body)
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
