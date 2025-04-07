package e2e_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// cd e2e
// go test -run TestInactivitySlash --timeout 5m

// start with 4 validators, one of them goes offline and come back
func TestInactivitySlash(t *testing.T) {
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
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
		cosmos.NewGenesisKV("app_state.slashing.params.signed_blocks_window", "4"),
	}

	nv := 4
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
				AdditionalStartArgs: []string{"--key-name", "validator", "--price-daemon-enabled=false"},
			},
		},
	})

	client, network := interchaintest.DockerSetup(t)

	chain := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(chain)

	ctx := context.Background()

	require.NoError(ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	val1 := chain.Validators[0]
	val1Addr, err := val1.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val1valAddr, err := val1.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val1 Account Address: ", val1Addr)
	fmt.Println("val1 Validator Address: ", val1valAddr)
	val2 := chain.Validators[1]
	val2Addr, err := val2.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val2valAddr, err := val2.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val2 Account Address: ", val2Addr)
	fmt.Println("val2 Validator Address: ", val2valAddr)
	val4 := chain.Validators[3]
	val4Addr, err := val4.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val4valAddr, err := val4.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val4 Account Address: ", val4Addr)
	fmt.Println("val4 Validator Address: ", val4valAddr)

	// queryValidators to confirm that 4 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 4)

	// val 4 goes offline
	height, err := chain.Height(ctx)
	require.NoError(err)
	err = val4.PauseContainer(ctx)
	fmt.Println("paused val4 at height: ", height)
	require.NoError(err)

	// wait 5 blocks
	require.NoError(testutil.WaitForBlocks(ctx, 5, val2))

	// 4 validators, 1 jailed
	fmt.Println("querying validators...")
	valsQueryRes, _, err := val2.ExecQuery(ctx, "staking", "validators")
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

	// wait 10 minutes
	time.Sleep(10 * time.Minute)

	// unjail val4
	txHash, err := val4.ExecTx(ctx, "validator", "slashing", "unjail", "--from", val4valAddr, "--keyring-dir", val4.HomeDir(), "--chain-id", val4.Chain.Config().ChainID)
	require.NoError(err)
	fmt.Println("unjailed val4 with tx hash: ", txHash)

	// make sure val4 is unjailed
	valsQueryRes, _, err = val2.ExecQuery(ctx, "staking", "validators")
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

// 2 vals, turn on minting, register new query, add to cycle list, report, dispute, resolve, stop chain, export, change something, import
func TestFork(t *testing.T) {
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
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
		cosmos.NewGenesisKV("app_state.slashing.params.signed_blocks_window", "4"),
	}

	nv := 2
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
				AdditionalStartArgs: []string{"--key-name", "validator", "--price-daemon-enabled=false"},
			},
		},
	})

	client, network := interchaintest.DockerSetup(t)

	chain := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(chain)

	ctx := context.Background()

	require.NoError(ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	val1 := chain.Validators[0]
	val1Addr, err := val1.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val1valAddr, err := val1.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val1 Account Address: ", val1Addr)
	fmt.Println("val1 Validator Address: ", val1valAddr)
	val2 := chain.Validators[1]
	val2Addr, err := val2.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val2valAddr, err := val2.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val2 Account Address: ", val2Addr)
	fmt.Println("val2 Validator Address: ", val2valAddr)

	type Vals struct {
		Addr      string
		ValAddr   string
		Validator *cosmos.ChainNode
	}
	vals := []Vals{
		{val1Addr, val1valAddr, val1},
		{val2Addr, val2valAddr, val2},
	}

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
	height, err := chain.Height(ctx)
	require.NoError(err)
	fmt.Println("minting is now on at height ", height)

	for i, val := range vals {
		moniker := fmt.Sprintf("val_%d_reporter_moniker", i)
		_, err = val.Validator.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", moniker, "--fees", "25loya", "--keyring-dir", val.Validator.HomeDir())
		require.NoError(err)
		fmt.Println("validator [", i, "] becomes a reporter")
	}

	// val1 registers a new query
	queryType := "NFLSuperBowlChampion"
	spec := e2e.DataSpec{
		DocumentHash:      "legit-ipfs-hash!",
		ResponseValueType: "string",
		AggregationMethod: "weighted-mode",
		AbiComponents: []*registrytypes.ABIComponent{
			{
				Name:            "year of game",
				FieldType:       "string",
				NestedComponent: []*registrytypes.ABIComponent{},
			},
		},
		Registrar:         val1Addr,
		QueryType:         queryType,
		ReportBlockWindow: 10,
	}
	specBz, err := json.Marshal(spec)
	fmt.Println("specBz: ", string(specBz))
	require.NoError(err)
	txHash, err := val1.ExecTx(ctx, val1Addr, "registry", "register-spec", queryType, string(specBz), "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (val1 registers a new query): ", txHash)

	// generate querydata
	queryBz, _, err := val1.ExecQuery(ctx, "registry", "generate-querydata", queryType, "[\"2025\"]")
	require.NoError(err)
	var queryData e2e.QueryGenerateQuerydataResponse
	require.NoError(json.Unmarshal(queryBz, &queryData))
	queryDataStr := hex.EncodeToString(queryData.QueryData)
	fmt.Println("queryDataStr: ", queryDataStr)

	// val1 tips the query
	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	_, _, err = val1.Exec(ctx, val1.TxCommand(val1Addr, "oracle", "tip", queryDataStr, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (val1 tips the query): ", txHash)

	// wait 1 block to prevent account sequence mismatch
	require.NoError(testutil.WaitForBlocks(ctx, 1, val1))

	// user0 and user1 report
	value := e2e.EncodeStringValue("Pittsburgh Steelers")
	fmt.Println("value: ", value)
	for i := range vals {
		txHash, err = vals[i].Validator.ExecTx(ctx, vals[i].Addr, "oracle", "submit-value", queryDataStr, value, "--keyring-dir", vals[i].Validator.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		require.NoError(err)
		fmt.Println("TX HASH (", vals[i].Addr, " reports the query): ", txHash)
	}

	// wait for query to expire
	require.NoError(testutil.WaitForBlocks(ctx, 10, val1))

	// verify reports
	type UserReports struct {
		UserReport e2e.QueryMicroReportsResponse
		Timestamp  string
		qId        string
	}
	userReports := make([]UserReports, len(vals))
	for i := range vals {
		var userReport e2e.QueryMicroReportsResponse
		res, _, err := val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", vals[i].Addr)
		require.NoError(err)
		require.NoError(json.Unmarshal(res, &userReport))
		fmt.Println("userReport: ", userReport)
		require.Equal(len(userReport.MicroReports), 1)
		reportedValue := userReport.MicroReports[0].Value
		fmt.Println("reportedValue: ", reportedValue)
		require.Equal(reportedValue, value)
		decodedVal, err := hex.DecodeString(reportedValue)
		require.NoError(err)
		fmt.Println("decodedVal: ", string(decodedVal))
		decodedBytes, err := base64.StdEncoding.DecodeString(userReport.MicroReports[0].QueryID)
		require.NoError(err)
		hexStr := hex.EncodeToString(decodedBytes)
		userReports[i] = UserReports{
			UserReport: userReport,
			qId:        hexStr,
		}
	}

	// verify aggregate
	res, _, err := val1.ExecQuery(ctx, "oracle", "get-current-aggregate-report", userReports[0].qId)
	require.NoError(err)
	var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
	require.NoError(json.Unmarshal(res, &currentAggRes))
	fmt.Println("currentAggRes: ", currentAggRes)
	require.Equal(currentAggRes.Aggregate.AggregatePower, "10000000") // 2 reporters * 5000000 power
	require.Equal(currentAggRes.Aggregate.AggregateValue, value)
	require.Equal(currentAggRes.Aggregate.Flagged, false)

	// val2 disputes val1 report
	txHash, err = val2.ExecTx(ctx, val2Addr, "dispute", "propose-dispute", val1Addr, userReports[0].UserReport.MicroReports[0].MetaId, userReports[0].qId, "warning", "1000000000loya", "false", "--keyring-dir", val2.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (val2 disputes val1 report): ", txHash)

	// verify dispute is open
	disputes, _, err := val1.ExecQuery(ctx, "dispute", "open-disputes")
	require.NoError(err)
	var openDisputes e2e.QueryOpenDisputesResponse
	require.NoError(json.Unmarshal(disputes, &openDisputes))
	fmt.Println("open disputes: ", openDisputes)
	require.Equal(len(openDisputes.OpenDisputes.Ids), 1)

	// Stop all validators first to prevent database locks
	require.NoError(chain.StopAllNodes(ctx))

	// Wait for processes to fully stop
	time.Sleep(5 * time.Second)

	// Try to export using validator 1 since we've properly stopped all processes
	genesis, _, err := val1.Exec(ctx, []string{
		"layerd", "export",
		"--home", val1.HomeDir(),
	}, val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("genesis: ", string(genesis))
	time.Sleep(15 * time.Second)

	// Write the genesis to a temporary file in the current directory
	exportPath := "exported_genesis.json"
	err = os.WriteFile(exportPath, genesis, 0644)
	require.NoError(err)
	fmt.Printf("Exported genesis file written to: %s\n", exportPath)

	// Read and verify the exported state
	exportedState, err := os.ReadFile(exportPath)
	require.NoError(err)
	fmt.Println("Exported state:", string(exportedState))

	// Clean up the temporary file
	// err = os.Remove(exportPath)
	// require.NoError(err)
}
