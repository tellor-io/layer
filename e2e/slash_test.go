package e2e_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestInactivitySlash tests the inactivity slashing mechanism
func TestInactivitySlash(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	config := e2e.DefaultSetupConfig()
	config.NumValidators = 4
	config.ModifyGenesis = []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.000025000000000000"),
		cosmos.NewGenesisKV("app_state.registry.dataspec.0.report_block_window", "5"),
		cosmos.NewGenesisKV("app_state.slashing.params.signed_blocks_window", "25"),
		cosmos.NewGenesisKV("app_state.slashing.params.downtime_jail_duration", "10s"),
	}
	chain, ic, ctx := e2e.SetupChainWithCustomConfig(t, config)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validators)

	val2 := validators[1].Node
	val4 := validators[3].Node
	val4valAddr := validators[3].ValAddr

	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 4)

	// val 4 goes offline
	height, err := chain.Height(ctx)
	require.NoError(err)
	err = val4.PauseContainer(ctx)
	fmt.Println("paused val4 at height: ", height)
	require.NoError(err)

	fmt.Println("waiting 25 blocks...")
	require.NoError(testutil.WaitForBlocks(ctx, 25, val2))

	// 4 validators, 1 jailed
	fmt.Println("querying validators...")
	valsQueryRes, _, err := e2e.QueryWithTimeout(ctx, val2, "staking", "validators")
	require.NoError(err)
	var validatorsRes e2e.QueryValidatorsResponse
	err = json.Unmarshal(valsQueryRes, &validatorsRes)
	require.NoError(err)
	fmt.Println("validatorsRes: ", validatorsRes)
	require.Equal(len(validatorsRes.Validators), 4)

	// make sure one validator is jailed
	jailedCount := 0
	for _, val := range validatorsRes.Validators {
		if val.Jailed {
			jailedCount++
		}
	}
	require.Equal(1, jailedCount, "expected exactly one validator to be jailed")

	// unpause val4
	err = val4.UnpauseContainer(ctx)
	require.NoError(err)
	height, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Println("unpaused val4 at height: ", height)

	// wait for jail duration to expire (30s + buffer)
	time.Sleep(20 * time.Second)

	// unjail val4
	txHash, err := val4.ExecTx(ctx, "validator", "slashing", "unjail", "--from", val4valAddr, "--keyring-dir", val4.HomeDir(), "--chain-id", val4.Chain.Config().ChainID)
	require.NoError(err)
	fmt.Println("unjailed val4 with tx hash: ", txHash)

	// make sure val4 is unjailed
	valsQueryRes, _, err = e2e.QueryWithTimeout(ctx, val2, "staking", "validators")
	require.NoError(err)
	var validatorsRes2 e2e.QueryValidatorsResponse
	err = json.Unmarshal(valsQueryRes, &validatorsRes2)
	require.NoError(err)
	fmt.Println("validatorsRes: ", validatorsRes2)
	require.Equal(len(validatorsRes2.Validators), 4)

	// make sure no validator is jailed
	jailedCount = 0
	for _, val := range validatorsRes2.Validators {
		if val.Jailed {
			jailedCount++
		}
	}
	require.Equal(0, jailedCount, "expected no validators to be jailed")
}
