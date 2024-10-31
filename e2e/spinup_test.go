package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	"github.com/tellor-io/layer/utils"
)

const (
	qData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value = "000000000000000000000000000000000000000000000058528649cf80ee0000"
)

type MicroReport struct {
	Reporter        string `json:"reporter"`
	Power           string `json:"power"`
	QueryType       string `json:"query_type"`
	QueryID         string `json:"query_id"`
	AggregateMethod string `json:"aggregate_method"`
	Value           string `json:"value"`
	Timestamp       string `json:"timestamp"`
	BlockNumber     string `json:"block_number"`
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

func TestLearn(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	layer := e2e.LayerSpinup(t) // *cosmos.CosmosChain type
	validatorI := layer.Validators[0]

	// ctx := context.Background()
	valAddress, err := validatorI.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)
	user := interchaintest.GetAndFundTestUsers(t, ctx, "user1", math.OneInt(), layer)[0]
	fmt.Println("User address: ", user.FormattedAddress())
	// create reporter
	txHash, err := validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)

	_, _, err = validatorI.Exec(ctx, validatorI.TxCommand("validator", "oracle", "tip", valAddress, qData, "1000000loya", "--keyring-dir", "/var/cosmos-chain/layer-1"), validatorI.Chain.Config().Env)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, validatorI)
	require.NoError(t, err)

	txHash, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", valAddress, qData, value, "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)

	res1, _, err := validatorI.ExecQuery(ctx, "oracle", "get-reportsby-reporter", valAddress)
	require.NoError(t, err)

	var microReports ReportsResponse
	err = json.Unmarshal(res1, &microReports)
	require.NoError(t, err)
	fmt.Println("Micro reports: ", microReports)
	require.Equal(t, microReports.MicroReports[0].Reporter, valAddress)

	// get aggreate report
	qidbz, err := utils.QueryIDFromDataString(qData)
	require.NoError(t, err)

	res2, _, err := validatorI.ExecQuery(ctx, "oracle", "get-current-aggregate-report", hex.EncodeToString(qidbz))
	require.NoError(t, err)

	var aggReport AggregateReport
	fmt.Println("Aggregate report: ", string(res2))

	err = json.Unmarshal(res2, &aggReport)
	require.NoError(t, err)
	fmt.Println("Aggregate report: ", aggReport)

	require.Equal(t, aggReport.Aggregate.AggregateReporter, valAddress)
}
