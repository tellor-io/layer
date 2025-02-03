package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cometbft/cometbft/libs/rand"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	"go.uber.org/zap/zaptest"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
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
			AdditionalStartArgs: []string{"--key-name", "validator", "--price-daemon-enabled=false"},
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

type DataSpec struct {
	DocumentHash      string                        `json:"document_hash,omitempty"`
	ResponseValueType string                        `json:"response_value_type,omitempty"`
	AbiComponents     []*registrytypes.ABIComponent `json:"abi_components,omitempty"`
	AggregationMethod string                        `json:"aggregation_method,omitempty"`
	Registrar         string                        `json:"registrar,omitempty"`
	ReportBlockWindow int                           `json:"report_block_window,omitempty"`
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
}

type OracleReporter struct {
	CommissionRate string `json:"commission_rate"`
	MinTokens      string `json:"min_tokens_required"`
}

type QuerySelectorReporterResponse struct {
	Reporter string `json:"reporter"`
}

type QueryDisputesTallyResponse struct {
	Users     *GroupTally `protobuf:"bytes,1,opt,name=users,proto3" json:"users,omitempty"`
	Reporters *GroupTally `protobuf:"bytes,2,opt,name=reporters,proto3" json:"reporters,omitempty"`
	Team      *VoteCounts `protobuf:"bytes,3,opt,name=team,proto3" json:"team,omitempty"`
}

type GroupTally struct {
	VoteCount       *VoteCounts `protobuf:"bytes,1,opt,name=voteCount,proto3" json:"voteCount,omitempty"`
	TotalPowerVoted string      `protobuf:"varint,2,opt,name=totalPowerVoted,proto3" json:"totalPowerVoted,omitempty"`
	TotalGroupPower string      `protobuf:"varint,3,opt,name=totalGroupPower,proto3" json:"totalGroupPower,omitempty"`
}

type VoteCounts struct {
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

// HELPERS FOR TESTING AGAINST THE CHAIN

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

func CreateDataSpec(reportBlockWindow int, registrar string) (DataSpec, error) {
	docHash := rand.Str(32)
	spec := DataSpec{
		DocumentHash:      docHash,
		ResponseValueType: "uint256",
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
		},
		AggregationMethod: "weighted-median",
		Registrar:         registrar,
		ReportBlockWindow: reportBlockWindow,
	}

	return spec, nil
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
