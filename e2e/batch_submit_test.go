package e2e_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	"github.com/tellor-io/layer/utils"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// cleanupDockerContainers removes any existing Docker containers to prevent conflicts
func cleanupDockerContainers(t *testing.T) {
	t.Helper()
	// This is a best-effort cleanup - we don't fail the test if it doesn't work

	// Stop all running containers first
	cmd := exec.Command("sh", "-c", "docker stop $(docker ps -q)")
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to stop Docker containers: %v", err)
	}

	// Remove all stopped containers
	cmd = exec.Command("docker", "container", "prune", "-f")
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to cleanup Docker containers: %v", err)
	}

	// Remove all unused networks
	cmd = exec.Command("docker", "network", "prune", "-f")
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to cleanup Docker networks: %v", err)
	}

	// Remove all unused volumes
	cmd = exec.Command("docker", "volume", "prune", "-f")
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to cleanup Docker volumes: %v", err)
	}

	// Also try to remove any containers with our test pattern
	cmd = exec.Command("docker", "ps", "-a", "--filter", "name=TestBatchSubmitValue", "--format", "{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	containerIDs := strings.Fields(string(output))
	for _, id := range containerIDs {
		cmd = exec.Command("docker", "rm", "-f", id)
		if err := cmd.Run(); err != nil {
			t.Logf("Warning: Failed to remove container %s: %v", id, err)
		}
	}

	// Clean up any interchaintest containers
	cmd = exec.Command("docker", "ps", "-a", "--filter", "name=interchaintest", "--format", "{{.ID}}")
	output, err = cmd.Output()
	if err == nil {
		containerIDs = strings.Fields(string(output))
		for _, id := range containerIDs {
			cmd = exec.Command("docker", "rm", "-f", id)
			if err := cmd.Run(); err != nil {
				t.Logf("Warning: Failed to remove interchaintest container %s: %v", id, err)
			}
		}
	}
}

