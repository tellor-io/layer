package client

import (
	"fmt"
	"os"

	"cosmossdk.io/log"
	"github.com/cometbft/cometbft/libs/bytes"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/tellor-io/layer/app"
)

func main() {
	tempApp := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(tempDir()))

	// define keyring backend and the path to the keystore dir
	krBackend := keyring.BackendTest
	krDir := os.ExpandEnv("$HOME/.layer")
	fmt.Println("Keyring dir:", krDir)

	kr, err := keyring.New("layer", krBackend, krDir, os.Stdin, tempApp.AppCodec())
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

	// Fetch the operator key from the keyring.
	info, err := kr.Key("frank")
	if err != nil {
		fmt.Printf("Failed to get operator key: %v\n", err)
		return
	}
	// Output the public key associated with the operator key.
	key, _ := info.GetPubKey()
	keyAddrStr := key.Address().String()
	fmt.Println("Operator Public Key:", keyAddrStr)

	// sign message
	tempmsg := []byte("hello")
	sig, pubKeyReturned, err := kr.Sign("frank", tempmsg, 1)
	if err != nil {
		fmt.Printf("Failed to sign message: %v\n", err)
		return
	}
	fmt.Println("Signature:", bytes.HexBytes(sig).String())
	fmt.Println("Public Key:", pubKeyReturned.Address().String())
}

func tempDir() string {
	dir, err := os.MkdirTemp("", "layer")
	if err != nil {
		fmt.Printf("Failed to create temp dir: %v", err)
	}
	return dir
}
