package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	"go.uber.org/zap/zaptest"

	"cosmossdk.io/math"

	types1 "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/query"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// HELPERS FOR BUILDING THE CHAIN

var (
	layerImageInfo = []ibc.DockerImage{
		{
			Repository: "layer",
			Version:    "local",
			UidGid:     "1025:1025",
		},
	}
	numVals      = 2
	numFullNodes = 2

	baseBech32 = "tellor"

	teamMnemonic = "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"
)

func LayerSpinup(t *testing.T) *cosmos.CosmosChain {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	cosmos.SetSDKConfig(baseBech32)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		LayerChainSpec(numVals, numFullNodes, "layer-1"),
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	layer := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(layer)

	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	require.NoError(t, ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})
	require.NoError(t, layer.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(t, layer.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	return layer
}

func LayerChainSpec(nv, nf int, chainId string) *interchaintest.ChainSpec {
	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
	}
	return &interchaintest.ChainSpec{
		NumValidators: &nv,
		NumFullNodes:  &nf,
		ChainConfig: ibc.ChainConfig{
			Type:                "cosmos",
			Name:                "layer",
			ChainID:             chainId,
			Bin:                 "layerd",
			Denom:               "loya",
			Bech32Prefix:        "tellor",
			CoinType:            "118",
			GasPrices:           "0.0025loya",
			GasAdjustment:       1.1,
			TrustingPeriod:      "504h",
			NoHostMount:         false,
			Images:              layerImageInfo,
			EncodingConfig:      LayerEncoding(),
			ModifyGenesis:       cosmos.ModifyGenesis(modifyGenesis),
			AdditionalStartArgs: []string{"--key-name", "validator"},
			PreGenesis:          pregenesis(),
		},
	}
}

func LayerEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()
	return &cfg
}

// for adding the secrets file required for bridging
func WriteSecretsFile(ctx context.Context, rpc, bridge string, tn *cosmos.ChainNode) error {
	secrets := []byte(`{
		"eth_rpc_url": "` + rpc + `",
		"token_bridge_contract": "` + bridge + `",
	}`)
	fmt.Println("Writing secrets file")
	return tn.WriteFile(ctx, secrets, "secrets.yaml")
}

func pregenesis() func(ibc.Chain) error {
	return func(chain ibc.Chain) error {
		layer := chain.(*cosmos.CosmosChain)
		for _, node := range layer.Validators {
			if err := WriteSecretsFile(context.Background(), "", "", node); err != nil {
				return err
			}
		}

		return nil
	}
}

// for unmarshalling the disputes response
type Disputes struct {
	Disputes []struct {
		DisputeID string   `json:"disputeId"`
		Metadata  Metadata `json:"metadata"`
	} `json:"disputes"`
}

type Metadata struct {
	HashID            string   `json:"hash_id"`
	DisputeID         string   `json:"dispute_id"`
	DisputeCategory   int      `json:"dispute_category"`
	DisputeFee        string   `json:"dispute_fee"`
	DisputeStatus     int      `json:"dispute_status"`
	DisputeStartTime  string   `json:"dispute_start_time"`
	DisputeEndTime    string   `json:"dispute_end_time"`
	DisputeStartBlock string   `json:"dispute_start_block"`
	DisputeRound      string   `json:"dispute_round"`
	SlashAmount       string   `json:"slash_amount"`
	BurnAmount        string   `json:"burn_amount"`
	InitialEvidence   Evidence `json:"initial_evidence"`
	FeeTotal          string   `json:"fee_total"`
	PrevDisputeIDs    []string `json:"prev_dispute_ids"`
	BlockNumber       string   `json:"block_number"`
	VoterReward       string   `json:"voter_reward"`
}

