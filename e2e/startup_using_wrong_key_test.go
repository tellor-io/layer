package e2e_test

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestStartupUsingWrongKey(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	config := e2e.DefaultSetupConfig()
	config.ModifyGenesis = []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "20s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.000025000000000000"),
	}

	chain, _, ctx := e2e.SetupChainWithCustomConfig(t, config)

	validatorsInfo, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validatorsInfo)

	// Setup validator info with EVM-specific fields
	type Validators struct {
		e2e.ValidatorInfo
		EVMPriv      *ecdsa.PrivateKey
		EVMAddr      string
		WrongKeyPriv *ecdsa.PrivateKey
		WrongKeyAddr string
	}

	validators := make([]Validators, len(validatorsInfo))
	for i, v := range validatorsInfo {
		validators[i] = Validators{
			ValidatorInfo: v,
		}
	}

	// Wait for vote extensions to register EVM addresses
	waitErr := testutil.WaitForBlocks(ctx, 7, validators[0].Node)
	require.NoError(waitErr)

	// Get the original private keys (these are the "correct" keys that validators were created with)
	for i, v := range validators {
		exportCmd := []string{
			"sh", "-c", "echo y | layerd keys export validator --unarmored-hex --unsafe --keyring-backend test --home " +
				v.Node.HomeDir(),
		}

		stdout, _, exportErr := v.Node.Exec(ctx, exportCmd, v.Node.Chain.Config().Env)
		require.NoError(exportErr)

		// Parse the output to extract the private key
		lines := strings.Split(string(stdout), "\n")
		var privKeyHex string
		for _, line := range lines {
			if len(line) >= 64 && !strings.Contains(line, "WARNING") {
				privKeyHex = line
				break
			}
		}

		require.NotEmpty(privKeyHex, "Failed to extract private key")

		// Create ECDSA private key from the exported key
		privKeyBytes, decodeErr := hex.DecodeString(privKeyHex)
		require.NoError(decodeErr)

		exportedPrivKey, privErr := crypto.ToECDSA(privKeyBytes)
		require.NoError(privErr)

		// Store the original private key
		validators[i].EVMPriv = exportedPrivKey

		fmt.Printf("Validator %d - Original private key loaded\n", i)
	}

	// Now we need to create a scenario where validator 0 has a different key than what it was created with
	// This simulates the real-world scenario where:
	// 1. Validator was created with Key A (stored in consensus key, used for block signing)
	// 2. Validator is running with --key-name "validator" but that key is now Key B (wrong key)
	// 3. Validator can still sign blocks with consensus key (Key A)
	// 4. But vote extensions are signed with Key B (wrong key), causing EVM address mismatch
	// 5. This should trigger the jailing mechanism

	fmt.Println("Setting up validator 0 with mismatched key scenario...")

	// Step 1: Generate a new private key that will be the "wrong" key for vote extensions
	wrongPrivKey, err := crypto.GenerateKey()
	require.NoError(err)
	validators[0].WrongKeyPriv = wrongPrivKey
	validators[0].WrongKeyAddr = crypto.PubkeyToAddress(wrongPrivKey.PublicKey).Hex()

	fmt.Printf("Generated wrong key for validator 0 with EVM address: %s\n", validators[0].WrongKeyAddr)

	// Step 2: Add the wrong key with a different name, then modify the keyring to make it the default
	// This simulates the scenario where the validator was created with one key but uses a different key for vote extensions
	wrongKeyHex := hex.EncodeToString(crypto.FromECDSA(wrongPrivKey))

	// Add the wrong key with the name "wrong-validator"
	addWrongKeyCmd := []string{
		"sh", "-c", fmt.Sprintf("layerd keys import-hex wrong-validator %s --keyring-backend test --home %s",
			wrongKeyHex, validators[0].Node.HomeDir()),
	}

	_, _, addKeyErr := validators[0].Node.Exec(ctx, addWrongKeyCmd, validators[0].Node.Chain.Config().Env)
	require.NoError(addKeyErr)

	getOriginalKeyCmd := []string{
		"sh", "-c", fmt.Sprintf("layerd keys show validator --bech val --keyring-backend test --home %s",
			validators[0].Node.HomeDir()),
	}
	stdout, _, getOriginalKeyErr := validators[0].Node.Exec(ctx, getOriginalKeyCmd, validators[0].Node.Chain.Config().Env)
	require.NoError(getOriginalKeyErr)
	fmt.Println("original-validator: ", string(stdout))

	// Step 3: Modify the keyring to make "wrong-validator" the default key for vote extensions
	// We do this by renaming the keys in the keyring
	// First, rename the original "validator" key to "original-validator"
	renameOriginalCmd := []string{
		"sh", "-c", fmt.Sprintf("echo 'y' | layerd keys rename validator original-validator --keyring-backend test --home %s",
			validators[0].Node.HomeDir()),
	}
	_, _, _ = validators[0].Node.Exec(ctx, renameOriginalCmd, validators[0].Node.Chain.Config().Env)

	// Then rename "wrong-validator" to "validator" (this is what the daemon will use)
	renameWrongCmd := []string{
		"sh", "-c", fmt.Sprintf("echo 'y' | layerd keys rename wrong-validator validator --keyring-backend test --home %s",
			validators[0].Node.HomeDir()),
	}

	_, _, renameErr := validators[0].Node.Exec(ctx, renameWrongCmd, validators[0].Node.Chain.Config().Env)
	require.NoError(renameErr)

	getNewKeyCmd := []string{
		"sh", "-c", fmt.Sprintf("layerd keys show validator --bech val --keyring-backend test --home %s",
			validators[0].Node.HomeDir()),
	}
	stdout, _, getNewKeyErr := validators[0].Node.Exec(ctx, getNewKeyCmd, validators[0].Node.Chain.Config().Env)
	require.NoError(getNewKeyErr)
	fmt.Println("new-validator: ", string(stdout))

	fmt.Println("✅ Validator 0 now uses wrong key for vote extensions")
	fmt.Println("✅ Original key is preserved as 'original-validator' in the keyring")
	fmt.Println("✅ Wrong key is now named 'validator' (what the daemon uses for vote extensions)")
	fmt.Println("✅ This simulates the real-world scenario where validator uses wrong key for vote extensions")

	// Verify that the validator can still sign blocks normally
	fmt.Println("\n=== Verifying validator can still sign blocks ===")

	// Wait for a few blocks to see if validator 0 can still participate in consensus
	initialHeight, err := chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Height before waiting: %d\n", initialHeight)

	// Wait for 3 blocks to see if validator 0 can still sign blocks
	err = testutil.WaitForBlocks(ctx, 3, validators[0].Node)
	require.NoError(err)

	finalHeight, err := chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Height after waiting: %d\n", finalHeight)

	// Check if validator 0 is still bonded (can sign blocks)
	val0Info, err := chain.StakingQueryValidator(ctx, validators[0].ValAddr)
	require.NoError(err)

	require.True(val0Info.Jailed)
}
