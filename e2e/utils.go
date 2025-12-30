package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	interchaintestutil "github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	"go.uber.org/zap/zaptest"

	"cosmossdk.io/math"

	types1 "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/query"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// ============================================================================
// CONSTANTS AND GLOBAL VARIABLES
// ============================================================================

var (
	layerImageInfo = []ibc.DockerImage{
		{
			Repository: "layer",
			Version:    "local",
			UIDGID:     "1025:1025",
		},
	}
	numVals      = 2
	numFullNodes = 2

	baseBech32 = "tellor"

	teamMnemonic = "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"

	DefaultGasPrice = "0.000025000000000000loya"
)

// ============================================================================
// CUSTOM TYPES FOR TESTS
// ============================================================================

// SetupConfig holds configuration for test setup
type SetupConfig struct {
	NumValidators   int
	NumFullNodes    int
	ModifyGenesis   []cosmos.GenesisKV
	GasPrices       string
	GlobalFeeMinGas string
}

// ValidatorInfo contains validator node and address information
type ValidatorInfo struct {
	Node    *cosmos.ChainNode
	AccAddr string
	ValAddr string
}

// Validators contains validator information for chain operations
type Validators struct {
	Addr    string
	ValAddr string
	Val     *cosmos.ChainNode
}

// ============================================================================
// DISPUTE AND VOTING TYPES
// ============================================================================

type Disputes struct {
	Disputes []struct {
		DisputeID string   `json:"disputeId"`
		Metadata  Metadata `json:"metadata"`
	} `json:"disputes"`
}

type Metadata struct {
	HashID            string   `json:"hash_id"`
	DisputeID         string   `json:"dispute_id"`
	DisputeCategory   string   `json:"dispute_category"`
	DisputeFee        string   `json:"dispute_fee"`
	DisputeStatus     string   `json:"dispute_status"`
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
	HashId             string     `protobuf:"bytes,1,opt,name=hash_id,json=hashId,proto3" json:"hash_id,omitempty"`
	DisputeId          string     `protobuf:"varint,2,opt,name=dispute_id,json=disputeId,proto3" json:"dispute_id,omitempty"`
	DisputeCategory    string     `protobuf:"varint,3,opt,name=dispute_category,json=disputeCategory,proto3,enum=layer.dispute.DisputeCategory" json:"dispute_category,omitempty"`
	DisputeFee         string     `protobuf:"bytes,4,opt,name=dispute_fee,json=disputeFee,proto3,customtype=cosmossdk.io/math.Int" json:"dispute_fee"`
	DisputeStatus      string     `protobuf:"varint,5,opt,name=dispute_status,json=disputeStatus,proto3,enum=layer.dispute.DisputeStatus" json:"dispute_status,omitempty"`
	DisputeStartTime   string     `protobuf:"bytes,6,opt,name=dispute_start_time,json=disputeStartTime,proto3,stdtime" json:"dispute_start_time"`
	DisputeEndTime     string     `protobuf:"bytes,7,opt,name=dispute_end_time,json=disputeEndTime,proto3,stdtime" json:"dispute_end_time"`
	DisputeStartBlock  string     `protobuf:"varint,8,opt,name=dispute_start_block,json=disputeStartBlock,proto3" json:"dispute_start_block,omitempty"`
	DisputeRound       string     `protobuf:"varint,9,opt,name=dispute_round,json=disputeRound,proto3" json:"dispute_round,omitempty"`
	SlashAmount        string     `protobuf:"bytes,10,opt,name=slash_amount,json=slashAmount,proto3,customtype=cosmossdk.io/math.Int" json:"slash_amount"`
	BurnAmount         string     `protobuf:"bytes,11,opt,name=burn_amount,json=burnAmount,proto3,customtype=cosmossdk.io/math.Int" json:"burn_amount"`
	InitialEvidence    Evidence   `protobuf:"bytes,12,opt,name=initial_evidence,json=initialEvidence,proto3" json:"initial_evidence"`
	FeeTotal           string     `protobuf:"bytes,13,opt,name=fee_total,json=feeTotal,proto3,customtype=cosmossdk.io/math.Int" json:"fee_total"`
	PrevDisputeIds     []string   `protobuf:"varint,14,rep,packed,name=prev_dispute_ids,json=prevDisputeIds,proto3" json:"prev_dispute_ids,omitempty"`
	BlockNumber        string     `protobuf:"varint,15,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	Open               bool       `protobuf:"varint,16,opt,name=open,proto3" json:"open,omitempty"`
	AdditionalEvidence []Evidence `protobuf:"bytes,17,rep,name=additional_evidence,json=additionalEvidence,proto3" json:"additional_evidence,omitempty"`
	VoterReward        string     `protobuf:"bytes,18,opt,name=voter_reward,json=voterReward,proto3,customtype=cosmossdk.io/math.Int" json:"voter_reward"`
	PendingExecution   bool       `protobuf:"varint,19,opt,name=pending_execution,json=pendingExecution,proto3" json:"pending_execution,omitempty"`
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

type Voter struct {
	Vote          string `protobuf:"varint,1,opt,name=vote,proto3,enum=layer.dispute.VoteEnum" json:"vote,omitempty"`
	VoterPower    string `protobuf:"bytes,2,opt,name=voter_power,json=voterPower,proto3,customtype=cosmossdk.io/math.Int" json:"voter_power"`
	ReporterPower string `protobuf:"bytes,3,opt,name=reporter_power,json=reporterPower,proto3,customtype=cosmossdk.io/math.Int" json:"reporter_power"`
	RewardClaimed bool   `protobuf:"varint,5,opt,name=reward_claimed,json=rewardClaimed,proto3" json:"reward_claimed,omitempty"`
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

// ============================================================================
// ORACLE AND REPORTING TYPES
// ============================================================================

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

type Aggregate struct {
	QueryId           string `protobuf:"bytes,1,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
	AggregateValue    string `protobuf:"bytes,2,opt,name=aggregate_value,json=aggregateValue,proto3" json:"aggregate_value,omitempty"`
	AggregateReporter string `protobuf:"bytes,3,opt,name=aggregate_reporter,json=aggregateReporter,proto3" json:"aggregate_reporter,omitempty"`
	AggregatePower    string `protobuf:"varint,4,opt,name=aggregate_power,json=aggregatePower,proto3" json:"aggregate_power,omitempty"`
	Flagged           bool   `protobuf:"varint,5,opt,name=flagged,proto3" json:"flagged,omitempty"`
	Index             string `protobuf:"varint,6,opt,name=index,proto3" json:"index,omitempty"`
	Height            string `protobuf:"varint,7,opt,name=height,proto3" json:"height,omitempty"`
	MicroHeight       string `protobuf:"varint,8,opt,name=micro_height,json=microHeight,proto3" json:"micro_height,omitempty"`
	MetaId            string `protobuf:"varint,9,opt,name=meta_id,json=metaId,proto3" json:"meta_id,omitempty"`
}