type MetaData2 struct {
	// unique dispute hash identifier
	HashId string `protobuf:"bytes,1,opt,name=hash_id,json=hashId,proto3" json:"hash_id,omitempty"`
	// current dispute id
	DisputeId string `protobuf:"varint,2,opt,name=dispute_id,json=disputeId,proto3" json:"dispute_id,omitempty"`
	// dispute severity level
	DisputeCategory int `protobuf:"varint,3,opt,name=dispute_category,json=disputeCategory,proto3,enum=layer.dispute.DisputeCategory" json:"dispute_category,omitempty"`
	// cost to start dispute
	DisputeFee string `protobuf:"bytes,4,opt,name=dispute_fee,json=disputeFee,proto3,customtype=cosmossdk.io/math.Int" json:"dispute_fee"`
	// current dispute status
	DisputeStatus int `protobuf:"varint,5,opt,name=dispute_status,json=disputeStatus,proto3,enum=layer.dispute.DisputeStatus" json:"dispute_status,omitempty"`
	// start time of the dispute that begins after dispute fee is fully paid
	DisputeStartTime string `protobuf:"bytes,6,opt,name=dispute_start_time,json=disputeStartTime,proto3,stdtime" json:"dispute_start_time"`
	// end time that the dispute stop taking votes and creating new rounds
	DisputeEndTime string `protobuf:"bytes,7,opt,name=dispute_end_time,json=disputeEndTime,proto3,stdtime" json:"dispute_end_time"`
	// height of the block that started the dispute
	DisputeStartBlock string `protobuf:"varint,8,opt,name=dispute_start_block,json=disputeStartBlock,proto3" json:"dispute_start_block,omitempty"`
	// current dispute round
	DisputeRound string `protobuf:"varint,9,opt,name=dispute_round,json=disputeRound,proto3" json:"dispute_round,omitempty"`
	// reporter's slashed amount
	SlashAmount string `protobuf:"bytes,10,opt,name=slash_amount,json=slashAmount,proto3,customtype=cosmossdk.io/math.Int" json:"slash_amount"`
	// burn amount that will be divided in half and paid to voters and the other half burned
	BurnAmount string `protobuf:"bytes,11,opt,name=burn_amount,json=burnAmount,proto3,customtype=cosmossdk.io/math.Int" json:"burn_amount"`
	// initial single report evidence to be disputed
	InitialEvidence Evidence `protobuf:"bytes,12,opt,name=initial_evidence,json=initialEvidence,proto3" json:"initial_evidence"`
	// fee payers that were involved in paying the dispute fee in order to start the dispute
	// total fee paid tracked to know if dispute fee is fully paid to start dispute
	FeeTotal string `protobuf:"bytes,13,opt,name=fee_total,json=feeTotal,proto3,customtype=cosmossdk.io/math.Int" json:"fee_total"`
	// list of dispute ids that preceded before this current round began
	PrevDisputeIds []string `protobuf:"varint,14,rep,packed,name=prev_dispute_ids,json=prevDisputeIds,proto3" json:"prev_dispute_ids,omitempty"`
	// block number when this specific dispute was created
	BlockNumber        string     `protobuf:"varint,15,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	Open               bool       `protobuf:"varint,16,opt,name=open,proto3" json:"open,omitempty"`
	AdditionalEvidence []Evidence `protobuf:"bytes,17,rep,name=additional_evidence,json=additionalEvidence,proto3" json:"additional_evidence,omitempty"`
	// total tokens allocated to voters
	VoterReward string `protobuf:"bytes,18,opt,name=voter_reward,json=voterReward,proto3,customtype=cosmossdk.io/math.Int" json:"voter_reward"`
	// pending execution is true if the dispute has reached quorum and is pending execution.
	// however, if a new dispute round begins, this is set to false again
	PendingExecution bool `protobuf:"varint,19,opt,name=pending_execution,json=pendingExecution,proto3" json:"pending_execution,omitempty"`
}

type Disputes2 struct {
	Disputes []struct {
		DisputeID string    `json:"disputeId"`
		Metadata  MetaData2 `json:"metadata"`
	} `json:"disputes"`
}

type Evidence struct {
	Reporter        string `json:"reporter"`
	Power           string `json:"power"`
	QueryType       string `json:"query_type"`
	QueryID         string `json:"query_id"`
	AggregateMethod string `json:"aggregate_method"`
	Value           string `json:"value"`
	Timestamp       string `json:"timestamp"`
	BlockNumber     string `json:"block_number"`
}

type MicroReport struct {
	Reporter        string `json:"reporter"`
	Power           string `json:"power"`
	QueryType       string `json:"query_type"`
	QueryID         string `json:"query_id"`
	AggregateMethod string `json:"aggregate_method"`
	Value           string `json:"value"`
	Timestamp       string `json:"timestamp"`
	BlockNumber     string `json:"block_number"`
	MetaId          string `json:"meta_id"`
}

type ReportsResponse struct {
	MicroReports []MicroReport `json:"microReports"`
}

type AggregateReport struct {
	Aggregate struct {
		QueryID           string `json:"query_id"`
		AggregateValue    string `json:"aggregate_value"`
		AggregateReporter string `json:"aggregate_reporter"`
		ReporterPower     string `json:"reporter_power"`
		Reporters         []struct {
			Reporter    string `json:"reporter"`
			Power       string `json:"power"`
			BlockNumber string `json:"block_number"`
		} `json:"reporters"`
		Index       string `json:"index"`
		Height      string `json:"height"`
		MicroHeight string `json:"micro_height"`
		MetaID      string `json:"meta_id"`
	} `json:"aggregate"`
	Timestamp string `json:"timestamp"`
}

type Proposal struct {
	Messages  []map[string]interface{} `json:"messages"`
	Metadata  string                   `json:"metadata"`
	Deposit   string                   `json:"deposit"`
	Title     string                   `json:"title"`
	Summary   string                   `json:"summary"`
	Expedited bool                     `json:"expedited"`
}

type CurrentTipsResponse struct {
	Tips math.Int `json:"tips"`
}

type DataSpecResponse struct {
	DocumentHash      string                        `json:"document_hash,omitempty"`
	ResponseValueType string                        `json:"response_value_type,omitempty"`
	AbiComponents     []*registrytypes.ABIComponent `json:"abi_components,omitempty"`
	AggregationMethod string                        `json:"aggregation_method,omitempty"`
	Registrar         string                        `json:"registrar,omitempty"`
	ReportBlockWindow string                        `json:"report_block_window,omitempty"`
}

type GetDataSpecResponse struct {
	Registrar string           `json:"registrar"`
	QueryType string           `json:"query_type"`
	Spec      DataSpecResponse `json:"spec"`
}

type GenerateQueryDataResponse struct {
	QueryData []byte `json:"query_data"`
}

type TippedQueriesResponse struct {
	Queries []QueryMeta `json:"queries"`
}

type QueryDelegatorDelegationsResponse struct {
	DelegationResponses []stakingtypes.DelegationResponse `json:"delegation_responses"`
	Pagination          struct {
		Total string `json:"total"` // Change from uint64 to string
	} `json:"pagination"`
}

type QueryReportersResponse struct {
	// all the reporters.
	Reporters []*Reporter `protobuf:"bytes,1,rep,name=reporters,proto3" json:"reporters,omitempty"`
	// pagination defines the pagination in the response.
	Pagination struct {
		Total string `json:"total"`
	} `json:"pagination"`
}

type QueryMeta struct {
	// unique id of the query that changes after query's lifecycle ends
	Id string `json:"id,omitempty"`
	// amount of tokens that was tipped
	Amount string `json:"amount"`
	// expiration time of the query
	Expiration string `json:"expiration,omitempty"`
	// timeframe of the query according to the data spec
	RegistrySpecBlockWindow string `json:"registry_spec_block_window,omitempty"`
	// indicates whether query has revealed reports
	HasRevealedReports bool `json:"has_revealed_reports,omitempty"`
	// query_data: decodable bytes to field of the data spec
	QueryData string `json:"query_data,omitempty"`
	// string identifier of the data spec
	QueryType string `json:"query_type,omitempty"`
	// bool cycle list query
	CycleList bool `json:"cycle_list,omitempty"`
}

type ReportersResponse struct {
	Reporters []*Reporter `json:"reporters"`
}

type Reporter struct {
	Address  string          `json:"address"`
	Metadata *OracleReporter `json:"metadata"`
	Power    string          `json:"power"`
}

type OracleReporter struct {
	CommissionRate string `json:"commission_rate"`
	MinTokens      string `json:"min_tokens_required"`
	Jailed         bool   `protobuf:"varint,3,opt,name=jailed,proto3" json:"jailed,omitempty"`
	// jailed_until is the time the reporter is jailed until
	JailedUntil time.Time `protobuf:"bytes,4,opt,name=jailed_until,json=jailedUntil,proto3,stdtime" json:"jailed_until"`
	Moniker     string    `json:"moniker"`
}

type QuerySelectorReporterResponse struct {
	Reporter string `json:"reporter"`
}

type QueryDisputesTallyResponse struct {
	Users       *GroupTally          `protobuf:"bytes,1,opt,name=users,proto3" json:"users,omitempty"`
	Reporters   *GroupTally          `protobuf:"bytes,2,opt,name=reporters,proto3" json:"reporters,omitempty"`
	Team        *FormattedVoteCounts `protobuf:"bytes,3,opt,name=team,proto3" json:"team,omitempty"`
	ChoiceTotal *ChoiceTotal         `protobuf:"bytes,4,opt,name=choiceTotal,proto3" json:"choiceTotal,omitempty"`
}

type ChoiceTotal struct {
	Support string `protobuf:"varint,1,opt,name=support,proto3" json:"support,omitempty"`
	Against string `protobuf:"varint,2,opt,name=against,proto3" json:"against,omitempty"`
	Invalid string `protobuf:"varint,3,opt,name=invalid,proto3" json:"invalid,omitempty"`
}

type GroupTally struct {
	VoteCount       *FormattedVoteCounts `protobuf:"bytes,1,opt,name=voteCount,proto3" json:"voteCount,omitempty"`
	TotalPowerVoted string               `protobuf:"varint,2,opt,name=totalPowerVoted,proto3" json:"totalPowerVoted,omitempty"`
	TotalGroupPower string               `protobuf:"varint,3,opt,name=totalGroupPower,proto3" json:"totalGroupPower,omitempty"`
}

type FormattedVoteCounts struct {
	Support string `protobuf:"varint,1,opt,name=support,proto3" json:"support,omitempty"`
	Against string `protobuf:"varint,2,opt,name=against,proto3" json:"against,omitempty"`
	Invalid string `protobuf:"varint,3,opt,name=invalid,proto3" json:"invalid,omitempty"`
}

type QueryOpenDisputesResponse struct {
	OpenDisputes *OpenDisputes `protobuf:"bytes,1,opt,name=openDisputes,proto3" json:"openDisputes,omitempty"`
}

type OpenDisputes struct {
	Ids []string `protobuf:"varint,1,rep,packed,name=ids,proto3" json:"ids,omitempty"`
}

type QueryValidatorsResponse struct {
	Validators []Validator `json:"validators"`
}

type QueryMicroReportsResponse struct {
	MicroReports []MicroReport `protobuf:"bytes,1,rep,name=microReports,proto3" json:"microReports"`
}

type Validator struct {
	// operator_address defines the address of the validator's operator; bech encoded in JSON.
	OperatorAddress string `protobuf:"bytes,1,opt,name=operator_address,json=operatorAddress,proto3" json:"operator_address,omitempty"`
	// consensus_pubkey is the consensus public key of the validator, as a Protobuf Any.
	ConsensusPubkey *types1.Any `protobuf:"bytes,2,opt,name=consensus_pubkey,json=consensusPubkey,proto3" json:"consensus_pubkey,omitempty"`
	// jailed defined whether the validator has been jailed from bonded status or not.
	Jailed bool `protobuf:"varint,3,opt,name=jailed,proto3" json:"jailed,omitempty"`
	// status is the validator status (bonded/unbonding/unbonded).
	Status int `protobuf:"varint,4,opt,name=status,proto3,enum=cosmos.staking.v1beta1.BondStatus" json:"status,omitempty"`
	// tokens define the delegated tokens (incl. self-delegation).
	Tokens string `protobuf:"bytes,5,opt,name=tokens,proto3,customtype=cosmossdk.io/math.Int" json:"tokens"`
	// delegator_shares defines total shares issued to a validator's delegators.
	DelegatorShares string `protobuf:"bytes,6,opt,name=delegator_shares,json=delegatorShares,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"delegator_shares"`
	// description defines the description terms for the validator.
	Description Description `protobuf:"bytes,7,opt,name=description,proto3" json:"description"`
	// unbonding_height defines, if unbonding, the height at which this validator has begun unbonding.
	UnbondingHeight string `protobuf:"varint,8,opt,name=unbonding_height,json=unbondingHeight,proto3" json:"unbonding_height,omitempty"`
	// unbonding_time defines, if unbonding, the min time for the validator to complete unbonding.
	UnbondingTime string `protobuf:"bytes,9,opt,name=unbonding_time,json=unbondingTime,proto3,stdtime" json:"unbonding_time"`
	// commission defines the commission parameters.
	Commission Commission `protobuf:"bytes,10,opt,name=commission,proto3" json:"commission"`
	// min_self_delegation is the validator's self declared minimum self delegation.
	// Since: cosmos-sdk 0.46
	MinSelfDelegation string `protobuf:"bytes,11,opt,name=min_self_delegation,json=minSelfDelegation,proto3" json:"min_self_delegation"`
	// strictly positive if this validator's unbonding has been stopped by external modules
	UnbondingOnHoldRefCount string `protobuf:"varint,12,opt,name=unbonding_on_hold_ref_count,json=unbondingOnHoldRefCount,proto3" json:"unbonding_on_hold_ref_count,omitempty"`
	// list of unbonding ids, each uniquely identifing an unbonding of this validator
	UnbondingIds []string `protobuf:"varint,13,rep,packed,name=unbonding_ids,json=unbondingIds,proto3" json:"unbonding_ids,omitempty"`
}

