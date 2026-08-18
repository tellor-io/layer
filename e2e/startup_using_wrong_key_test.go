package e2e_test

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

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

	queryEVMAddr := func(valAddr string) (string, error) {
		cmd := []string{
			"layerd", "query", "bridge", "get-evm-address-by-validator-address", valAddr,
			"--node", validators[0].Node.Chain.GetRPCAddress(),
			"--home", validators[0].Node.HomeDir(),
			"--chain-id", validators[0].Node.Chain.Config().ChainID,
			"--output", "json",
		}
		stdout, _, execErr := validators[0].Node.Exec(ctx, cmd, nil)
		if execErr != nil {
			return "", execErr
		}
		var resp struct {
			EvmAddress string `json:"evm_address"`
		}
		if jsonErr := json.Unmarshal(stdout, &resp); jsonErr != nil {
			return "", jsonErr
		}
		return resp.EvmAddress, nil
	}

	// Anchor the premise of both phases: validator 0's initial registration must have
	// landed, mapping its operator to the EVM address of the key it was created with.
	originalEVMAddr := crypto.PubkeyToAddress(validators[0].EVMPriv.PublicKey).Hex()
	var registeredEVMAddr string
	err = testutil.WaitForCondition(1*time.Minute, 2*time.Second, func() (bool, error) {
		addr, qErr := queryEVMAddr(validators[0].ValAddr)
		if qErr != nil {
			// not registered yet; WaitForCondition aborts on fn error, so keep polling
			return false, nil
		}
		registeredEVMAddr = addr
		return true, nil
	})
	require.NoError(err, "validator 0's initial EVM address registration never landed")
	// the query returns unprefixed lowercase hex (common.Bytes2Hex)
	require.True(strings.EqualFold(registeredEVMAddr, strings.TrimPrefix(originalEVMAddr, "0x")),
		"registered EVM address %s should match the original key's address %s", registeredEVMAddr, originalEVMAddr)

	// Mid-run keyring swap must not jail (operator identity is cached at signer
	// construction). Restart with the wrong key named "validator" must jail.

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
		"sh", "-c", fmt.Sprintf("layerd keys show validator --bech val -a --keyring-backend test --home %s",
			validators[0].Node.HomeDir()),
	}
	stdout, _, getOriginalKeyErr := validators[0].Node.Exec(ctx, getOriginalKeyCmd, validators[0].Node.Chain.Config().Env)
	require.NoError(getOriginalKeyErr)
	originalValAddr := strings.TrimSpace(string(stdout))
	require.NotEmpty(originalValAddr)
	fmt.Println("original-validator: ", originalValAddr)

	getWrongKeyCmd := []string{
		"sh", "-c", fmt.Sprintf("layerd keys show wrong-validator --bech val -a --keyring-backend test --home %s",
			validators[0].Node.HomeDir()),
	}
	stdout, _, getWrongKeyErr := validators[0].Node.Exec(ctx, getWrongKeyCmd, validators[0].Node.Chain.Config().Env)
	require.NoError(getWrongKeyErr)
	wrongValAddr := strings.TrimSpace(string(stdout))
	require.NotEmpty(wrongValAddr)

	// Step 3: Modify the keyring to make "wrong-validator" the default key for vote extensions
	// We do this by renaming the keys in the keyring
	// First, rename the original "validator" key to "original-validator"
	renameOriginalCmd := []string{
		"sh", "-c", fmt.Sprintf("echo 'y' | layerd keys rename validator original-validator --keyring-backend test --home %s",
			validators[0].Node.HomeDir()),
	}
	_, _, renameOrigErr := validators[0].Node.Exec(ctx, renameOriginalCmd, validators[0].Node.Chain.Config().Env)
	require.NoError(renameOrigErr)

	// Then rename "wrong-validator" to "validator" (this is what the daemon will use)
	renameWrongCmd := []string{
		"sh", "-c", fmt.Sprintf("echo 'y' | layerd keys rename wrong-validator validator --keyring-backend test --home %s",
			validators[0].Node.HomeDir()),
	}

	_, _, renameErr := validators[0].Node.Exec(ctx, renameWrongCmd, validators[0].Node.Chain.Config().Env)
	require.NoError(renameErr)

	getNewKeyCmd := []string{
		"sh", "-c", fmt.Sprintf("layerd keys show validator --bech val -a --keyring-backend test --home %s",
			validators[0].Node.HomeDir()),
	}
	stdout, _, getNewKeyErr := validators[0].Node.Exec(ctx, getNewKeyCmd, validators[0].Node.Chain.Config().Env)
	require.NoError(getNewKeyErr)
	swappedValAddr := strings.TrimSpace(string(stdout))
	fmt.Println("new-validator: ", swappedValAddr)

	// Both phases assume the swap took effect: "validator" must now resolve to the
	// wrong key, not the original.
	require.Equal(wrongValAddr, swappedValAddr, "keyring swap did not take effect: 'validator' should resolve to the wrong key")
	require.NotEqual(originalValAddr, swappedValAddr, "keyring swap did not take effect: 'validator' still resolves to the original key")

	fmt.Println("validator keyring now names the wrong key 'validator'")

	fmt.Println("\n=== Phase 1: mid-run key swap must not jail ===")
	err = testutil.WaitForBlocks(ctx, 3, validators[0].Node)
	require.NoError(err)

	val0Info, err := chain.StakingQueryValidator(ctx, validators[0].ValAddr)
	require.NoError(err)
	require.False(val0Info.Jailed, "mid-run key swap must not jail: operator identity is cached at signer construction")

	// Positive anchor: the live signer kept the original identity, so the registered
	// EVM address must be unchanged (not merely "not jailed yet").
	midSwapEVMAddr, err := queryEVMAddr(validators[0].ValAddr)
	require.NoError(err)
	require.True(strings.EqualFold(midSwapEVMAddr, strings.TrimPrefix(originalEVMAddr, "0x")),
		"mid-run key swap must not change the registered EVM address: got %s, want %s", midSwapEVMAddr, originalEVMAddr)

	fmt.Println("\n=== Phase 2: restart with wrong key must jail ===")
	err = validators[0].Node.StopContainer(ctx)
	require.NoError(err)
	err = validators[0].Node.StartContainer(ctx)
	require.NoError(err)

	// StartContainer returning does not mean validator 0 has caught up, produced a
	// commit vote carrying its InitialSignature, and been jailed by PreBlocker — poll
	// until the jail lands instead of sleeping a fixed number of blocks. Queries can
	// transiently fail while the node restarts, so treat errors as "not yet".
	err = testutil.WaitForCondition(3*time.Minute, 2*time.Second, func() (bool, error) {
		info, qErr := chain.StakingQueryValidator(ctx, validators[0].ValAddr)
		if qErr != nil {
			return false, nil
		}
		return info.Jailed, nil
	})
	require.NoError(err, "timed out waiting for validator 0 to be jailed after restarting with the wrong key")

	val0Info, err = chain.StakingQueryValidator(ctx, validators[0].ValAddr)
	require.NoError(err)
	require.True(val0Info.Jailed, "restart with wrong key must jail via initial-signature mismatch")
}
