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
	// Since #1046 the vote-extension signer resolves and caches its operator identity
	// once at startup, so the swap plays out in two phases:
	// 4. While the daemon keeps running, the swap is invisible (cached identity) and
	//    the validator must NOT be jailed
	// 5. After a restart the signer derives its identity from the wrong key, emits an
	//    initial registration signature that fails EVM-address recovery in the
	//    proposer's PreBlocker, and the validator must be jailed

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

	// Phase 1: with the daemon still running, the swap must NOT jail. The
	// vote-extension signer cached the true operator identity at startup, so the
	// node keeps claiming its registered identity and never re-emits an initial
	// registration signature (the only automatic jail trigger).
	fmt.Println("\n=== Phase 1: mid-run key swap must not jail ===")
	err = testutil.WaitForBlocks(ctx, 3, validators[0].Node)
	require.NoError(err)

	val0Info, err := chain.StakingQueryValidator(ctx, validators[0].ValAddr)
	require.NoError(err)
	require.False(val0Info.Jailed, "mid-run key swap must not jail: operator identity is cached at signer construction")

	// Phase 2: restart the node so the signer re-derives its identity from the
	// wrong key now named "validator" (the keyring lives on the node's home
	// volume, so the swap survives the restart). The wrong valoper has no
	// registered EVM address, so the daemon emits an initial registration
	// signature that fails EVM-address recovery in the proposer's PreBlocker,
	// which jails validator 0.
	fmt.Println("\n=== Phase 2: restart with wrong key must jail ===")
	err = validators[0].Node.StopContainer(ctx)
	require.NoError(err)
	err = validators[0].Node.StartContainer(ctx)
	require.NoError(err)

	// Use validator 1 as the height source: validator 0 is briefly down while
	// restarting and leaves the active set once jailed. Jailing lands within ~2
	// blocks of validator 0 rejoining consensus.
	err = testutil.WaitForBlocks(ctx, 5, validators[1].Node)
	require.NoError(err)

	val0Info, err = chain.StakingQueryValidator(ctx, validators[0].ValAddr)
	require.NoError(err)
	require.True(val0Info.Jailed, "restart with wrong key must jail via initial-signature mismatch")
}