type Description struct {
	// moniker defines a human-readable name for the validator.
	Moniker string `protobuf:"bytes,1,opt,name=moniker,proto3" json:"moniker,omitempty"`
	// identity defines an optional identity signature (ex. UPort or Keybase).
	Identity string `protobuf:"bytes,2,opt,name=identity,proto3" json:"identity,omitempty"`
	// website defines an optional website link.
	Website string `protobuf:"bytes,3,opt,name=website,proto3" json:"website,omitempty"`
	// security_contact defines an optional email for security contact.
	SecurityContact string `protobuf:"bytes,4,opt,name=security_contact,json=securityContact,proto3" json:"security_contact,omitempty"`
	// details define other optional details.
	Details string `protobuf:"bytes,5,opt,name=details,proto3" json:"details,omitempty"`
}

type Commission struct {
	// commission_rates defines the initial commission rates to be used for creating a validator.
	CommissionRates `protobuf:"bytes,1,opt,name=commission_rates,json=commissionRates,proto3,embedded=commission_rates" json:"commission_rates"`
	// update_time is the last time the commission rate was changed.
	UpdateTime time.Time `protobuf:"bytes,2,opt,name=update_time,json=updateTime,proto3,stdtime" json:"update_time"`
}

type CommissionRates struct {
	// rate is the commission rate charged to delegators, as a fraction.
	Rate math.LegacyDec `protobuf:"bytes,1,opt,name=rate,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"rate"`
	// max_rate defines the maximum commission rate which validator can ever charge, as a fraction.
	MaxRate math.LegacyDec `protobuf:"bytes,2,opt,name=max_rate,json=maxRate,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"max_rate"`
	// max_change_rate defines the maximum daily increase of the validator commission, as a fraction.
	MaxChangeRate math.LegacyDec `protobuf:"bytes,3,opt,name=max_change_rate,json=maxChangeRate,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"max_change_rate"`
}