func TestBatchSubmitValue(t *testing.T) {
	require := require.New(t)

	// Clean up any existing containers before starting
	cleanupDockerContainers(t)

	// Set SDK config before parsing addresses
	cosmos.SetSDKConfig("tellor")

	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "20s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
		// Increase tip window from 2 blocks to 5 blocks for better test reliability
		cosmos.NewGenesisKV("app_state.registry.dataspec.0.report_block_window", "5"),
	}

	nv := 4
	nf := 0

	// Additional cleanup right before chain setup
	cleanupDockerContainers(t)

	chain, _, ctx := e2e.SetupTestChain(t, nv, nf, modifyGenesis)

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

	// Non-cycle list queries (using constants from dispute_test.go)
	trxQueryData, err := hex.DecodeString(trxQData)
	require.NoError(err)
	suiQueryData, err := hex.DecodeString(suiQData)
	require.NoError(err)
	bchQueryData, err := hex.DecodeString(bchQData)
	require.NoError(err)

	// convert to base64 for CLI
	queryData1 := base64.StdEncoding.EncodeToString(trxQueryData)
	queryData2 := base64.StdEncoding.EncodeToString(suiQueryData)
	queryData3 := base64.StdEncoding.EncodeToString(bchQueryData)

	fmt.Printf("Non-cycle list queries:\n1: TRX/USD\n2: SUI/USD\n3: BCH/USD\n")

	// Random value
	value := "000000000000000000000000000000000000000000000000000000000000001e" // hex encoded value (30)

	// Create strings for CLI input
	value1 := fmt.Sprintf(`{"query_data":"%s", "value":"%s"}`, queryData1, value)
	value2 := fmt.Sprintf(`{"query_data":"%s", "value":"%s"}`, queryData2, value)
	value3 := fmt.Sprintf(`{"query_data":"%s", "value":"%s"}`, queryData3, value)

	// Try to submit values for all three queries initially
	// Without tips, ALL should fail since these are not cycle list queries
	fmt.Println("\n=== Testing initial submission (expecting ALL to fail - no tips) ===")
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
	txRes, _, err := e2e.QueryWithTimeout(ctx, val1, "tx", txHash1)
	require.NoError(err)
	fmt.Println("Transaction result for first submission:", string(txRes))

	// ======================================================================================
	// Now tip all three queries to make them submittable
	fmt.Println("\n=== Tipping all three non-cycle list queries ===")
	tipAmount := sdk.NewCoin("loya", math.NewInt(1000000)) // 1 TRB

	// wait 1 block
	require.NoError(testutil.WaitForBlocks(ctx, 1, val1))

	// Tip query 1 (TRX/USD)
	cmd := val2.TxCommand("validator", "oracle", "tip", queryData1, tipAmount.String(), "--keyring-dir", val2.HomeDir())
	stdout, _, err := val2.Exec(ctx, cmd, val2.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("Tipped TRX/USD query")
	output1 := cosmos.CosmosTx{}
	err = json.Unmarshal(stdout, &output1)
	require.NoError(err)
	fmt.Println("Transaction output for TRX tip:", output1)

	// Tip query 2 (SUI/USD)
	cmd = val3.TxCommand("validator", "oracle", "tip", queryData2, tipAmount.String(), "--keyring-dir", val3.HomeDir())
	stdout, _, err = val3.Exec(ctx, cmd, val3.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("Tipped SUI/USD query")
	output2 := cosmos.CosmosTx{}
	err = json.Unmarshal(stdout, &output2)
	require.NoError(err)
	fmt.Println("Transaction output for SUI tip:", output2)

	// Tip query 3 (BCH/USD)
	cmd = val4.TxCommand("validator", "oracle", "tip", queryData3, tipAmount.String(), "--keyring-dir", val4.HomeDir())
	stdout, _, err = val4.Exec(ctx, cmd, val4.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("Tipped BCH/USD query")
	output3 := cosmos.CosmosTx{}
	err = json.Unmarshal(stdout, &output3)
	require.NoError(err)
	fmt.Println("Transaction output for BCH tip:", output3)

	// wait 1 block
	require.NoError(testutil.WaitForBlocks(ctx, 1, val1))

	// Verify all tip transactions were processed
	fmt.Println("\n=== Verifying tip transactions were processed ===")

	// Check tip transaction 1
	txRes1, _, err := e2e.QueryWithTimeout(ctx, val1, "tx", output1.TxHash)
	require.NoError(err)
	fmt.Println("Tip transaction 1 result:", string(txRes1))

	// Check tip transaction 2
	txRes2, _, err := e2e.QueryWithTimeout(ctx, val1, "tx", output2.TxHash)
	require.NoError(err)
	fmt.Println("Tip transaction 2 result:", string(txRes2))

	// Check tip transaction 3
	txRes3, _, err := e2e.QueryWithTimeout(ctx, val1, "tx", output3.TxHash)
	require.NoError(err)
	fmt.Println("Tip transaction 3 result:", string(txRes3))

	// Check if all queries are tipped
	fmt.Println("\n=== Checking tipped queries ===")
	tippedQueriesRes, _, err := e2e.QueryWithTimeout(ctx, val1, "oracle", "get-tipped-queries", "--page-limit", "10")
	require.NoError(err)
	fmt.Println("Tipped queries response:", string(tippedQueriesRes))
	// ======================================================================================

	// Now batch submit all three queries again - this time all should succeed
	fmt.Println("\n=== Batch submitting all three queries (expecting all to succeed after tips) ===")
	// Execute second batch submit
	txHash, err = val1.ExecTx(ctx, "validator", "oracle", "batch-submit-value",
		"--values", value1,
		"--values", value2,
		"--values", value3,
		"--fees", "25loya",
		"--gas", "400000",
		"--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("Second batch submit tx hash:", txHash)
	require.NoError(testutil.WaitForBlocks(ctx, 4, val1))

	// Verify all three reports were created by querying reports
	fmt.Println("\n=== Verifying reports were created ===")

	// wait 6 blocks (5 block reporting window), wait 6 to be safe
	require.NoError(testutil.WaitForBlocks(ctx, 6, val1))

	microReports := make([]e2e.MicroReport, 3)
	queryNames := []string{"TRX/USD", "SUI/USD", "BCH/USD"}
	// Query reports for each query ID
	for i, qDataBytes := range [][]byte{trxQueryData, suiQueryData, bchQueryData} {
		// Convert query data to query ID
		queryId := hex.EncodeToString(utils.QueryIDFromData(qDataBytes))

		// Query reports by query ID
		reportsRes, _, err := e2e.QueryWithTimeout(ctx, val1, "oracle", "get-reportsby-qid", queryId, "--page-limit", "10")
		require.NoError(err)

		// Debug: Print raw response before unmarshalling
		fmt.Printf("Raw reports response for %s: %s\n", queryNames[i], string(reportsRes))

		var reports e2e.QueryMicroReportsResponse
		err = json.Unmarshal(reportsRes, &reports)
		require.NoError(err)

		fmt.Printf("%s (ID: %s) has %d reports\n", queryNames[i], queryId, len(reports.MicroReports))
		require.Equal(len(reports.MicroReports), 1, "%s should have exactly 1 report", queryNames[i])
		microReports[i] = reports.MicroReports[0]

		// Verify the report has the expected value
		latestReport := reports.MicroReports[0]
		fmt.Printf("  Report value: %s\n", latestReport.Value)
		require.Equal(latestReport.Value, value, "%s report value should match submitted value", queryNames[i])
	}

	// Query aggregates to verify they were created
	fmt.Println("\n=== Checking for aggregate reports ===")

	for i, qDataBytes := range [][]byte{trxQueryData, suiQueryData, bchQueryData} {
		// Convert query data to query ID
		queryId := hex.EncodeToString(utils.QueryIDFromData(qDataBytes))

		// Try to get current aggregate
		aggRes, _, err := e2e.QueryWithTimeout(ctx, val1, "oracle", "get-current-aggregate-report", queryId)
		require.NoError(err, "Failed to query aggregate for %s", queryNames[i])

		var aggregate e2e.QueryGetCurrentAggregateReportResponse
		err = json.Unmarshal(aggRes, &aggregate)
		require.NoError(err, "Failed to unmarshal aggregate response for %s", queryNames[i])
		require.NotEmpty(aggregate.Aggregate.QueryId, "%s should have an aggregate report", queryNames[i])

		fmt.Printf("%s has aggregate report with height %s\n", queryNames[i], aggregate.Aggregate.Height)
	}

	fmt.Println("\n=== Batch submit test completed successfully ===")

	// Debug: Print microReports values before using them
	fmt.Printf("Debug: microReports[0] (TRX/USD) values:\n")
	fmt.Printf("  Reporter: %s\n", microReports[0].Reporter)
	fmt.Printf("  MetaId: %s\n", microReports[0].MetaId)
	fmt.Printf("  QueryID: %s\n", microReports[0].QueryID)
	fmt.Printf("  Power: %s\n", microReports[0].Power)
	fmt.Printf("  Value: %s\n", microReports[0].Value)
	fmt.Printf("  Timestamp: %s\n", microReports[0].Timestamp)

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
	txRes, _, err = e2e.QueryWithTimeout(ctx, val2, "tx", txHash)
	require.NoError(err)
	fmt.Println("Transaction result for first submission:", string(txRes))

	val1StakingAfter, err := chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power after dispute: ", val1StakingAfter.Tokens)
	require.Equal(val1StakingAfter.Tokens, val1StakingBefore.Tokens.Sub(math.NewInt(50000*1e6)))

	openDisputesRes, _, err := e2e.QueryWithTimeout(ctx, val1, "dispute", "open-disputes")
	require.NoError(err)
	var openDisputes e2e.QueryOpenDisputesResponse
	require.NoError(json.Unmarshal(openDisputesRes, &openDisputes))
	require.Greater(len(openDisputes.OpenDisputes.Ids), 0)
	fmt.Println("openDisputes: ", openDisputes.OpenDisputes.Ids)

	// Test retrieve-data functionality - use the aggregate's timestamp instead of micro report timestamp
	// First get the aggregate to find the correct timestamp
	aggRes, _, err := e2e.QueryWithTimeout(ctx, val1, "oracle", "get-current-aggregate-report", microReports[0].QueryID)
	require.NoError(err, "Failed to get aggregate for retrieve-data test")

	var aggregate e2e.QueryGetCurrentAggregateReportResponse
	err = json.Unmarshal(aggRes, &aggregate)
	require.NoError(err, "Failed to unmarshal aggregate for retrieve-data test")
	require.NotEmpty(aggregate.Aggregate.QueryId, "Aggregate should exist for retrieve-data test")

	// Use the aggregate's timestamp for retrieve-data
	res, _, err := e2e.QueryWithTimeout(ctx, val1, "oracle", "retrieve-data", microReports[0].QueryID, aggregate.Timestamp)
	require.NoError(err, "Failed to retrieve data")
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
