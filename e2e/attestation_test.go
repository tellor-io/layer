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

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const teamMnemonic = "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"

// cd e2e
// go test -run TestAttestation --timeout 5m

// check snapshot/attestation data for report1_consensus, report2_not_consensus, request_attestations_report2 (lastConsTs should equal report1 timestamp)
func TestAttestation(t *testing.T) {
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
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	type Validators struct {
		Addr    string
		ValAddr string
		Val     *cosmos.ChainNode
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
			Addr:    valAddr,
			ValAddr: valvalAddr,
			Val:     val,
		}
	}

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Val))
	require.NoError(testutil.WaitForBlocks(ctx, 5, validators[0].Val))
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
	res, _, err := validators[0].Val.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), 2)
	require.Contains(reportersRes.Reporters[0].Metadata.Moniker, "reporter_moniker")
	require.Contains(reportersRes.Reporters[1].Metadata.Moniker, "reporter_moniker")
	require.NotEqual(reportersRes.Reporters[0].Metadata.Moniker, reportersRes.Reporters[1].Metadata.Moniker)

	// validator reporters report for the cycle list
	currentCycleListRes, _, err := validators[0].Val.ExecQuery(ctx, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)
	fmt.Println("current cycle list: ", currentCycleList)
	for i, v := range validators {
		// report for the cycle list
		_, _, err = v.Val.Exec(ctx, v.Val.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "25loya", "--keyring-dir", v.Val.HomeDir()), v.Val.Chain.Config().Env)
		require.NoError(err)
		height, err := chain.Height(ctx)
		require.NoError(err)
		fmt.Println("validator [", i, "] reported at height ", height)
	}

	// wait for query to expire
	err = testutil.WaitForBlocks(ctx, 2, validators[0].Val)
	require.NoError(err)

	// check on reports
	var queryId1, queryId2 string
	for i, v := range validators {
		reports, _, err := v.Val.ExecQuery(ctx, "oracle", "get-reportsby-reporter", v.Addr)
		require.NoError(err)
		var reportsRes e2e.QueryMicroReportsResponse
		err = json.Unmarshal(reports, &reportsRes)
		require.NoError(err)
		fmt.Println("reports from: ", v.Addr, ": ", reportsRes)
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
	res, _, err = validators[0].Val.ExecQuery(ctx, "oracle", "get-current-aggregate-report", queryId1)
	require.NoError(err)
	var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
	err = json.Unmarshal(res, &currentAggRes)
	require.NoError(err)
	timestamp := currentAggRes.Timestamp

	// get snapshots
	snapshots, _, err := validators[0].Val.ExecQuery(ctx, "bridge", "get-snapshots-by-report", queryId1, timestamp)
	require.NoError(err)
	var snapshotsRes e2e.QueryGetSnapshotsByReportResponse
	err = json.Unmarshal(snapshots, &snapshotsRes)
	require.NoError(err)
	fmt.Println("snapshots: ", snapshotsRes)

	// get attestations by snapshot
	for _, snapshot := range snapshotsRes.Snapshots {
		attestations, _, err := validators[0].Val.ExecQuery(ctx, "bridge", "get-attestation-by-snapshot", snapshot)
		require.NoError(err)
		fmt.Println("attestations bz: ", attestations)
		var attestationsRes e2e.QueryGetAttestationDataBySnapshotResponse
		err = json.Unmarshal(attestations, &attestationsRes)
		require.NoError(err)
		fmt.Println("attestations: ", attestationsRes) // investigate why this is empty, bytes are not empty

		// get attestation data by snapshot
		fmt.Println("snapshot: ", snapshot)
		attestationData, _, err := validators[0].Val.ExecQuery(ctx, "bridge", "get-attestation-data-by-snapshot", snapshot)
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

	// sleep until cycle list query data is the same as the previous report
	var success bool
	var cycleListQData string
	for !success {
		cycleListRes, _, err := validators[0].Val.ExecQuery(ctx, "oracle", "current-cyclelist-query")
		require.NoError(err)
		var cycleList e2e.QueryCurrentCyclelistQueryResponse
		err = json.Unmarshal(cycleListRes, &cycleList)
		require.NoError(err)
		if cycleList.QueryData == currentCycleList.QueryData {
			success = true
			cycleListQData = cycleList.QueryData
		}
		time.Sleep(1 * time.Second)
	}

	// report for the cycle list from 1 val so not a consensus report
	_, _, err = validators[0].Val.Exec(ctx, validators[0].Val.TxCommand("validator", "oracle", "submit-value", cycleListQData, value, "--fees", "25loya", "--keyring-dir", validators[0].Val.HomeDir()), validators[0].Val.Chain.Config().Env)
	require.NoError(err)
	height, err := chain.Height(ctx)
	require.NoError(err)
	fmt.Println("validator [0] reported at height ", height)

	// wait 2 blocks for query to expire
	err = testutil.WaitForBlocks(ctx, 2, validators[0].Val)
	require.NoError(err)

	// get reports by reporter
	var queryId3 string
	reports, _, err := validators[0].Val.ExecQuery(ctx, "oracle", "get-reportsby-reporter", validators[0].Addr)
	require.NoError(err)
	var reportsRes e2e.QueryMicroReportsResponse
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(err)
	fmt.Println("reports from: ", validators[0].Addr, ": ", reportsRes)
	require.Equal(len(reportsRes.MicroReports), 2) // val0 should have two reports now
	queryId3 = reportsRes.MicroReports[0].QueryID
	require.Equal(queryId1, queryId3) // make sure query is same as first report, then we can reuse the decoded queryId from earlier

	// query GetCurrentAggregateReport to get aggregate timestamp
	res, _, err = validators[0].Val.ExecQuery(ctx, "oracle", "get-current-aggregate-report", queryId1)
	require.NoError(err)
	err = json.Unmarshal(res, &currentAggRes)
	require.NoError(err)
	prevTimestamp := timestamp
	timestamp = currentAggRes.Timestamp
	fmt.Println("timestamp: ", timestamp)
	fmt.Println("currentAggRes: ", currentAggRes)

	// get snapshots
	snapshots, _, err = validators[0].Val.ExecQuery(ctx, "bridge", "get-snapshots-by-report", queryId1, timestamp)
	require.NoError(err)
	err = json.Unmarshal(snapshots, &snapshotsRes)
	require.NoError(err)
	fmt.Println("snapshots: ", snapshotsRes)

	// get attestations by snapshot
	for _, snapshot := range snapshotsRes.Snapshots {
		attestations, _, err := validators[0].Val.ExecQuery(ctx, "bridge", "get-attestation-by-snapshot", snapshot)
		require.NoError(err)
		fmt.Println("attestations bz: ", attestations)
		var attestationsRes e2e.QueryGetAttestationDataBySnapshotResponse
		err = json.Unmarshal(attestations, &attestationsRes)
		require.NoError(err)
		fmt.Println("attestations: ", attestationsRes) // investigate why this is empty, bytes are not empty

		// get attestation data by snapshot
		fmt.Println("snapshot: ", snapshot)
		attestationData, _, err := validators[0].Val.ExecQuery(ctx, "bridge", "get-attestation-data-by-snapshot", snapshot)
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
	err = testutil.WaitForBlocks(ctx, 5, validators[0].Val)
	require.NoError(err)
	txHash, err := validators[0].Val.ExecTx(ctx, "validator", "bridge", "request-attestations", validators[0].Addr, queryId1, timestamp, "--keyring-dir", validators[0].Val.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val0 requests attestation for report2): ", txHash)

	// get snapshots by report
	snapshots, _, err = validators[0].Val.ExecQuery(ctx, "bridge", "get-snapshots-by-report", queryId1, timestamp)
	require.NoError(err)
	err = json.Unmarshal(snapshots, &snapshotsRes)
	require.NoError(err)
	fmt.Println("snapshots: ", snapshotsRes)
	require.Equal(len(snapshotsRes.Snapshots), 2) // should be auto generated plus additional requested snapshot

	// get attestation data for new attestation request
	attestationData, _, err := validators[0].Val.ExecQuery(ctx, "bridge", "get-attestation-data-by-snapshot", snapshotsRes.Snapshots[1])
	require.NoError(err)
	var attestationDataRes e2e.QueryGetAttestationDataBySnapshotResponse
	err = json.Unmarshal(attestationData, &attestationDataRes)
	require.NoError(err)
	require.Equal(attestationDataRes.QueryId, queryId1)
	require.Equal(attestationDataRes.Timestamp, timestamp)
	require.Equal(attestationDataRes.AggregateValue, value)
	require.Equal(attestationDataRes.AggregatePower, "5000000")         // 50% power
	require.NotNil(attestationDataRes.Checkpoint)                       // validator checkpoint not nil
	require.Greater(attestationDataRes.AttestationTimestamp, timestamp) // attestation was after report2 timestamp
	require.Equal(attestationDataRes.PreviousReportTimestamp, prevTimestamp)
	require.Equal(attestationDataRes.NextReportTimestamp, "0")
	require.Equal(attestationDataRes.LastConsensusTimestamp, prevTimestamp) // lastConsTs should equal report1 timestamp
	fmt.Println("attestationData: ", attestationDataRes)

	// claim validator rewards
	txHash, err = validators[0].Val.ExecTx(ctx, "validator", "distribution", "withdraw-all-rewards", "--keyring-dir", validators[0].Val.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val0 claims rewards): ", txHash)
}

func TestNoStakeAttestation(t *testing.T) {
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
	}

	nv := 2
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
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	type Validators struct {
		Addr    string
		ValAddr string
		Val     *cosmos.ChainNode
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
			Addr:    valAddr,
			ValAddr: valvalAddr,
			Val:     val,
		}
	}

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Val))
	require.NoError(testutil.WaitForBlocks(ctx, 5, validators[0].Val))
	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")

	// both validators submit no stake reports
	for i, v := range validators {
		txHash, err := v.Val.ExecTx(ctx, "validator", "oracle", "no-stake-report", ltcQData, value, "--fees", "25loya", "--keyring-dir", v.Val.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (val", i, " reports no stake): ", txHash)
	}

	// query no stake reports for each validator
	for _, v := range validators {
		reports, _, err := v.Val.ExecQuery(ctx, "oracle", "get-reporters-no-stake-reports", v.Addr)
		require.NoError(err)
		var nsReportsRes e2e.QueryGetReportersNoStakeReportsResponse
		err = json.Unmarshal(reports, &nsReportsRes)
		require.NoError(err)
		fmt.Println("nsReportsRes: ", nsReportsRes.NoStakeReports[0])
		require.Equal(len(nsReportsRes.NoStakeReports), 1)
		require.Equal(nsReportsRes.NoStakeReports[0].QueryData, ltcQData)
		require.Equal(nsReportsRes.NoStakeReports[0].Value, value)

	}

	// query no stake reports per queryId
	reports, _, err := validators[0].Val.ExecQuery(ctx, "oracle", "get-no-stake-reports-by-query-id", ltcQId, "--page-limit", "2")
	require.NoError(err)
	var nsReportsByQIdRes e2e.QueryGetNoStakeReportsByQueryIdResponse
	err = json.Unmarshal(reports, &nsReportsByQIdRes)
	require.NoError(err)
	fmt.Println("nsReportsByQIdRes 0 : ", nsReportsByQIdRes.NoStakeReports[0])
	require.Equal(len(nsReportsByQIdRes.NoStakeReports), 2)
	require.Equal(nsReportsByQIdRes.NoStakeReports[0].QueryData, ltcQData)
	require.Equal(nsReportsByQIdRes.NoStakeReports[1].QueryData, ltcQData)
	require.Equal(nsReportsByQIdRes.NoStakeReports[0].Value, value)
	require.Equal(nsReportsByQIdRes.NoStakeReports[1].Value, value)

	// request an attestation for the first report
	timestamp := nsReportsByQIdRes.NoStakeReports[0].Timestamp
	txHash, err := validators[0].Val.ExecTx(ctx, "validator", "bridge", "request-attestations", validators[0].Addr, ltcQId, timestamp, "--keyring-dir", validators[0].Val.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val0 requests attestation for report1): ", txHash)

	// get snapshot by report
	res, _, err := validators[0].Val.ExecQuery(ctx, "bridge", "get-snapshots-by-report", ltcQId, timestamp)
	require.NoError(err)
	var snapshotsRes e2e.QueryGetSnapshotsByReportResponse
	err = json.Unmarshal(res, &snapshotsRes)
	require.NoError(err)
	fmt.Println("snapshotsRes: ", snapshotsRes)
	require.Equal(len(snapshotsRes.Snapshots), 1)

	// get attestation data by snapshot
	attestationData, _, err := validators[0].Val.ExecQuery(ctx, "bridge", "get-attestation-data-by-snapshot", snapshotsRes.Snapshots[0])
	require.NoError(err)
	var attestationDataRes e2e.QueryGetAttestationDataBySnapshotResponse
	err = json.Unmarshal(attestationData, &attestationDataRes)
	require.NoError(err)
	fmt.Println("attestationDataRes: ", attestationDataRes)
	require.Equal(attestationDataRes.QueryId, ltcQId)
	require.Equal(attestationDataRes.Timestamp, timestamp)
	require.NotNil(attestationDataRes.AggregateValue)
	require.Equal(attestationDataRes.AggregatePower, "0")
	require.NotNil(attestationDataRes.Checkpoint)                       // validator checkpoint not nil
	require.Greater(attestationDataRes.AttestationTimestamp, timestamp) // attestation was after report timestamp
	require.Equal(attestationDataRes.PreviousReportTimestamp, "0")
	require.Equal(attestationDataRes.NextReportTimestamp, "0")
	require.Equal(attestationDataRes.LastConsensusTimestamp, "0")
}