type QueryCurrentCyclelistQueryResponse struct {
	QueryData string     `protobuf:"bytes,1,opt,name=query_data,json=queryData,proto3" json:"query_data,omitempty"`
	QueryMeta *QueryMeta `protobuf:"bytes,2,opt,name=query_meta,json=queryMeta,proto3" json:"query_meta,omitempty"`
}

type QueryGetCurrentAggregateReportResponse struct {
	// aggregate defines the current aggregate report.
	Aggregate *Aggregate `protobuf:"bytes,1,opt,name=aggregate,proto3" json:"aggregate,omitempty"`
	// timestamp defines the timestamp of the aggregate report.
	Timestamp string `protobuf:"varint,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

type Aggregate struct {
	// query_id is the id of the query
	QueryId []byte `protobuf:"bytes,1,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
	// aggregate_value is the value of the aggregate
	AggregateValue string `protobuf:"bytes,2,opt,name=aggregate_value,json=aggregateValue,proto3" json:"aggregate_value,omitempty"`
	// aggregate_reporter is the address of the reporter
	AggregateReporter string `protobuf:"bytes,3,opt,name=aggregate_reporter,json=aggregateReporter,proto3" json:"aggregate_reporter,omitempty"`
	// aggregate_power is the power of all the reporters
	// that reported for the aggregate
	AggregatePower string `protobuf:"varint,4,opt,name=aggregate_power,json=aggregatePower,proto3" json:"aggregate_power,omitempty"`
	// flagged is true if the aggregate was flagged by a dispute
	Flagged bool `protobuf:"varint,5,opt,name=flagged,proto3" json:"flagged,omitempty"`
	// index is the index of the aggregate
	Index string `protobuf:"varint,6,opt,name=index,proto3" json:"index,omitempty"`
	// height of the aggregate report
	Height string `protobuf:"varint,7,opt,name=height,proto3" json:"height,omitempty"`
	// height of the micro report
	MicroHeight string `protobuf:"varint,8,opt,name=micro_height,json=microHeight,proto3" json:"micro_height,omitempty"`
	// meta_id is the id of the querymeta iterator
	MetaId string `protobuf:"varint,9,opt,name=meta_id,json=metaId,proto3" json:"meta_id,omitempty"`
}

