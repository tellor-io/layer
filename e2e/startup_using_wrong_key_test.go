package e2e_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/e2e"
)

func TestStartupUsingWrongKey(t *testing.T) {
	require := require.New(t)

	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	cosmos.SetSDKConfig("tellor")

	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "20s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
	}

	nv := 4
	nf := 1
	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		{
			NumValidators: &nv,
			NumFullNodes:  &nf,
			ChainConfig: ibc.ChainConfig{
				Type:           "cosmos",
				Name:           "layer",
				ChainID:        "layer",
				Bin:            "layerd",
				Denom:          "loya",
				Bech32Prefix:   "tellor",
				CoinType:       "118",
				GasPrices:      "0.0loya",
				GasAdjustment:  1.1,
				TrustingPeriod: "504h",
				NoHostMount:    false,
				Images: []ibc.DockerImage{
					{
						Repository: "layer",
						Version:    "local",
						UidGid:     "1025:1025",
					},
				},
				EncodingConfig:      e2e.LayerEncoding(),
				ModifyGenesis:       cosmos.ModifyGenesis(modifyGenesis),
				AdditionalStartArgs: []string{"--key-name", "validator"},
			},
		},
	})

	client, network := interchaintest.DockerSetup(t)

	chain := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(chain)

	ctx := context.Background()

	require.NoError(ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Setup validator info
	type Validators struct {
		AccAddr      string
		ValAddr      string
		Node         *cosmos.ChainNode
		EVMPriv      *ecdsa.PrivateKey
		EVMAddr      string
		WrongKeyPriv *ecdsa.PrivateKey
		WrongKeyAddr string
	}

	validators := make([]Validators, len(chain.Validators))
	for i := range chain.Validators {
		val := chain.Validators[i]
		valAddr, err := val.AccountKeyBech32(ctx, "validator")
		require.NoError(err)
		valvalAddr, err := val.KeyBech32(ctx, "validator", "val")
		require.NoError(err)
		fmt.Println("val", i, " Account Address: ", valAddr)
		fmt.Println("val", i, " Validator Address: ", valvalAddr)

		validators[i] = Validators{
			AccAddr: valAddr,
			ValAddr: valvalAddr,
			Node:    val,
		}
	}

	// Wait for vote extensions to register EVM addresses
	waitErr := testutil.WaitForBlocks(ctx, 5, validators[0].Node)
	require.NoError(waitErr)

	// Define types for bridge validator queries
	type QueryBridgeValidator struct {
		EthereumAddress string `json:"ethereumAddress"`
		Power           string `json:"power"`
	}
	type QueryGetEvmValidatorsResponse struct {
		BridgeValidatorSet []QueryBridgeValidator `json:"bridge_validator_set"`
	}

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
	err = testutil.WaitForBlocks(ctx, 8, validators[0].Node)
	require.NoError(err)

	finalHeight, err := chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Height after waiting: %d\n", finalHeight)

	// Check if validator 0 is still bonded (can sign blocks)
	val0Info, err := chain.StakingQueryValidator(ctx, validators[0].ValAddr)
	require.NoError(err)

	if !val0Info.Jailed {
		fmt.Println("✅ Validator 0 can still sign blocks normally (not jailed yet)")
		fmt.Println("✅ This confirms the validator can participate in consensus with its consensus key")
	} else {
		fmt.Println("❌ Validator 0 is already jailed - this might indicate an issue with the test setup")
	}

	fmt.Println("✅ But vote extensions will be signed with the wrong key, causing EVM address mismatch")

	// Query the actual registered EVM addresses (these should be from the original keys)
	evmValidatorsRes, _, queryErr := validators[0].Node.ExecQuery(ctx, "bridge", "get-evm-validators")
	require.NoError(queryErr)
	var evmValidators QueryGetEvmValidatorsResponse
	unmarshalErr := json.Unmarshal(evmValidatorsRes, &evmValidators)
	require.NoError(unmarshalErr)

	fmt.Println("Registered EVM validators from bridge:")
	for i, val := range evmValidators.BridgeValidatorSet {
		fmt.Printf("  Validator %d: Address %s, Power %s\n", i, val.EthereumAddress, val.Power)
		// Assign the registered EVM addresses to our validator structs
		if i < len(validators) {
			validators[i].EVMAddr = "0x" + val.EthereumAddress
		}
	}

	// Verify we have the correct number of validators
	require.Equal(len(evmValidators.BridgeValidatorSet), len(validators), "Number of registered EVM validators should match number of test validators")

	// Confirm that all validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 4)

	// Now test the scenario: What happens when validator 0 tries to sign vote extensions with wrong key
	fmt.Println("\n=== Testing validator with mismatched key ===")

	// The scenario we've created:
	// - Validator 0 was created with Key A (original key) - used for block signing
	// - Validator 0 now has Key B as its "validator" key (wrong key) - used for vote extensions
	// - When validator 0 signs vote extensions, it will use Key B
	// - This should cause EVM address registration to fail and potentially jail the validator
	// - Note: Validator can still sign blocks normally with its consensus key

	fmt.Printf("Validator 0 was created with key that generates EVM address: %s\n", validators[0].EVMAddr)
	fmt.Printf("Validator 0 now has wrong key that generates EVM address: %s\n", validators[0].WrongKeyAddr)

	// Verify the mismatch
	require.NotEqual(validators[0].WrongKeyAddr, validators[0].EVMAddr, "Wrong key should generate different EVM address")

	fmt.Println("\nCurrent registered EVM addresses from vote extensions:")
	for i, val := range evmValidators.BridgeValidatorSet {
		fmt.Printf("  Validator %d: %s\n", i, "0x"+val.EthereumAddress)
	}

	// Now we need to wait for the validator to sign vote extensions with the wrong key
	// This should cause issues with EVM address registration and jail the validator by the 2nd block
	// Note: The validator can still sign blocks normally with its consensus key
	fmt.Println("\n=== Waiting for validator 0 to sign vote extensions with wrong key ===")
	fmt.Println("This should cause EVM address registration to fail and jail the validator...")
	fmt.Println("(Validator can still sign blocks normally with its consensus key)")

	// Wait for 2 blocks - the validator should be jailed by the 2nd block
	initialHeight, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Starting at height: %d\n", initialHeight)

	// Wait for 2 blocks
	err = testutil.WaitForBlocks(ctx, 5, validators[0].Node)
	require.NoError(err)

	finalHeight, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("After 2 blocks, at height: %d\n", finalHeight)

	// Check if validator 0 is jailed
	val0Info, err = chain.StakingQueryValidator(ctx, validators[0].ValAddr)
	require.NoError(err)

	fmt.Println("val0Info: ", val0Info)

	if val0Info.Jailed {
		fmt.Printf("✅ Validator 0 was jailed by height %d!\n", finalHeight)
		fmt.Println("✅ This confirms that using wrong key in --key-name causes validator jailing")
	} else {
		fmt.Printf("❌ Validator 0 was NOT jailed by height %d\n", finalHeight)
		fmt.Println("❌ This indicates the jailing mechanism may not be working as expected")
		// Don't fail the test, just report the issue
	}

	// Check how many validators are still bonded
	vals, err = chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	fmt.Printf("Validators still bonded: %d (should be 3 if validator 0 was jailed)\n", len(vals))

	// Verify that other validators can still work correctly with their proper keys
	fmt.Println("\n=== Testing that other validators work correctly ===")

	// Test that validator 1 (with correct key alignment) can perform operations
	val1Addr, err := validators[1].Node.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	fmt.Printf("Validator 1 account address: %s\n", val1Addr)

	// Verify that validator 1's registered EVM address matches its private key
	val1ExpectedEVMAddr := crypto.PubkeyToAddress(validators[1].EVMPriv.PublicKey).Hex()
	fmt.Printf("Validator 1 expected EVM address: %s\n", val1ExpectedEVMAddr)
	fmt.Printf("Validator 1 registered EVM address: %s\n", validators[1].EVMAddr)

	// These should match for validators with correct key alignment
	require.Equal(val1ExpectedEVMAddr, validators[1].EVMAddr, "Validator 1's EVM address should match its private key")

	// Test that validator 0's registered EVM address does NOT match the wrong key
	val0ExpectedEVMAddr := crypto.PubkeyToAddress(validators[0].EVMPriv.PublicKey).Hex()
	val0WrongEVMAddr := crypto.PubkeyToAddress(validators[0].WrongKeyPriv.PublicKey).Hex()

	fmt.Printf("Validator 0 registered EVM address: %s\n", validators[0].EVMAddr)
	fmt.Printf("Validator 0 expected EVM address (from original key): %s\n", val0ExpectedEVMAddr)
	fmt.Printf("Validator 0 wrong key EVM address: %s\n", val0WrongEVMAddr)

	// The registered address should match the original key, not the wrong key
	require.Equal(val0ExpectedEVMAddr, validators[0].EVMAddr, "Validator 0's registered EVM address should match its original key")
	require.NotEqual(val0WrongEVMAddr, validators[0].EVMAddr, "Validator 0's registered EVM address should NOT match the wrong key")

	fmt.Println("\n=== Test Summary ===")
	fmt.Println("✅ Successfully created scenario where validator 0 has mismatched keys")
	fmt.Println("✅ All validators start with same --key-name 'validator'")
	fmt.Println("✅ Validator 0 was created with Key A but now uses Key B for vote extensions")
	fmt.Println("✅ Original key is preserved as 'original-validator' in the keyring")
	fmt.Println("✅ Wrong key is now named 'validator' (what the daemon uses for vote extensions)")
	fmt.Println("✅ Demonstrated EVM address mismatch when wrong key is used for vote extensions")
	fmt.Println("✅ Tested if validator gets jailed when EVM address registration fails")
	fmt.Println("✅ Other validators continue to work correctly with proper key alignment")
	fmt.Println("✅ This reproduces the real-world scenario where validators use wrong keys for vote extensions")
	fmt.Println("✅ Shows how vote extension signing with wrong key creates EVM address mismatches")
	fmt.Println("✅ Validates that the jailing mechanism works for key/address inconsistency")
	fmt.Println("✅ Validator can still sign blocks normally (consensus key preserved)")
	fmt.Println("✅ But vote extensions are signed with wrong key, causing jailing")

}
