package e2e_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// check snapshot/attestation data for report1_consensus, report2_not_consensus, request_attestations_report2 (lastConsTs should equal report1 timestamp)
func TestConsensusAttestation(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChainWithCustomConfig(t, e2e.DefaultSetupConfig())
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validators)

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Node))
	require.NoError(testutil.WaitForBlocks(ctx, 5, validators[0].Node))
	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")

	// both validators become reporters
	for i, val := range chain.Validators {
		moniker := fmt.Sprintf("reporter_moniker%d", i)
		txHash, err := val.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.5", "100000000", moniker, "--keyring-dir", val.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (val", i+1, "becomes a reporter): ", txHash)
	}

	// query reporters
	res, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), 2)
	require.Contains(reportersRes.Reporters[0].Metadata.Moniker, "reporter_moniker")
	require.Contains(reportersRes.Reporters[1].Metadata.Moniker, "reporter_moniker")
	require.NotEqual(reportersRes.Reporters[0].Metadata.Moniker, reportersRes.Reporters[1].Metadata.Moniker)

	currentCycleListRes, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)

	// wait 1 block
	require.NoError(testutil.WaitForBlocks(ctx, 1, validators[0].Node))

	// validators' reporters report for the cycle list
	var txHash1, txHash2 string
	var err1, err2 error

	txHash1, err1 = e2e.SubmitCycleList(ctx, validators[0].Node, validators[0].AccAddr, value, "5loya")
	txHash2, err2 = e2e.SubmitCycleList(ctx, validators[1].Node, validators[1].AccAddr, value, "5loya")

	require.NoError(err1, "validator[0] report failed")
	require.NoError(err2, "validator[1] report failed")

	fmt.Println("validator [ 0 ] reported at tx:", txHash1)
	fmt.Println("validator [ 1 ] reported at tx:", txHash2)

	// Wait 1 block for both reports to be included
	err = testutil.WaitForBlocks(ctx, 1, validators[0].Node)
	require.NoError(err)

	// Wait 1 more block for aggregation
	err = testutil.WaitForBlocks(ctx, 1, validators[0].Node)
	require.NoError(err)

	// check on reports
	var queryId1, queryId2 string
	for i, v := range validators {
		reports, _, err := e2e.QueryWithTimeout(ctx, v.Node, "oracle", "get-reportsby-reporter", v.AccAddr, "--page-limit", "1")
		require.NoError(err)
		var reportsRes e2e.QueryMicroReportsResponse
		err = json.Unmarshal(reports, &reportsRes)
		require.NoError(err)
		fmt.Println("reports from: ", v.AccAddr, ": ", reportsRes)
		require.Equal(len(reportsRes.MicroReports), 1) // each reporter should have one report
		if i == 0 {
			queryId1 = reportsRes.MicroReports[0].QueryID
		} else {
			queryId2 = reportsRes.MicroReports[0].QueryID
		}
	}
	require.Equal(queryId1, queryId2) // make sure both reporters reported for the same query
	fmt.Println("queryId1: ", queryId1)
	fmt.Println("queryId2: ", queryId2)

	// query GetCurrentAggregateReport to get aggregate timestamp
	res, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "get-current-aggregate-report", queryId1)
	require.NoError(err)
	var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
	err = json.Unmarshal(res, &currentAggRes)
	require.NoError(err)
	timestamp := currentAggRes.Timestamp

	// get snapshots
	snapshots, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-snapshots-by-report", queryId1, timestamp)
	require.NoError(err)
	var snapshotsRes e2e.QueryGetSnapshotsByReportResponse
	err = json.Unmarshal(snapshots, &snapshotsRes)
	require.NoError(err)
	fmt.Println("snapshots: ", snapshotsRes)

	// get attestations by snapshot
	for _, snapshot := range snapshotsRes.Snapshots {
		attestations, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-attestation-by-snapshot", snapshot)
		require.NoError(err)
		fmt.Println("attestations bz: ", attestations)
		var attestationsRes e2e.QueryGetAttestationDataBySnapshotResponse
		err = json.Unmarshal(attestations, &attestationsRes)
		require.NoError(err)

		// get attestation data by snapshot
		fmt.Println("snapshot: ", snapshot)
		attestationData, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-attestation-data-by-snapshot", snapshot)
		require.NoError(err)
		var attestationDataRes e2e.QueryGetAttestationDataBySnapshotResponse
		err = json.Unmarshal(attestationData, &attestationDataRes)
		require.NoError(err)
		require.Equal(attestationDataRes.QueryId, queryId1)
		require.Equal(attestationDataRes.Timestamp, timestamp)
		require.Equal(attestationDataRes.AggregateValue, value)
		require.Equal(attestationDataRes.AggregatePower, "10000000") // 100% power
		require.NotNil(attestationDataRes.Checkpoint)                // validator checkpoint not nil
		require.Equal(attestationDataRes.AttestationTimestamp, timestamp)
		require.Equal(attestationDataRes.PreviousReportTimestamp, "0")      // first report for qId
		require.Equal(attestationDataRes.NextReportTimestamp, "0")          // first report for qId
		require.Equal(attestationDataRes.LastConsensusTimestamp, timestamp) // lastConsTs should equal report1 timestamp
	}

	// Wait until the same cycle-list query is live again, then report from only
	// one validator (non-consensus). SpotPrice windows are 2 blocks and the
	// cycle list has only 3 queries, so SubmitCycleList's re-query can miss the
	// window and land on a different query — leaving get-current-aggregate on
	// the first 100% report. Submit the matched QueryData directly and retry
	// if the window already expired.
	prevTimestamp := timestamp
	var success bool
	deadline := time.Now().Add(2 * time.Minute)
	for !success {
		require.True(time.Now().Before(deadline), "timed out waiting for non-consensus second report")

		cycleListRes, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "current-cyclelist-query")
		require.NoError(err)
		var cycleList e2e.QueryCurrentCyclelistQueryResponse
		err = json.Unmarshal(cycleListRes, &cycleList)
		require.NoError(err)
		if cycleList.QueryData != currentCycleList.QueryData {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		txHashBytes, _, err := validators[0].Node.Exec(ctx,
			validators[0].Node.TxCommand(validators[0].AccAddr, "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "5loya", "--keyring-dir", validators[0].Node.HomeDir()),
			validators[0].Node.Chain.Config().Env)
		if err != nil {
			fmt.Println("second-round submit failed (likely window expired), retrying:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		txHash, err := e2e.GetTxHashFromExec(txHashBytes)
		if err != nil {
			fmt.Println("second-round submit tx failed, retrying:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		fmt.Println("validator [ 0 ] reported at tx:", txHash)

		// Wait for inclusion + aggregation, then confirm current aggregate moved
		// to a new timestamp with 50% power before continuing.
		require.NoError(testutil.WaitForBlocks(ctx, 3, validators[0].Node))
		for i := 0; i < 10; i++ {
			res, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "get-current-aggregate-report", queryId1)
			require.NoError(err)
			err = json.Unmarshal(res, &currentAggRes)
			require.NoError(err)
			if currentAggRes.Timestamp != prevTimestamp && currentAggRes.Aggregate != nil && currentAggRes.Aggregate.AggregatePower == "5000000" {
				success = true
				break
			}
			require.NoError(testutil.WaitForBlocks(ctx, 1, validators[0].Node))
		}
		if !success {
			fmt.Println("current aggregate not yet non-consensus for queryId1; waiting for next cycle")
		}
	}

	// Newest report must be for the same queryId as the first consensus report.
	reports, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "get-reportsby-reporter", validators[0].AccAddr, "--page-limit", "1", "--page-reverse")
	require.NoError(err)
	var reportsRes e2e.QueryMicroReportsResponse
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(err)
	fmt.Println("newest report from: ", validators[0].AccAddr, ": ", reportsRes)
	require.Equal(len(reportsRes.MicroReports), 1)
	require.Equal(reportsRes.MicroReports[0].QueryID, queryId1)

	timestamp = currentAggRes.Timestamp
	fmt.Println("timestamp: ", timestamp)
	fmt.Println("currentAggRes: ", currentAggRes)

	// get snapshots
	snapshots, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-snapshots-by-report", queryId1, timestamp)
	require.NoError(err)
	err = json.Unmarshal(snapshots, &snapshotsRes)
	require.NoError(err)
	fmt.Println("snapshots: ", snapshotsRes)

	// get attestations by snapshot
	for _, snapshot := range snapshotsRes.Snapshots {
		attestations, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-attestation-by-snapshot", snapshot)
		require.NoError(err)
		fmt.Println("attestations bz: ", attestations)
		var attestationsRes e2e.QueryGetAttestationDataBySnapshotResponse
		err = json.Unmarshal(attestations, &attestationsRes)
		require.NoError(err)

		// get attestation data by snapshot
		fmt.Println("snapshot: ", snapshot)
		attestationData, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-attestation-data-by-snapshot", snapshot)
		require.NoError(err)
		var attestationDataRes e2e.QueryGetAttestationDataBySnapshotResponse
		err = json.Unmarshal(attestationData, &attestationDataRes)
		require.NoError(err)
		require.Equal(attestationDataRes.QueryId, queryId1)
		require.Equal(attestationDataRes.Timestamp, timestamp)
		require.Equal(attestationDataRes.AggregateValue, value)
		require.Equal(attestationDataRes.AggregatePower, "5000000") // 50% power
		require.NotNil(attestationDataRes.Checkpoint)               // validator checkpoint not nil
		require.Equal(attestationDataRes.AttestationTimestamp, timestamp)
		require.Equal(attestationDataRes.PreviousReportTimestamp, prevTimestamp)
		require.Equal(attestationDataRes.NextReportTimestamp, "0")              // first report for qId
		require.Equal(attestationDataRes.LastConsensusTimestamp, prevTimestamp) // lastConsTs should equal report1 timestamp
	}

	// wait and request attestations for report2
	err = testutil.WaitForBlocks(ctx, 5, validators[0].Node)
	require.NoError(err)
	txHash, err := validators[0].Node.ExecTx(ctx, "validator", "bridge", "request-attestations", validators[0].AccAddr, queryId1, timestamp, "--keyring-dir", validators[0].Node.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val0 requests attestation for report2): ", txHash)

	// get snapshots by report
	snapshots, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-snapshots-by-report", queryId1, timestamp)
	require.NoError(err)
	err = json.Unmarshal(snapshots, &snapshotsRes)
	require.NoError(err)
	fmt.Println("snapshots: ", snapshotsRes)
	require.Equal(len(snapshotsRes.Snapshots), 2) // should be auto generated plus additional requested snapshot

	// get attestation data for new attestation request
	attestationData, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-attestation-data-by-snapshot", snapshotsRes.Snapshots[1])
	require.NoError(err)
	var attestationDataRes e2e.QueryGetAttestationDataBySnapshotResponse
	err = json.Unmarshal(attestationData, &attestationDataRes)
	require.NoError(err)
	require.Equal(attestationDataRes.QueryId, queryId1)
	require.Equal(attestationDataRes.Timestamp, timestamp)
	require.Equal(attestationDataRes.AggregateValue, value)
	require.Equal(attestationDataRes.AggregatePower, "5000000") // 50% power
	require.NotNil(attestationDataRes.Checkpoint)               // validator checkpoint not nil
	attestationTimestamp, err := strconv.ParseUint(attestationDataRes.AttestationTimestamp, 10, 64)
	require.NoError(err)
	timestampInt, err := strconv.ParseUint(attestationDataRes.Timestamp, 10, 64)
	require.NoError(err)
	require.Greater(attestationTimestamp, timestampInt) // attestation was after report2 timestamp
	require.Equal(attestationDataRes.PreviousReportTimestamp, prevTimestamp)
	require.Equal(attestationDataRes.NextReportTimestamp, "0")
	require.Equal(attestationDataRes.LastConsensusTimestamp, prevTimestamp) // lastConsTs should equal report1 timestamp
	fmt.Println("attestationData: ", attestationDataRes)

	// claim validator rewards
	txHash, err = validators[0].Node.ExecTx(ctx, "validator", "distribution", "withdraw-all-rewards", "--keyring-dir", validators[0].Node.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val0 claims rewards): ", txHash)
}