type QueryGetSnapshotsByReportResponse struct {
	Snapshots []string `protobuf:"bytes,1,rep,name=snapshots,proto3" json:"snapshots,omitempty"`
}

type QueryGetAttestationBySnapshotResponse struct {
	Attestations []string `protobuf:"bytes,1,rep,name=attestations,proto3" json:"attestations,omitempty"`
}

type QueryGetAttestationDataBySnapshotResponse struct {
	QueryId                 string `protobuf:"bytes,1,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
	Timestamp               string `protobuf:"bytes,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	AggregateValue          string `protobuf:"bytes,3,opt,name=aggregate_value,json=aggregateValue,proto3" json:"aggregate_value,omitempty"`
	AggregatePower          string `protobuf:"bytes,4,opt,name=aggregate_power,json=aggregatePower,proto3" json:"aggregate_power,omitempty"`
	Checkpoint              string `protobuf:"bytes,5,opt,name=checkpoint,proto3" json:"checkpoint,omitempty"`
	AttestationTimestamp    string `protobuf:"bytes,6,opt,name=attestation_timestamp,json=attestationTimestamp,proto3" json:"attestation_timestamp,omitempty"`
	PreviousReportTimestamp string `protobuf:"bytes,7,opt,name=previous_report_timestamp,json=previousReportTimestamp,proto3" json:"previous_report_timestamp,omitempty"`
	NextReportTimestamp     string `protobuf:"bytes,8,opt,name=next_report_timestamp,json=nextReportTimestamp,proto3" json:"next_report_timestamp,omitempty"`
	LastConsensusTimestamp  string `protobuf:"bytes,9,opt,name=last_consensus_timestamp,json=lastConsensusTimestamp,proto3" json:"last_consensus_timestamp,omitempty"`
}

