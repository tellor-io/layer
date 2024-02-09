package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
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