type Reporter struct {
	Address  string          `json:"address"`
	Metadata *OracleReporter `json:"metadata"`
	Power    string          `json:"power"`
}

type OracleReporter struct {
	CommissionRate string    `json:"commission_rate"`
	MinTokens      string    `json:"min_tokens_required"`
	Jailed         bool      `protobuf:"varint,3,opt,name=jailed,proto3" json:"jailed,omitempty"`
	JailedUntil    time.Time `protobuf:"bytes,4,opt,name=jailed_until,json=jailedUntil,proto3,stdtime" json:"jailed_until"`
	Moniker        string    `json:"moniker"`
}

type NoStakeMicroReportStrings struct {
	Reporter    string `protobuf:"bytes,1,opt,name=reporter,proto3" json:"reporter,omitempty"`
	Value       string `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp   string `protobuf:"varint,4,opt,name=timestamp,json=timestamp,proto3" json:"timestamp,omitempty"`
	BlockNumber string `protobuf:"varint,5,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
}

// ============================================================================
// REGISTRY TYPES
// ============================================================================

type QueryMeta struct {
	Id                      string `json:"id,omitempty"`
	Amount                  string `json:"amount"`
	Expiration              string `json:"expiration,omitempty"`
	RegistrySpecBlockWindow string `json:"registry_spec_block_window,omitempty"`
	HasRevealedReports      bool   `json:"has_revealed_reports,omitempty"`
	QueryData               string `json:"query_data,omitempty"`
	QueryType               string `json:"query_type,omitempty"`
	CycleList               bool   `json:"cycle_list,omitempty"`
}