type QueryRetrieveDataResponse struct {
	Aggregate *Aggregate `protobuf:"bytes,1,opt,name=aggregate,proto3" json:"aggregate,omitempty"`
}

type DataSpec struct {
	// ipfs hash of the data spec
	DocumentHash string `protobuf:"bytes,1,opt,name=document_hash,json=documentHash,proto3" json:"document_hash,omitempty"`
	// the value's datatype for decoding the value
	ResponseValueType string `protobuf:"bytes,2,opt,name=response_value_type,json=responseValueType,proto3" json:"response_value_type,omitempty"`
	// the abi components for decoding
	AbiComponents []*registrytypes.ABIComponent `protobuf:"bytes,3,rep,name=abi_components,json=abiComponents,proto3" json:"abi_components,omitempty"`
	// how to aggregate the data (ie. average, median, mode, etc) for aggregating reports and arriving at final value
	AggregationMethod string `protobuf:"bytes,4,opt,name=aggregation_method,json=aggregationMethod,proto3" json:"aggregation_method,omitempty"`
	// address that originally registered the data spec
	Registrar string `protobuf:"bytes,5,opt,name=registrar,proto3" json:"registrar,omitempty"`
	// report_buffer_window specifies the duration of the time window following an initial report
	// during which additional reports can be submitted. This duration acts as a buffer, allowing
	// a collection of related reports in a defined time frame. The window ensures that all
	// pertinent reports are aggregated together before arriving at a final value. This defaults
	// to 0s if not specified.
	// extensions: treat as a golang time.duration, don't allow nil values, don't omit empty values
	ReportBlockWindow uint64 `protobuf:"varint,6,opt,name=report_block_window,json=reportBlockWindow,proto3" json:"report_block_window,omitempty"`
	// querytype is the first arg in queryData
	QueryType string `protobuf:"bytes,7,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
}

type DataSpec2 struct {
	// ipfs hash of the data spec
	DocumentHash string `protobuf:"bytes,1,opt,name=document_hash,json=documentHash,proto3" json:"document_hash,omitempty"`
	// the value's datatype for decoding the value
	ResponseValueType string `protobuf:"bytes,2,opt,name=response_value_type,json=responseValueType,proto3" json:"response_value_type,omitempty"`
	// the abi components for decoding
	AbiComponents []*registrytypes.ABIComponent `protobuf:"bytes,3,rep,name=abi_components,json=abiComponents,proto3" json:"abi_components,omitempty"`
	// how to aggregate the data (ie. average, median, mode, etc) for aggregating reports and arriving at final value
	AggregationMethod string `protobuf:"bytes,4,opt,name=aggregation_method,json=aggregationMethod,proto3" json:"aggregation_method,omitempty"`
	// address that originally registered the data spec
	Registrar string `protobuf:"bytes,5,opt,name=registrar,proto3" json:"registrar,omitempty"`
	// report_buffer_window specifies the duration of the time window following an initial report
	// during which additional reports can be submitted. This duration acts as a buffer, allowing
	// a collection of related reports in a defined time frame. The window ensures that all
	// pertinent reports are aggregated together before arriving at a final value. This defaults
	// to 0s if not specified.
	// extensions: treat as a golang time.duration, don't allow nil values, don't omit empty values
	ReportBlockWindow string `protobuf:"varint,6,opt,name=report_block_window,json=reportBlockWindow,proto3" json:"report_block_window,omitempty"`
	// querytype is the first arg in queryData
	QueryType string `protobuf:"bytes,7,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
}
type QueryGenerateQuerydataResponse struct {
	// query_data is the generated query_data hex string.
	QueryData string `protobuf:"bytes,1,opt,name=query_data,json=queryData,proto3" json:"query_data,omitempty"`
}

// QueryTeamAddressResponse is response type for the Query/TeamAddress RPC method.
type QueryTeamAddressResponse struct {
	// teamAddress holds the team address.
	TeamAddress string `protobuf:"bytes,1,opt,name=team_address,json=teamAddress,proto3" json:"team_address,omitempty"`
}

type QueryTeamVoteResponse struct {
	// teamVote holds the team voter info for a dispute.
	TeamVote Voter `protobuf:"bytes,1,opt,name=team_vote,json=teamVote,proto3" json:"team_vote"`
}

