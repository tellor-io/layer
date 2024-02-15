package client

import (
	"fmt"
	"os"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

func (c *Client) SignMessage(msg []byte) ([]byte, error) {
	// define keyring backend and the path to the keystore dir
	krBackend := keyring.BackendTest
	krDir := os.ExpandEnv("$HOME/.layer")
	c.logger.Info("Keyring dir:", "dir", krDir)

	kr, err := keyring.New("layer", krBackend, krDir, os.Stdin, c.codec)
	if err != nil {
		fmt.Printf("Failed to create keyring: %v\n", err)
		return nil, err
	}
	fmt.Println("Keyring created")

	krlist, err := kr.List()
	if err != nil {
		fmt.Printf("Failed to list keys: %v\n", err)
		return nil, err
	}

	for _, k := range krlist {
		fmt.Println("name: ", k.Name)
	}

	// Fetch the operator key from the keyring.
	info, err := kr.Key("alice")
	if err != nil {
		fmt.Printf("Failed to get operator key: %v\n", err)
		return nil, err
	}
	// Output the public key associated with the operator key.
	key, _ := info.GetPubKey()
	keyAddrStr := key.Address().String()
	fmt.Println("Operator Public Key:", keyAddrStr)

	// sign message
	// tempmsg := []byte("hello")
	sig, pubKeyReturned, err := kr.Sign("alice", msg, 1)
	if err != nil {
		fmt.Printf("Failed to sign message: %v\n", err)
		return nil, err
	}
	c.logger.Info("Signature:", "sig", bytes.HexBytes(sig).String())
	c.logger.Info("Public Key:", pubKeyReturned.Address().String())
	return sig, nil
}
