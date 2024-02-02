package main

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

func main() {
	// define keyring backend and the path to the keystore dir
	krBackend := keyring.BackendFile
	krDir := os.ExpandEnv("$HOME/.layer")
	fmt.Println("Keyring dir:", krDir)

	// create a new keyring instance
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	kr, err := keyring.New("cosmos", krBackend, krDir, os.Stdin, cdc)
	if err != nil {
		fmt.Printf("Failed to create keyring: %v\n", err)
		return
	}
	fmt.Println("Keyring created")

	krlist, err := kr.List()
	if err != nil {
		fmt.Printf("Failed to list keys: %v\n", err)
		return
	}

	for _, k := range krlist {
		fmt.Println("name: ", k.Name)
	}

	// bob, _, err := kr.NewMnemonic("Bob", keyring.English, sdk.FullFundraiserPath, DefaultBIP39Passphrase, sec)

	// kr.

	// Fetch the operator key from the keyring.
	// Replace "operatorKeyName" with the actual name of your operator key.
	info, err := kr.Key("dan")
	if err != nil {
		fmt.Printf("Failed to get operator key: %v\n", err)
		return
	}
	// Output the public key associated with the operator key.
	key, _ := info.GetPubKey()
	keyAddrStr := key.Address().String()
	fmt.Println("Operator Public Key:", keyAddrStr)
}