type QueryGetTippedQueriesResponse struct {
	// querymeta but string query data
	Queries []*QueryMetaButString `protobuf:"bytes,1,rep,name=queries,proto3" json:"queries,omitempty"`
	// pagination defines the pagination in the response.
	Pagination *query.PageResponse `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

// QueryMetaButString is QueryMeta but with the query_data as a string for query display purposes
type QueryMetaButString struct {
	// unique id of the query that changes after query's lifecycle ends
	Id string `json:"id,omitempty"`
	// amount of tokens that was tipped
	Amount string `json:"amount"`
	// expiration time of the query
	Expiration string `json:"expiration,omitempty"`
	// timeframe of the query according to the data spec
	RegistrySpecBlockWindow string `json:"registry_spec_block_window,omitempty"`
	// indicates whether query has revealed reports
	HasRevealedReports bool `json:"has_revealed_reports,omitempty"`
	// query_data: decodable bytes to field of the data spec
	QueryData string `json:"query_data,omitempty"`
	// string identifier of the data spec
	QueryType string `json:"query_type,omitempty"`
	// bool cycle list query
	CycleList bool `json:"cycle_list,omitempty"`
}

type QueryGetDataSpecResponse struct {
	// spec is the data spec corresponding to the query type.
	Spec *DataSpec2 `protobuf:"bytes,1,opt,name=spec,proto3" json:"spec,omitempty"`
}

type Voter struct {
	Vote          disputetypes.VoteEnum `protobuf:"varint,1,opt,name=vote,proto3,enum=layer.dispute.VoteEnum" json:"vote,omitempty"`
	VoterPower    math.Int              `protobuf:"bytes,2,opt,name=voter_power,json=voterPower,proto3,customtype=cosmossdk.io/math.Int" json:"voter_power"`
	ReporterPower math.Int              `protobuf:"bytes,3,opt,name=reporter_power,json=reporterPower,proto3,customtype=cosmossdk.io/math.Int" json:"reporter_power"`
	RewardClaimed bool                  `protobuf:"varint,5,opt,name=reward_claimed,json=rewardClaimed,proto3" json:"reward_claimed,omitempty"`
}

// HELPERS FOR TESTING AGAINST THE CHAIN

func EncodeStringValue(value string) string {
	// Create a string ABI type
	stringABIType, _ := abi.NewType("string", "", nil)

	// Create the arguments and pack the string
	arguments := abi.Arguments{{Type: stringABIType}}
	encodedBytes, _ := arguments.Pack(value)

	// Convert to hex string
	encodedString := hex.EncodeToString(encodedBytes)
	return encodedString
}

type QueryGetDepositClaimedResponse struct {
	Claimed bool `protobuf:"varint,1,opt,name=claimed,proto3" json:"claimed,omitempty"`
}

type Validators struct {
	Addr    string
	ValAddr string
	Val     *cosmos.ChainNode
}

type QueryGetNoStakeReportsByQueryIdResponse struct {
	// no_stake_reports defines the no stake reports.
	NoStakeReports []*NoStakeMicroReportStrings `protobuf:"bytes,1,rep,name=no_stake_reports,json=noStakeReports,proto3" json:"no_stake_reports,omitempty"`
	// pagination defines the pagination in the response.
	Pagination *PageResponse `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

type QueryGetReportersNoStakeReportsResponse struct {
	// no_stake_reports defines the no stake reports.
	NoStakeReports []*NoStakeMicroReportStrings `protobuf:"bytes,1,rep,name=no_stake_reports,json=noStakeReports,proto3" json:"no_stake_reports,omitempty"`
	// pagination defines the pagination in the response.
	Pagination *PageResponse `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

type PageResponse struct {
	// next_key is the key to be passed to PageRequest.key to
	// query the next page most efficiently. It will be empty if
	// there are no more results.
	NextKey string `protobuf:"bytes,1,opt,name=next_key,json=nextKey,proto3" json:"next_key,omitempty"`
	// total is total number of results available if PageRequest.count_total
	// was set, its value is undefined otherwise
	Total string `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
}

type NoStakeMicroReportStrings struct {
	// reporter is the address of the reporter
	Reporter string `protobuf:"bytes,1,opt,name=reporter,proto3" json:"reporter,omitempty"`
	// hex string of the response value
	Value string `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
	// timestamp of when the report was created
	Timestamp string `protobuf:"varint,4,opt,name=timestamp,json=timestamp,proto3" json:"timestamp,omitempty"`
	// block number of when the report was created
	BlockNumber string `protobuf:"varint,5,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
}

func GetChainVals(ctx context.Context, chain *cosmos.CosmosChain) ([]Validators, error) {
	validators := make([]Validators, len(chain.Validators))
	for i := range chain.Validators {
		val := chain.Validators[i]
		valAddr, err := val.AccountKeyBech32(ctx, "validator")
		if err != nil {
			return nil, err
		}
		valvalAddr, err := val.KeyBech32(ctx, "validator", "val")
		if err != nil {
			return nil, err
		}
		fmt.Println("val", i, " Account Address: ", valAddr)
		fmt.Println("val", i, " Validator Address: ", valvalAddr)
		validators[i] = Validators{
			Addr:    valAddr,
			ValAddr: valvalAddr,
			Val:     val,
		}
	}
	return validators, nil
}

func CreateTestAccounts(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain, numAccounts int, fundAmt math.Int) ([]string, error) {
	t.Helper()
	users := make([]string, numAccounts)
	for i := range make([]struct{}, numAccounts) {
		keyname := fmt.Sprintf("user%d", i)
		user := interchaintest.GetAndFundTestUsers(t, ctx, keyname, fundAmt, chain)[0]
		users[i] = user.FormattedAddress()
	}
	return users, nil
}

func ExecProposal(ctx context.Context, keyName string, prop Proposal, tn *cosmos.ChainNode) (string, error) {
	content, err := json.Marshal(prop)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	proposalFilename := fmt.Sprintf("%x.json", hash)
	err = tn.WriteFile(ctx, content, proposalFilename)
	if err != nil {
		return "", fmt.Errorf("writing param change proposal: %w", err)
	}

	proposalPath := filepath.Join(tn.HomeDir(), proposalFilename)

	command := []string{
		"gov", "submit-proposal",
		proposalPath,
	}

	return tn.ExecTx(ctx, keyName, command...)
}

func TurnOnMinting(ctx context.Context, layer *cosmos.CosmosChain, validatorI *cosmos.ChainNode) error {
	prop := Proposal{
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
	_, err := ExecProposal(ctx, "validator", prop, validatorI)
	if err != nil {
		return err
	}

	for _, v := range layer.Validators {
		_, err = v.ExecTx(ctx, "validator", "gov", "vote", "1", "yes", "--gas", "1000000", "--fees", "1000000loya", "--keyring-dir", "/var/cosmos-chain/layer-1")
		if err != nil {
			return err
		}
	}

	return nil
}

func GetValAddresses(ctx context.Context, layer *cosmos.CosmosChain) (validators []*cosmos.ChainNode, valAccAddresses, valAddresses []string, err error) {
	for _, validator := range layer.Validators {
		valAccAddress, err := validator.AccountKeyBech32(ctx, "validator")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error getting validator account address: %w", err)
		}
		valAccAddresses = append(valAccAddresses, valAccAddress)

		valAddress, err := validator.KeyBech32(ctx, "validator", "val")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error getting validator address: %w", err)
		}
		valAddresses = append(valAddresses, valAddress)

		fmt.Printf("valAccAddress: %s\n", valAccAddress)
		fmt.Printf("valValAddress: %s\n", valAddress)
	}

	return layer.Validators, valAccAddresses, valAddresses, nil
}