type QueryMetaButString struct {
	Id                      string `json:"id,omitempty"`
	Amount                  string `json:"amount"`
	Expiration              string `json:"expiration,omitempty"`
	RegistrySpecBlockWindow string `json:"registry_spec_block_window,omitempty"`
	HasRevealedReports      bool   `json:"has_revealed_reports,omitempty"`
	QueryData               string `json:"query_data,omitempty"`
	QueryType               string `json:"query_type,omitempty"`
	CycleList               bool   `json:"cycle_list,omitempty"`
}

type DataSpec struct {
	DocumentHash      string                        `protobuf:"bytes,1,opt,name=document_hash,json=documentHash,proto3" json:"document_hash,omitempty"`
	ResponseValueType string                        `protobuf:"bytes,2,opt,name=response_value_type,json=responseValueType,proto3" json:"response_value_type,omitempty"`
	AbiComponents     []*registrytypes.ABIComponent `protobuf:"bytes,3,rep,name=abi_components,json=abiComponents,proto3" json:"abi_components,omitempty"`
	AggregationMethod string                        `protobuf:"bytes,4,opt,name=aggregation_method,json=aggregationMethod,proto3" json:"aggregation_method,omitempty"`
	Registrar         string                        `protobuf:"bytes,5,opt,name=registrar,proto3" json:"registrar,omitempty"`
	ReportBlockWindow string                        `protobuf:"varint,6,opt,name=report_block_window,json=reportBlockWindow,proto3" json:"report_block_window,omitempty"`
	QueryType         string                        `protobuf:"bytes,7,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
}

type DataSpec2 struct {
	DocumentHash      string                        `protobuf:"bytes,1,opt,name=document_hash,json=documentHash,proto3" json:"document_hash,omitempty"`
	ResponseValueType string                        `protobuf:"bytes,2,opt,name=response_value_type,json=responseValueType,proto3" json:"response_value_type,omitempty"`
	AbiComponents     []*registrytypes.ABIComponent `protobuf:"bytes,3,rep,name=abi_components,json=abiComponents,proto3" json:"abi_components,omitempty"`
	AggregationMethod string                        `protobuf:"bytes,4,opt,name=aggregation_method,json=aggregationMethod,proto3" json:"aggregation_method,omitempty"`
	Registrar         string                        `protobuf:"bytes,5,opt,name=registrar,proto3" json:"registrar,omitempty"`
	ReportBlockWindow string                        `protobuf:"varint,6,opt,name=report_block_window,json=reportBlockWindow,proto3" json:"report_block_window,omitempty"`
	QueryType         string                        `protobuf:"bytes,7,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
}

type DataSpecResponse struct {
	DocumentHash      string                        `json:"document_hash,omitempty"`
	ResponseValueType string                        `json:"response_value_type,omitempty"`
	AbiComponents     []*registrytypes.ABIComponent `json:"abi_components,omitempty"`
	AggregationMethod string                        `json:"aggregation_method,omitempty"`
	Registrar         string                        `json:"registrar,omitempty"`
	ReportBlockWindow string                        `json:"report_block_window,omitempty"`
}

// ============================================================================
// GOVERNANCE AND PROPOSAL TYPES
// ============================================================================

type Proposal struct {
	Messages  []map[string]interface{} `json:"messages"`
	Metadata  string                   `json:"metadata"`
	Deposit   string                   `json:"deposit"`
	Title     string                   `json:"title"`
	Summary   string                   `json:"summary"`
	Expedited bool                     `json:"expedited"`
}

// ============================================================================
// QUERY RESPONSES
// ============================================================================

type CurrentTipsResponse struct {
	Tips string `json:"tips"`
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
		Total string `json:"total"`
	} `json:"pagination"`
}

type QueryReportersResponse struct {
	Reporters  []*Reporter `protobuf:"bytes,1,rep,name=reporters,proto3" json:"reporters,omitempty"`
	Pagination struct {
		Total string `json:"total"`
	} `json:"pagination"`
}

type ReportersResponse struct {
	Reporters []*Reporter `json:"reporters"`
}

type QuerySelectorReporterResponse struct {
	Reporter string `json:"reporter"`
}

type QueryValidatorsResponse struct {
	Validators []Validator `json:"validators"`
}

type QueryMicroReportsResponse struct {
	MicroReports []MicroReport `protobuf:"bytes,1,rep,name=microReports,proto3" json:"microReports"`
}

