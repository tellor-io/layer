package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"

	"github.com/cosmos/cosmos-sdk/telemetry"
)

type QueryDataStorage struct {
	QueryType     string
	QueryData     string
	ExpectedValue string
	TipAmount     sdk.Coin
}

// ~22 min to run 100 aggregates...
func TestLotsOfAggregatesInSameBlock(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	layer := e2e.LayerSpinup(t)

	numAggregates := 100

	// get a validator to call upon
	telemetry.MeasureSince(time.Now(), "get val addresses")
	validators, valAccAddresses, _, err := e2e.GetValAddresses(ctx, layer)
	require.NoError(err)
	validatorI := validators[0]
	valIAccAddress := valAccAddresses[0]

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, layer, validatorI))
	telemetry.MeasureSince(time.Now(), "turn on minting")

	// validatorI becomes a reporter
	txHash, err := validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(err)
	fmt.Println("TX HASH (validatorI becomes a reporter): ", txHash)
	telemetry.MeasureSince(time.Now(), "validatorI becomes a reporter")
	// Create and fund a user who will be tipping
	userName := "user1"
	userStartingBalance := math.NewInt(999_999 * 1e6)
	user := interchaintest.GetAndFundTestUsers(t, ctx, userName, userStartingBalance, layer)[0]

	// create data specs that expire 4 blocks apart
	// store necessary tip/submit values for each query in a struct
	// 4 blocks apart because we want to see a bunch of aggregates happen on the same end block .ExecTx takes 2 blocks to complete
	// set a random tip and value for each query
	expectedTipTotal := math.ZeroInt()
	registrar := valAccAddresses[0]
	var queryDatas []QueryDataStorage
	for i := 1; i <= numAggregates; i++ {
		currentHeight, err := validatorI.Height(ctx)
		require.NoError(err)
		fmt.Println("Current height: ", currentHeight)

		expirationWindow := i * 4
		dataSpec, err := e2e.CreateDataSpec(expirationWindow, registrar)
		require.NoError(err)
		jsonSpec, err := json.Marshal(dataSpec)
		require.NoError(err)
		queryType := fmt.Sprintf("queryType%d", expirationWindow)
		txHash, err := validatorI.ExecTx(ctx, "validator", "registry", "register-spec", queryType, string(jsonSpec), "--keyring-dir", "/var/cosmos-chain/layer-1")
		require.NoError(err)
		fmt.Printf("TX Hash (create data spec with expiration %d): %s\n", expirationWindow, txHash)

		queryDataBz, _, err := validatorI.ExecQuery(ctx, "registry", "generate-querydata", queryType, "[\"trb\", \"usd\"]")
		require.NoError(err)
		var qData e2e.GenerateQueryDataResponse
		require.NoError(json.Unmarshal(queryDataBz, &qData))
		queryDataHexString := hex.EncodeToString(qData.QueryData)

		// generate a random tip amount
		randomTipInt := rand.Int63n(1000000) + 1
		randomTip := sdk.NewCoin("loya", math.NewInt(randomTipInt))

		value := layerutil.EncodeValue(rand.Float64() * 100)

		queryDatas = append(queryDatas, QueryDataStorage{
			QueryType:     queryType,
			QueryData:     queryDataHexString,
			TipAmount:     randomTip,
			ExpectedValue: value,
		})

		twoPercent := randomTip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100))
		expectedTipTotal = expectedTipTotal.Add(randomTip.Amount.Sub(twoPercent))
	}
	fmt.Println("Expected tip total: ", expectedTipTotal.String())

	// Send a tip from a user on block n, submit a value on block n+2, submit next tip on block n+4, etc
	// reverse iterate so the query with the longest expiry is tipped first
	// do not add queries or computations to this loop or else it may throw off 4 block cycle
	// 4 block cycle is still not guaranteed
	var currentHeight int64
	var txHashes []string
	for i := len(queryDatas) - 1; i >= 0; i-- {
		currentHeight, err = validatorI.Height(ctx)
		require.NoError(err)
		fmt.Println("Current height: ", currentHeight)

		// tip for query
		txHash, err := validatorI.ExecTx(ctx, userName, "oracle", "tip", user.FormattedAddress(), string(queryDatas[i].QueryData), queryDatas[i].TipAmount.String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
		require.NoError(err)
		txHashes = append(txHashes, txHash)
		fmt.Printf("TX Hash (user tip %s): %s\n", queryDatas[i].TipAmount.String(), txHash)

		// query tip for the query type
		// tip, err := e2e.QueryTips(string(queryDatas[i].QueryData), ctx, validatorI)
		// require.NoError(err)
		// fmt.Println("Tip: ", tip.Tips.String())

		// submit value for query
		txHash, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", valIAccAddress, string(queryDatas[i].QueryData), queryDatas[i].ExpectedValue, "--keyring-dir", "/var/cosmos-chain/layer-1")
		require.NoError(err)
		fmt.Printf("TX Hash (validator submits value for query %d): %s\n", i+1, txHash)
	}

	// query available tips as sanity check, all queries should be open still
	// allTips, _, err := validatorI.ExecQuery(ctx, "oracle", "tipped-queries")
	// require.NoError(err)
	// fmt.Println("All tips: ", string(allTips))
	// var tippedQueries e2e.TippedQueriesResponse
	// require.NoError(json.Unmarshal(allTips, &tippedQueries))
	// fmt.Println("Tipped queries: ", tippedQueries)

	// wait for aggregates to be set
	require.NoError(testutil.WaitForBlocks(ctx, 4, validatorI))

	// todo: claim tbr and tip rewards, check balances
}