func GetTxHashFromExec(stdout []byte) (string, error) {
	output := cosmos.CosmosTx{}
	err := json.Unmarshal(stdout, &output)
	if err != nil {
		panic("error unmarshalling stdout")
	}
	fmt.Println("RawLog: ", output.RawLog)
	if output.Code != 0 {
		return output.TxHash, fmt.Errorf("transaction failed with code %d: %s", output.Code, output.RawLog)
	}
	return output.TxHash, nil
}

func QueryTips(queryData string, ctx context.Context, validatorI *cosmos.ChainNode) (CurrentTipsResponse, error) {
	availableTips, _, err := validatorI.ExecQuery(ctx, "oracle", "get-current-tip", queryData)
	if err != nil {
		return CurrentTipsResponse{}, err
	}
	var currentTips CurrentTipsResponse
	err = json.Unmarshal(availableTips, &currentTips)
	if err != nil {
		return CurrentTipsResponse{}, err
	}
	return currentTips, nil
}

func DelegateToValidator(ctx context.Context, userKey string, validator *cosmos.ChainNode, valAddr string, amount math.Int) (string, error) {
	delegateAmt := sdk.NewCoin("loya", amount)
	txHash, err := validator.ExecTx(ctx, userKey, "staking", "delegate", valAddr, delegateAmt.String(), "--keyring-dir", validator.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	if err != nil {
		return "", err
	}
	return txHash, nil
}

func CreateReporter(ctx context.Context, accountAddr string, validator *cosmos.ChainNode, moniker string) (string, error) {
	commissRate := "0.01"
	minStakeAmt := "1000000"
	txHash, err := validator.ExecTx(ctx, accountAddr, "reporter", "create-reporter", commissRate, minStakeAmt, moniker, "--keyring-dir", validator.HomeDir())
	if err != nil {
		return "", err
	}
	return txHash, nil
}
