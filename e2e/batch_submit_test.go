package e2e_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	"github.com/tellor-io/layer/utils"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestBatchSubmitValue(t *testing.T) {
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

	// Get validators
	validators := getValidators(t, ctx, chain)
	val1 := validators[0].Val
	val2 := validators[1].Val
	val3 := validators[2].Val
	val4 := validators[3].Val
	// val1Addr := validators[0].Addr

	// Create a reporter from validator 1
	fmt.Println("Creating reporter from validator 1...")
	txHash, err := val1.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", "reporter1", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("Reporter creation tx hash:", txHash)

	// Cyclist queries
	ethQueryDataStr := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	ethQueryData, err := hex.DecodeString(ethQueryDataStr)
	require.NoError(err)
	btcQueryDataStr := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	btcQueryData, err := hex.DecodeString(btcQueryDataStr)
	require.NoError(err)
	trbQueryDataStr := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	trbQueryData, err := hex.DecodeString(trbQueryDataStr)
	require.NoError(err)
	// convert to base64 for CLI
	queryData1 := base64.StdEncoding.EncodeToString(ethQueryData)
	queryData2 := base64.StdEncoding.EncodeToString(btcQueryData)
	queryData3 := base64.StdEncoding.EncodeToString(trbQueryData)

	fmt.Printf("Cycle list queries:\n1: %s\n2: %s\n3: %s\n", queryData1, queryData2, queryData3)

	// Random value
	value := "000000000000000000000000000000000000000000000000000000000000001e" // hex encoded value (30)

	// Create strings for CLI input
	value1 := fmt.Sprintf(`{"query_data":"%s", "value":"%s"}`, queryData1, value)
	value2 := fmt.Sprintf(`{"query_data":"%s", "value":"%s"}`, queryData2, value)
	value3 := fmt.Sprintf(`{"query_data":"%s", "value":"%s"}`, queryData3, value)

	// Try to submit values for all three queries initially
	// Without tips, only one of them should succeed (since without tips only one query can be in the cycle list at a time)
	fmt.Println("\n=== Testing initial submission (expecting 2 failures, 1 success) ===")
	// Execute batch submit
	txHash1, err := val1.ExecTx(
		ctx, "validator",
		"oracle",
		"batch-submit-value",
		"--values", value1,
		"--values", value2,
		"--values", value3,
		"--fees", "25loya",
		"--keyring-dir",
		val1.HomeDir(),
	)
	require.NoError(err)

	// Query the transaction result to see which ones failed
	txRes, _, err := val1.ExecQuery(ctx, "tx", txHash1)
	require.NoError(err)
	fmt.Println("Transaction result for first submission:", string(txRes))

	// ======================================================================================
	// Now tip all three queries to make them submittable
	fmt.Println("\n=== Tipping all three cycle list queries ===")
	tipAmount := sdk.NewCoin("loya", math.NewInt(1000000)) // 1 TRB

	// Tip query 1
	// Use Exec so it doesn't wait 2 Blocks
	cmd := val2.TxCommand("validator", "oracle", "tip", queryData1, tipAmount.String(), "--keyring-dir", val2.HomeDir())
	stdout, _, err := val2.Exec(ctx, cmd, val2.Chain.Config().Env)
	require.NoError(err)

	fmt.Println("Tipped query 1")
	output := cosmos.CosmosTx{}
	err = json.Unmarshal(stdout, &output)
	require.NoError(err)
	fmt.Println("Transaction output for first tip:", output)

	// Tip query 2
	cmd = val3.TxCommand("validator", "oracle", "tip", queryData2, tipAmount.String(), "--keyring-dir", val3.HomeDir())
	stdout, _, err = val3.Exec(ctx, cmd, val3.Chain.Config().Env)
	require.NoError(err)

	fmt.Println("Tipped query 2")

	output = cosmos.CosmosTx{}
	err = json.Unmarshal(stdout, &output)
	require.NoError(err)
	fmt.Println("Transaction output for second tip:", output)

	// Tip query 3
	cmd = val4.TxCommand("validator", "oracle", "tip", queryData3, tipAmount.String(), "--keyring-dir", val4.HomeDir())
	stdout, _, err = val4.Exec(ctx, cmd, val4.Chain.Config().Env)
	require.NoError(err)

	fmt.Println("Tipped query 3")

	output = cosmos.CosmosTx{}
	err = json.Unmarshal(stdout, &output)
	require.NoError(err)
	fmt.Println("Transaction output for third tip:", output)
	// ======================================================================================

	// Now batch submit all three queries again - this time all should succeed
	fmt.Println("\n=== Batch submitting all three queries (expecting all to succeed) ===")
	// Execute second batch submit with JSON file
	cmd = val1.TxCommand("validator", "oracle", "batch-submit-value",
		"--values", value1,
		"--values", value2,
		"--values", value3,
		"--fees", "25loya",
		"--gas",
		"400000",
		"--keyring-dir",
		val1.HomeDir())

	stdout, _, err = val1.Exec(ctx, cmd, val1.Chain.Config().Env)
	require.NoError(err)
	var output2 cosmos.CosmosTx
	err = json.Unmarshal(stdout, &output2)
	require.NoError(err)
	fmt.Println("Second batch submit tx hash:", output2)
	require.NoError(testutil.WaitForBlocks(ctx, 4, val1))

	// Verify all three reports were created by querying reports
	fmt.Println("\n=== Verifying reports were created ===")

	microReports := make([]e2e.MicroReport, 3)
	// Query reports for each query ID
	for i, qDataBytes := range [][]byte{ethQueryData, btcQueryData, trbQueryData} {
		// Convert query data to query ID
		queryId := hex.EncodeToString(utils.QueryIDFromData(qDataBytes))

		// Query reports by query ID
		reportsRes, _, err := val1.ExecQuery(ctx, "oracle", "get-reportsby-qid", queryId, "--page-limit", "10")
		require.NoError(err)

		var reports e2e.QueryMicroReportsResponse
		err = json.Unmarshal(reportsRes, &reports)
		require.NoError(err)

		fmt.Printf("Query %d (ID: %s) has %d reports\n", i+1, queryId, len(reports.MicroReports))
		require.Greater(len(reports.MicroReports), 0, "Query %d should have at least one report", i+1)
		microReports[i] = reports.MicroReports[len(reports.MicroReports)-1]
		// Verify the latest report has the expected value
		if len(reports.MicroReports) > 0 {
			latestReport := reports.MicroReports[len(reports.MicroReports)-1]
			fmt.Printf("  Latest report value: %s\n", latestReport.Value)

			// Check if the value matches one of our submitted values
			expectedValues := value
			valueFound := false
			if latestReport.Value == expectedValues {
				valueFound = true
			}
			require.True(valueFound || latestReport.Value == value, "Report value should match one of the submitted values")
		}
	}

	// Query aggregates to verify they were created
	fmt.Println("\n=== Checking for aggregate reports ===")

	for i, qDataBytes := range [][]byte{ethQueryData, trbQueryData, btcQueryData} {
		// Convert query data to query ID
		queryId := hex.EncodeToString(utils.QueryIDFromData(qDataBytes))

		// Try to get current aggregate
		aggRes, _, err := val1.ExecQuery(ctx, "oracle", "get-current-aggregate-report", queryId)
		if err == nil {
			var aggregate e2e.QueryGetCurrentAggregateReportResponse
			err = json.Unmarshal(aggRes, &aggregate)
			if err == nil && string(aggregate.Aggregate.QueryId) != "" {
				fmt.Printf("Query %d has aggregate report with height %s\n", i+1, aggregate.Aggregate.Height)
			}
		}
	}

	fmt.Println("\n=== Batch submit test completed successfully ===")

	// dispute values submitted in batch by validator 1
	fmt.Println("\n=== Dispute a report that was submitted via batch ===")
	_, err = val1.ExecTx(
		ctx, "validator", "dispute", "propose-dispute",
		microReports[0].Reporter, microReports[0].MetaId,
		microReports[0].QueryID, warning, "500000000loya", "true", "--keyring-dir", val1.HomeDir(),
	)
	require.Error(err, "proposer cannot pay from their bond when creating a dispute on themselves")
	fmt.Println("Reporter power: ", microReports[0].Power)

	//
	val1valAddr, err := val1.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	val1StakingBefore, err := chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power before dispute: ", val1StakingBefore.Tokens)

	txHash, err = val2.ExecTx(
		ctx, "validator", "dispute", "propose-dispute",
		microReports[0].Reporter, microReports[0].MetaId,
		microReports[0].QueryID, warning, "500000000000loya", "false", "--keyring-dir", val2.HomeDir(), "--gas", "1000000", "--fees", "1000000loya",
	)
	require.NoError(err)
	fmt.Println("TX HASH (dispute on ", microReports[0].Reporter, "): ", txHash)
	txRes, _, err = val2.ExecQuery(ctx, "tx", txHash)
	require.NoError(err)
	fmt.Println("Transaction result for first submission:", string(txRes))

	val1StakingAfter, err := chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power after dispute: ", val1StakingAfter.Tokens)
	require.Equal(val1StakingAfter.Tokens, val1StakingBefore.Tokens.Sub(math.NewInt(50000*1e6)))

	openDisputesRes, _, err := val1.ExecQuery(ctx, "dispute", "open-disputes")
	require.NoError(err)
	var openDisputes e2e.QueryOpenDisputesResponse
	require.NoError(json.Unmarshal(openDisputesRes, &openDisputes))
	require.Greater(len(openDisputes.OpenDisputes.Ids), 0)
	fmt.Println("openDisputes: ", openDisputes.OpenDisputes.Ids)

	res, _, err := val1.ExecQuery(ctx, "oracle", "retrieve-data", microReports[0].QueryID, microReports[0].Timestamp)
	require.NoError(err)
	var data e2e.QueryRetrieveDataResponse
	require.NoError(json.Unmarshal(res, &data))
	require.Equal(data.Aggregate.Flagged, true)
}

// Helper function to get validators
func getValidators(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain) []struct {
	Val  *cosmos.ChainNode
	Addr string
} {
	t.Helper()
	validators := make([]struct {
		Val  *cosmos.ChainNode
		Addr string
	}, 0)

	for i, val := range chain.Validators {
		addr, err := val.AccountKeyBech32(ctx, "validator")
		require.NoError(t, err)
		validators = append(validators, struct {
			Val  *cosmos.ChainNode
			Addr string
		}{
			Val:  val,
			Addr: addr,
		})
		fmt.Printf("Validator %d address: %s\n", i+1, addr)
	}

	return validators
}
