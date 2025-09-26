package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// cd e2e
// go test -run TestProfitability --timeout 5m

// 10 validators, each report for the cycle list once and claim rewards
func TestProfitability(t *testing.T) {
	require := require.New(t)

	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	cosmos.SetSDKConfig("tellor")

	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "45s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.000025000000000000"),
		// cosmos.NewGenesisKV("app_state.slashing.params.signed_blocks_window", "2"),
	}

	nv := 10
	nf := 0
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
				GasPrices:      "0.000025000000000000loya",
				GasAdjustment:  1.1,
				TrustingPeriod: "504h",
				NoHostMount:    false,
				Images: []ibc.DockerImage{
					{
						Repository: "layer",
						Version:    "local",
						UIDGID:     "1025:1025",
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

	val1 := chain.Validators[0]

	type Validator struct {
		Addr                string
		ValAddr             string
		Val                 *cosmos.ChainNode
		FreeFloatingBalance string
		StakeAmount         string
	}

	validators := make([]Validator, len(chain.Validators))
	for i, v := range chain.Validators {
		valAddr, err := v.AccountKeyBech32(ctx, "validator")
		require.NoError(err)
		fmt.Println("validator [", i, "] address: ", valAddr)
		val1valAddr, err := v.KeyBech32(ctx, "validator", "val")
		require.NoError(err)
		fmt.Println("validator [", i, "] val address: ", val1valAddr)
		freeFloatingBalance, err := chain.BankQueryBalance(ctx, valAddr, "loya")
		require.NoError(err)
		fmt.Println("validator [", i, "] free floating balance: ", freeFloatingBalance)
		stakeAmount, err := chain.StakingQueryValidator(ctx, val1valAddr)
		require.NoError(err)
		fmt.Println("validator [", i, "] stake amount: ", stakeAmount.Tokens)
		validators[i] = Validator{
			Addr:                valAddr,
			ValAddr:             val1valAddr,
			Val:                 v,
			FreeFloatingBalance: freeFloatingBalance.String(),
			StakeAmount:         stakeAmount.Tokens.String(),
		}
	}

	// queryValidators to confirm that 10 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 10)
	height, err := chain.Height(ctx)
	require.NoError(err)
	fmt.Println("current height: ", height)

	// turn on minting
	prop := e2e.Proposal{
		Messages: []map[string]interface{}{
			{
				"@type":     "/layer.mint.MsgInit",
				"authority": "tellor10d07y265gmmuvt4z0w9aw880jnsr700j6527vx",
			},
		},
		Metadata:  "ipfs://CID",
		Deposit:   "50000000loya",
		Title:     "Init tbr minting",
		Summary:   "Initialize inflationary rewards",
		Expedited: false,
	}
	_, err = e2e.ExecProposal(ctx, "validator", prop, val1)
	require.NoError(err)

	time1 := time.Now()
	fmt.Println("time1: ", time1.Format(time.RFC3339))
	// wait 5 blocks
	require.NoError(testutil.WaitForBlocks(ctx, 5, val1))
	time2 := time.Now()
	fmt.Println("time2 after 5 blocks: ", time2.Format(time.RFC3339))
	fmt.Println("time taken: ", time2.Sub(time1))

	for _, v := range chain.Validators {
		_, err = v.ExecTx(ctx, "validator", "gov", "vote", "1", "yes", "--gas", "1000000", "--fees", "25loya", "--keyring-dir", "/var/cosmos-chain/layer-1")
		if err != nil {
			fmt.Println("error voting on proposal: ", err)
		} else {
			fmt.Println("voted on proposal")
		}
	}

	require.NoError(testutil.WaitForBlocks(ctx, 5, val1))
	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	fmt.Println("proposal result: ", result)
	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")
	height, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Println("minting is now on at height ", height)

	// turn one validator into a reporter
	_, err = val1.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", "val1_moniker", "--fees", "25loya", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("validator [0] becomes a reporter")

	// report for the cycle list
	currentCycleListRes, _, err := val1.ExecQuery(ctx, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)
	fmt.Println("current cycle list: ", currentCycleList)
	value := layerutil.EncodeValue(123456789.99)
	_, _, err = val1.Exec(ctx, val1.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "25loya", "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	height, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Println("validator [0] reported at height ", height)

	// turn other 9 validators into reporters
	for i, v := range validators {
		if i == 0 {
			continue
		}
		moniker := fmt.Sprintf("val%d_moniker", i)
		_, err = v.Val.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", moniker, "--fees", "25loya", "--keyring-dir", v.Val.HomeDir())
		require.NoError(err)
		fmt.Println("validator [", i, "] becomes a reporter")
	}

	// everybody reports for the cycle list
	for i, v := range validators {
		currentCycleListRes, _, err := v.Val.ExecQuery(ctx, "oracle", "current-cyclelist-query")
		require.NoError(err)
		var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
		err = json.Unmarshal(currentCycleListRes, &currentCycleList)
		require.NoError(err)
		fmt.Println("current cycle list: ", currentCycleList)

		// report for the cycle list
		_, _, err = v.Val.Exec(ctx, v.Val.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "25loya", "--keyring-dir", v.Val.HomeDir()), v.Val.Chain.Config().Env)
		require.NoError(err)
		height, err = chain.Height(ctx)
		require.NoError(err)
		fmt.Println("validator [", i, "] reported at height ", height)
	}

	// check val1 stake before claim:
	dels, err := chain.StakingQueryDelegations(ctx, validators[1].Addr)
	require.NoError(err)
	fmt.Println("VAL1 delegations before claim: ", dels)

	// check val1 free floating balance before claim:
	freeFloatingBalance, err := chain.BankQueryBalance(ctx, validators[1].Addr, "loya")
	require.NoError(err)
	fmt.Println("VAL1 free floating balance before claim: ", freeFloatingBalance)

	// check val0 stake before claim:
	del, err := chain.StakingQueryDelegations(ctx, validators[0].Addr)
	require.NoError(err)
	fmt.Println("VAL0 delegations before claim: ", del)

	// check val0 free floating balance before claim:
	freeFloatingBalance, err = chain.BankQueryBalance(ctx, validators[0].Addr, "loya")
	require.NoError(err)
	fmt.Println("VAL0 free floating balance before claim: ", freeFloatingBalance)

	// claim validator rewards
	txHash, err := validators[0].Val.ExecTx(ctx, "validator", "distribution", "withdraw-all-rewards", "--keyring-dir", validators[0].Val.HomeDir(), "--from", validators[1].Addr, "--fees", "2222loya")
	require.NoError(err)
	fmt.Println("TX HASH (val1 pays for val0 to claim val0 rewards): ", txHash)

	// check val1 stake after claim:
	dels, err = chain.StakingQueryDelegations(ctx, validators[1].Addr)
	require.NoError(err)
	fmt.Println("VAL1 delegations after claim: ", dels)

	// check val1 free floating balance after claim:
	freeFloatingBalance, err = chain.BankQueryBalance(ctx, validators[1].Addr, "loya")
	require.NoError(err)
	fmt.Println("VAL1 free floating balance after claim: ", freeFloatingBalance)

	// check val0 stake after claim:
	del, err = chain.StakingQueryDelegations(ctx, validators[0].Addr)
	require.NoError(err)
	fmt.Println("VAL0 delegations after claim: ", del)

	// check val0 free floating balance after claim:
	freeFloatingBalance, err = chain.BankQueryBalance(ctx, validators[0].Addr, "loya")
	require.NoError(err)
	fmt.Println("VAL0 free floating balance after claim: ", freeFloatingBalance)

	require.NoError(testutil.WaitForBlocks(ctx, 3, val1))

	// check on reports per reporter
	for _, v := range validators {
		reports, _, err := v.Val.ExecQuery(ctx, "oracle", "get-reportsby-reporter", v.Addr, "--page-limit", "1")
		require.NoError(err)
		var reportsRes e2e.QueryMicroReportsResponse
		err = json.Unmarshal(reports, &reportsRes)
		require.NoError(err)
		fmt.Println("reports from: ", v.Addr, ": ", reportsRes)
	}

	// check on each reporters free floating balance and stake amount
	for _, v := range validators {
		freeFloatingBalance, err := chain.BankQueryBalance(ctx, v.Addr, "loya")
		require.NoError(err)
		fmt.Println("validator [", v.Addr, "] free floating balance: ", freeFloatingBalance)
		stakeAmount, err := chain.StakingQueryValidator(ctx, v.ValAddr)
		require.NoError(err)
		fmt.Println("validator [", v.Addr, "] stake amount: ", stakeAmount.Tokens)
	}

	// claim reporting rewards for each val/reporter
	for _, v := range validators {
		_, err = v.Val.ExecTx(ctx, "validator", "reporter", "withdraw-tip", v.Addr, v.ValAddr, "--fees", "25loya", "--keyring-dir", v.Val.HomeDir())
		fmt.Println("error from claiming (some ppl didnt get their report in): ", err)
		fmt.Println("validator [", v.Addr, "] claimed rewards")
	}

	// check on each val/reporter's free floating balance and stake amount
	for _, v := range validators {
		height, err = chain.Height(ctx)
		require.NoError(err)
		fmt.Println("height: ", height)
		freeFloatingBalance, err := chain.BankQueryBalance(ctx, v.Addr, "loya")
		require.NoError(err)
		fmt.Println("validator [", v.Addr, "] free floating balance: ", freeFloatingBalance)
		stakeAmount, err := chain.StakingQueryValidator(ctx, v.ValAddr)
		require.NoError(err)
		fmt.Println("validator [", v.Addr, "] stake amount: ", stakeAmount.Tokens)
	}
}