type Validator struct {
	OperatorAddress         string      `protobuf:"bytes,1,opt,name=operator_address,json=operatorAddress,proto3" json:"operator_address,omitempty"`
	ConsensusPubkey         *types1.Any `protobuf:"bytes,2,opt,name=consensus_pubkey,json=consensusPubkey,proto3" json:"consensus_pubkey,omitempty"`
	Jailed                  bool        `protobuf:"varint,3,opt,name=jailed,proto3" json:"jailed,omitempty"`
	Status                  string      `protobuf:"varint,4,opt,name=status,proto3,enum=cosmos.staking.v1beta1.BondStatus" json:"status,omitempty"`
	Tokens                  string      `protobuf:"bytes,5,opt,name=tokens,proto3,customtype=cosmossdk.io/math.Int" json:"tokens"`
	DelegatorShares         string      `protobuf:"bytes,6,opt,name=delegator_shares,json=delegatorShares,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"delegator_shares"`
	Description             Description `protobuf:"bytes,7,opt,name=description,proto3" json:"description"`
	UnbondingHeight         string      `protobuf:"varint,8,opt,name=unbonding_height,json=unbondingHeight,proto3" json:"unbonding_height,omitempty"`
	UnbondingTime           string      `protobuf:"bytes,9,opt,name=unbonding_time,json=unbondingTime,proto3,stdtime" json:"unbonding_time"`
	Commission              Commission  `protobuf:"bytes,10,opt,name=commission,proto3" json:"commission"`
	MinSelfDelegation       string      `protobuf:"bytes,11,opt,name=min_self_delegation,json=minSelfDelegation,proto3" json:"min_self_delegation"`
	UnbondingOnHoldRefCount string      `protobuf:"varint,12,opt,name=unbonding_on_hold_ref_count,json=unbondingOnHoldRefCount,proto3" json:"unbonding_on_hold_ref_count,omitempty"`
	UnbondingIds            []string    `protobuf:"varint,13,rep,packed,name=unbonding_ids,json=unbondingIds,proto3" json:"unbonding_ids,omitempty"`
}

type Description struct {
	Moniker         string `protobuf:"bytes,1,opt,name=moniker,proto3" json:"moniker,omitempty"`
	Identity        string `protobuf:"bytes,2,opt,name=identity,proto3" json:"identity,omitempty"`
	Website         string `protobuf:"bytes,3,opt,name=website,proto3" json:"website,omitempty"`
	SecurityContact string `protobuf:"bytes,4,opt,name=security_contact,json=securityContact,proto3" json:"security_contact,omitempty"`
	Details         string `protobuf:"bytes,5,opt,name=details,proto3" json:"details,omitempty"`
}

type Commission struct {
	CommissionRates CommissionRates `protobuf:"bytes,1,opt,name=commission_rates,json=commissionRates,proto3,embedded=commission_rates" json:"commission_rates"`
	UpdateTime      time.Time       `protobuf:"bytes,2,opt,name=update_time,json=updateTime,proto3,stdtime" json:"update_time"`
}

type CommissionRates struct {
	Rate          string `protobuf:"bytes,1,opt,name=rate,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"rate"`
	MaxRate       string `protobuf:"bytes,2,opt,name=max_rate,json=maxRate,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"max_rate"`
	MaxChangeRate string `protobuf:"bytes,3,opt,name=max_change_rate,json=maxChangeRate,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"max_change_rate"`
}

type QueryCurrentCyclelistQueryResponse struct {
	QueryData string     `protobuf:"bytes,1,opt,name=query_data,json=queryData,proto3" json:"query_data,omitempty"`
	QueryMeta *QueryMeta `protobuf:"bytes,2,opt,name=query_meta,json=queryMeta,proto3" json:"query_meta,omitempty"`
}

type QueryGetCurrentAggregateReportResponse struct {
	Aggregate *Aggregate `protobuf:"bytes,1,opt,name=aggregate,proto3" json:"aggregate,omitempty"`
	Timestamp string     `protobuf:"varint,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
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

type QueryGenerateQuerydataResponse struct {
	QueryData string `protobuf:"bytes,1,opt,name=query_data,json=queryData,proto3" json:"query_data,omitempty"`
}

type QueryTeamAddressResponse struct {
	TeamAddress string `protobuf:"bytes,1,opt,name=team_address,json=teamAddress,proto3" json:"team_address,omitempty"`
}

type QueryTeamVoteResponse struct {
	TeamVote Voter `protobuf:"bytes,1,opt,name=team_vote,json=teamVote,proto3" json:"team_vote"`
}

type QueryGetTippedQueriesResponse struct {
	Queries    []*QueryMetaButString `protobuf:"bytes,1,rep,name=queries,proto3" json:"queries,omitempty"`
	Pagination *query.PageResponse   `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

type QueryGetDataSpecResponse struct {
	Spec *DataSpec2 `protobuf:"bytes,1,opt,name=spec,proto3" json:"spec,omitempty"`
}

type QueryGetDepositClaimedResponse struct {
	Claimed bool `protobuf:"varint,1,opt,name=claimed,proto3" json:"claimed,omitempty"`
}

type QueryGetNoStakeReportsByQueryIdResponse struct {
	NoStakeReports []*NoStakeMicroReportStrings `protobuf:"bytes,1,rep,name=no_stake_reports,json=noStakeReports,proto3" json:"no_stake_reports,omitempty"`
	Pagination     *PageResponse                `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

type QueryGetReportersNoStakeReportsResponse struct {
	NoStakeReports []*NoStakeMicroReportStrings `protobuf:"bytes,1,rep,name=no_stake_reports,json=noStakeReports,proto3" json:"no_stake_reports,omitempty"`
	Pagination     *PageResponse                `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

type PageResponse struct {
	NextKey string `protobuf:"bytes,1,opt,name=next_key,json=nextKey,proto3" json:"next_key,omitempty"`
	Total   string `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
}

// ============================================================================
// CHAIN SETUP AND CONFIGURATION
// ============================================================================

// DefaultSetupConfig returns standard test configuration
func DefaultSetupConfig() SetupConfig {
	fmt.Println("Using DefaultSetupConfig...")
	return SetupConfig{
		NumValidators:   2,
		NumFullNodes:    0,
		ModifyGenesis:   CreateStandardGenesis(),
		GasPrices:       DefaultGasPrice,
		GlobalFeeMinGas: "0.000025000000000000",
	}
}

// CreateStandardGenesis creates a standard genesis configuration
func CreateStandardGenesis() []cosmos.GenesisKV {
	teamAddressBytes := sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()

	return []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", teamAddressBytes),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.000025000000000000"),
	}
}

// ============================================================================
// VALIDATOR AND ACCOUNT OPERATIONS
// ============================================================================

// GetValidators retrieves all validators with their addresses
func GetValidators(ctx context.Context, chain *cosmos.CosmosChain) ([]ValidatorInfo, error) {
	var validators []ValidatorInfo

	for _, validator := range chain.Validators {
		accAddr, err := validator.AccountKeyBech32(ctx, "validator")
		if err != nil {
			return nil, fmt.Errorf("error getting validator account address: %w", err)
		}

		valAddr, err := validator.KeyBech32(ctx, "validator", "val")
		if err != nil {
			return nil, fmt.Errorf("error getting validator address: %w", err)
		}

		validators = append(validators, ValidatorInfo{
			Node:    validator,
			AccAddr: accAddr,
			ValAddr: valAddr,
		})
	}

	return validators, nil
}

// SetupTestChainWithConfig creates a test chain with the given configuration
func SetupChainWithCustomConfig(t *testing.T, config SetupConfig) (*cosmos.CosmosChain, *interchaintest.Interchain, context.Context) {
	t.Helper()
	fmt.Println("Setting up chain with custom config...")
	require := require.New(t)

	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	time.Sleep(1 * time.Second)

	// Use the genesis configuration from config, or default if empty
	modifyGenesis := config.ModifyGenesis
	if modifyGenesis == nil {
		modifyGenesis = CreateStandardGenesis()
	}

	fmt.Println("Creating chain spec...")
	// Create chain spec
	chainSpec := &interchaintest.ChainSpec{
		NumValidators: &config.NumValidators,
		NumFullNodes:  &config.NumFullNodes,
		ChainConfig: ibc.ChainConfig{
			Type:           "cosmos",
			Name:           "layer",
			ChainID:        "layer",
			Bin:            "layerd",
			Denom:          "loya",
			Bech32Prefix:   "tellor",
			CoinType:       "118",
			GasPrices:      config.GasPrices,
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
			EncodingConfig:      LayerEncoding(),
			ModifyGenesis:       cosmos.ModifyGenesis(modifyGenesis),
			AdditionalStartArgs: []string{"--key-name", "validator"},
		},
	}

	// Create chains
	fmt.Println("Creating chains...")
	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{chainSpec})

	fmt.Println("Creating client and network...")
	client, network := interchaintest.DockerSetup(t)
	time.Sleep(1 * time.Second)

	layer := chains[0].(*cosmos.CosmosChain)
	ic := interchaintest.NewInterchain().AddChain(layer)

	ctx := context.Background()
	fmt.Println("Building chain...")
	require.NoError(ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))
	time.Sleep(1 * time.Second)

	t.Cleanup(func() {
		_ = ic.Close()
		time.Sleep(1 * time.Second)
	})

	require.NoError(layer.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(layer.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	return layer, ic, ctx
}

// SetupStandardTestChain creates a test chain with standard configuration
func SetupChain(t *testing.T, numVals, numFullNodes int) (*cosmos.CosmosChain, *interchaintest.Interchain, context.Context) {
	t.Helper()
	fmt.Println("Setting up chain with standard configuration...")
	config := DefaultSetupConfig()
	config.NumValidators = numVals
	config.NumFullNodes = numFullNodes
	return SetupChainWithCustomConfig(t, config)
}

// PrintValidatorInfo prints validator information for debugging
func PrintValidatorInfo(ctx context.Context, validators []ValidatorInfo) {
	for i, validator := range validators {
		fmt.Printf("Validator %d:\n", i+1)
		fmt.Printf("  Account Address: %s\n", validator.AccAddr)
		fmt.Printf("  Validator Address: %s\n", validator.ValAddr)
		fmt.Printf("  Node: %s\n", validator.Node.Name())
	}
}

// CreateReporterFromValidator creates a reporter from a validator with stake
func CreateReporterFromValidator(ctx context.Context, validator ValidatorInfo, reporterName string, stakeAmount math.Int) (string, error) {
	txHash, err := validator.Node.ExecTx(ctx, "validator", "reporter", "create-reporter",
		"0.1", stakeAmount.String(), reporterName, "--keyring-dir", validator.Node.HomeDir())
	return txHash, err
}

// ============================================================================
// ORACLE AND REPORTER OPERATIONS
// ============================================================================

// TipQuery tips a query with the specified amount
func TipQuery(ctx context.Context, validator *cosmos.ChainNode, queryData string, tipCoin sdk.Coin) (string, error) {
	cmd := validator.TxCommand("validator", "oracle", "tip", queryData, tipCoin.String(), "--keyring-dir", validator.HomeDir())
	stdout, _, err := validator.Exec(ctx, cmd, validator.Chain.Config().Env)
	if err != nil {
		return "", err
	}

	// Parse the transaction output to get the tx hash
	var output cosmos.CosmosTx
	err = json.Unmarshal(stdout, &output)
	if err != nil {
		return "", err
	}

	return output.TxHash, nil
}

// SubmitBatchReport submits a batch of reports
func SubmitBatchReport(ctx context.Context, validator *cosmos.ChainNode, reports []string, fees string) (string, error) {
	args := []string{"oracle", "batch-submit-value"}
	for _, report := range reports {
		args = append(args, "--values", report)
	}
	args = append(args, "--gas", "1000000", "--fees", fees, "--keyring-dir", validator.HomeDir())

	stdout, _, err := validator.Exec(ctx, validator.TxCommand("validator", args...), validator.Chain.Config().Env)
	if err != nil {
		return "", err
	}

	txHash, err := GetTxHashFromExec(stdout)
	if err != nil {
		return "", err
	}

	return txHash, nil
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// Retry executes a function with retry logic to handle Docker container cleanup race conditions
func Retry(t *testing.T, testName string, operation func() error) error {
	t.Helper()
	maxRetries := 3
	delay := 5 * time.Second
	var lastErr error

	for i := range maxRetries {
		if i > 0 {
			t.Logf("[%s] Retry attempt %d/%d due to previous failure: %v",
				testName, i+1, maxRetries, lastErr)
			time.Sleep(delay)
		}

		err := operation()
		if err == nil {
			return nil // Success!
		}

		lastErr = err
		t.Logf("[%s] Attempt %d failed: %v", testName, i+1, err)
	}

	return fmt.Errorf("[%s] Operation failed after %d attempts. Last error: %w",
		testName, maxRetries, lastErr)
}

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

	var ic *interchaintest.Interchain

	ctx := context.Background()

	// Set Docker daemon configuration for better container management
	os.Setenv("DOCKER_BUILDKIT", "0") // Disable BuildKit for more stable behavior

	client, network := interchaintest.DockerSetup(t)

	// Use retry logic for the interchain build to handle container cleanup issues
	err = Retry(t, "LayerSpinup", func() error {
		ic = interchaintest.NewInterchain().AddChain(layer)
		return ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
			TestName:         t.Name(),
			Client:           client,
			NetworkID:        network,
			SkipPathCreation: true,
		})
	})
	require.NoError(t, err)

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
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
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
			GasPrices:           DefaultGasPrice,
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

// ============================================================================
// BRIDGE AND SECRETS OPERATIONS
// ============================================================================

// WriteSecretsFile adds the secrets file required for bridging
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

// HELPERS FOR TESTING AGAINST THE CHAIN

// ============================================================================
// TRANSACTION AND ENCODING UTILITIES
// ============================================================================

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

// EncodeOracleAttestationData encodes oracle attestation data for bridge operations
// This must match keeper.EncodeOracleAttestationData exactly
func EncodeOracleAttestationData(
	queryId []byte,
	value string,
	timestamp uint64,
	aggregatePower uint64,
	previousTimestamp uint64,
	nextTimestamp uint64,
	checkpoint []byte,
	attestationTimestamp uint64,
	lastConsensusTimestamp uint64,
) ([]byte, error) {
	// domainSeparator is bytes "tellorCurrentAttestation"
	NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR := []byte("tellorCurrentAttestation")
	// convert domain separator to bytes32
	var domainSepBytes32 [32]byte
	copy(domainSepBytes32[:], NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR)

	// convert queryId to bytes32
	var queryIdBytes32 [32]byte
	copy(queryIdBytes32[:], queryId)

	// convert value to bytes
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}

	// convert timestamps and power to big.Int
	timestampBig := new(big.Int).SetUint64(timestamp)
	aggregatePowerBig := new(big.Int).SetUint64(aggregatePower)
	previousTimestampBig := new(big.Int).SetUint64(previousTimestamp)
	nextTimestampBig := new(big.Int).SetUint64(nextTimestamp)
	attestationTimestampBig := new(big.Int).SetUint64(attestationTimestamp)
	lastConsensusTimestampBig := new(big.Int).SetUint64(lastConsensusTimestamp)

	// convert checkpoint to bytes32
	var checkpointBytes32 [32]byte
	copy(checkpointBytes32[:], checkpoint)

	// prepare ABI encoding types
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	arguments := abi.Arguments{
		{Type: bytes32Type}, // domain separator
		{Type: bytes32Type}, // queryId
		{Type: bytesType},   // value
		{Type: uint256Type}, // timestamp
		{Type: uint256Type}, // aggregatePower
		{Type: uint256Type}, // previousTimestamp
		{Type: uint256Type}, // nextTimestamp
		{Type: bytes32Type}, // checkpoint
		{Type: uint256Type}, // attestationTimestamp
		{Type: uint256Type}, // lastConsensusTimestamp
	}

	encodedData, err := arguments.Pack(
		domainSepBytes32,
		queryIdBytes32,
		valueBytes,
		timestampBig,
		aggregatePowerBig,
		previousTimestampBig,
		nextTimestampBig,
		checkpointBytes32,
		attestationTimestampBig,
		lastConsensusTimestampBig,
	)
	if err != nil {
		return nil, err
	}

	return crypto.Keccak256(encodedData), nil
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

// ============================================================================
// GOVERNANCE OPERATIONS
// ============================================================================

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
	fmt.Println("Turning on minting...")
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
		_, err = v.ExecTx(ctx, "validator", "gov", "vote", "1", "yes", "--gas", "1000000", "--fees", "500loya", "--keyring-dir", layer.HomeDir())
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

// ============================================================================
// QUERY HELPERS
// ============================================================================

// QueryWithTimeout executes a query with a 5-second timeout
func QueryWithTimeout(ctx context.Context, validatorI *cosmos.ChainNode, args ...string) ([]byte, []byte, error) {
	queryCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()
	return validatorI.ExecQuery(queryCtx, args...)
}

func QueryTips(queryData string, ctx context.Context, validatorI *cosmos.ChainNode) (CurrentTipsResponse, error) {
	availableTips, _, err := QueryWithTimeout(ctx, validatorI, "oracle", "get-current-tip", queryData)
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
	txHash, err := validator.ExecTx(ctx, userKey, "staking", "delegate", valAddr, delegateAmt.String(), "--keyring-dir", validator.HomeDir(), "--gas", "500000", "--fees", "50loya")
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

// SubmitCycleList queries the current cycle list and submits a value for it
// Returns the tx hash if successful, retries up to 3 times total
// Does NOT wait for blocks - use SubmitCycleListSafe for safer timing
func SubmitCycleList(ctx context.Context, node *cosmos.ChainNode, keyName, value, fees string) (string, error) {
	maxRetries := 2
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("SubmitCycleList attempt %d/%d\n", attempt, maxRetries)
		// Query current cycle list
		currentCycleListRes, _, err := QueryWithTimeout(ctx, node, "oracle", "current-cyclelist-query")
		if err != nil {
			lastErr = fmt.Errorf("failed to query current cycle list: %w", err)
			fmt.Printf("Attempt %d failed: %v\n", attempt, lastErr)
			continue
		}
		var currentCycleList QueryCurrentCyclelistQueryResponse
		err = json.Unmarshal(currentCycleListRes, &currentCycleList)
		if err != nil {
			lastErr = fmt.Errorf("failed to unmarshal cycle list response: %w", err)
			fmt.Printf("Attempt %d failed: %v\n", attempt, lastErr)
			continue
		}

		// Submit value
		txHashBytes, _, err := node.Exec(ctx,
			node.TxCommand(keyName, "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", fees, "--keyring-dir", node.HomeDir()),
			node.Chain.Config().Env)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute submit-value transaction: %w", err)
			fmt.Printf("Attempt %d failed: %v\n", attempt, lastErr)
			continue
		}

		// Parse the tx hash and check if transaction succeeded
		txHash, err := GetTxHashFromExec(txHashBytes)
		if err != nil {
			lastErr = fmt.Errorf("transaction failed: %w", err)
			fmt.Printf("Attempt %d transaction failed: %v\n", attempt, lastErr)
			continue
		}

		// Success!
		fmt.Printf("SubmitCycleList succeeded on attempt %d, tx hash: %s\n", attempt, txHash)
		return txHash, nil
	}

	return "", fmt.Errorf("SubmitCycleList failed after %d attempts. Last error: %w", maxRetries, lastErr)
}

// SubmitCycleListSafe queries the current cycle list and submits a value for it
// Waits 1 block after submission for better timing guarantees
// Returns the tx hash if successful, retries up to 3 times total
func SubmitCycleListSafe(ctx context.Context, node *cosmos.ChainNode, keyName, value, fees string) (string, error) {
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("SubmitCycleListSafe attempt %d/%d\n", attempt, maxRetries)
		// Query current cycle list
		currentCycleListRes, _, err := QueryWithTimeout(ctx, node, "oracle", "current-cyclelist-query")
		if err != nil {
			lastErr = fmt.Errorf("failed to query current cycle list: %w", err)
			fmt.Printf("Attempt %d failed: %v\n", attempt, lastErr)
			continue
		}
		var currentCycleList QueryCurrentCyclelistQueryResponse
		err = json.Unmarshal(currentCycleListRes, &currentCycleList)
		if err != nil {
			lastErr = fmt.Errorf("failed to unmarshal cycle list response: %w", err)
			fmt.Printf("Attempt %d failed: %v\n", attempt, lastErr)
			continue
		}

		// Submit value
		txHashBytes, _, err := node.Exec(ctx,
			node.TxCommand(keyName, "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", fees, "--keyring-dir", node.HomeDir()),
			node.Chain.Config().Env)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute submit-value transaction: %w", err)
			fmt.Printf("Attempt %d failed: %v\n", attempt, lastErr)
			continue
		}

		// Wait 1 block
		err = interchaintestutil.WaitForBlocks(ctx, 1, node)
		if err != nil {
			lastErr = fmt.Errorf("failed to wait for block: %w", err)
			fmt.Printf("Attempt %d failed: %v\n", attempt, lastErr)
			continue
		}

		// Parse the tx hash and check if transaction succeeded
		txHash, err := GetTxHashFromExec(txHashBytes)
		if err != nil {
			lastErr = fmt.Errorf("transaction failed: %w", err)
			fmt.Printf("Attempt %d transaction failed: %v\n", attempt, lastErr)
			continue
		}

		// Success!
		fmt.Printf("SubmitCycleListSafe succeeded on attempt %d, tx hash: %s\n", attempt, txHash)
		return txHash, nil
	}

	return "", fmt.Errorf("SubmitCycleListSafe failed after %d attempts. Last error: %w", maxRetries, lastErr)
}
