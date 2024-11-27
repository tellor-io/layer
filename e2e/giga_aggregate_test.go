package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"
)

const (
	type6qData  = "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005747970653600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000001610000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000016200000000000000000000000000000000000000000000000000000000000000"
	typ6qId     = "0x3309f88571c671b353c0082cd4219096885b6805b58a9e6aa086c4a46ddbdcde"
	type12qData = "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000006747970653132000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000001610000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000016200000000000000000000000000000000000000000000000000000000000000"
)

// Hold user addr and tip amount
type UserTip struct {
	Address string
	Tip     sdk.Coin
}

type QueryDataStorage struct {
	QueryType     string
	QueryData     string
	ExpectedValue string
}

func TestLotsOfAggregatesInSameBlock(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	layer := e2e.LayerSpinup(t)

	// get a validator
	validators, valAccAddresses, _, err := e2e.GetValAddresses(ctx, layer)
	require.NoError(err)
	validatorI := validators[0]
	valIAccAddress := valAccAddresses[0]
	// validatorII := validators[1]

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, layer, validatorI))

	// validatorI becomes a reporter
	txHash, err := validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(err)
	fmt.Println("TX HASH (validatorI becomes a reporter): ", txHash)

	// Create and fund n users, set how much they are going to tip
	n := 5
	var userTips []UserTip
	expectedTipTotal := math.ZeroInt()
	for i := 0; i < n; i++ {
		userName := fmt.Sprintf("user%d", i+1)
		user := interchaintest.GetAndFundTestUsers(t, ctx, userName, math.NewInt(900_000*1e6), layer)[0]

		// Generate a random tip amount
		randomTipInt := rand.Int63n(1000000) + 1
		randomTip := sdk.NewCoin("loya", math.NewInt(randomTipInt))

		// Store the user's address and the random tip amount
		userTips = append(userTips, UserTip{
			Address: user.FormattedAddress(),
			Tip:     randomTip,
		})

		twoPercent := randomTip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100))
		expectedTipTotal = expectedTipTotal.Add(randomTip.Amount.Sub(twoPercent))
	}
	fmt.Println("Expected tip total: ", expectedTipTotal.String())

	// create data specs that expire 4 blocks apart
	// store the query data for each spec
	// 4 blocks apart because we want to see a bunch of aggregates happen on the same end block z
	// .ExecTx takes 2 blocks to complete
	// queryType1 is tipped on block n, submitted for on n+2, queryType2 is tipped on block n+4, submitted for on n+6, ...
	// So queryType1 to be tipped on block z - n,

	registrar := valAccAddresses[0]
	var queryDatas []QueryDataStorage
	for i := 1; i <= 5; i++ {
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

		queryDatas = append(queryDatas, QueryDataStorage{
			QueryType: queryType,
			QueryData: queryDataHexString,
		})
	}

	// Send a tip from a user on block n, submit a value on block n+2 in reverse order
	var currentHeight int64
	var txHashes []string
	for i := len(userTips) - 1; i >= 0; i-- { // Reverse iteration
		currentHeight, err = validatorI.Height(ctx)
		require.NoError(err)
		fmt.Println("Current height: ", currentHeight)

		queryData := queryDatas[i].QueryData
		userAddr := userTips[i].Address
		randomTip := userTips[i].Tip
		userName := fmt.Sprintf("user%d", i+1)
		txHash, err := validatorI.ExecTx(ctx, userName, "oracle", "tip", userAddr, string(queryData), randomTip.String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
		require.NoError(err)
		txHashes = append(txHashes, txHash)
		fmt.Printf("TX Hash (user %d tip %s): %s\n", i+1, randomTip.String(), txHash)

		// query tip for the query
		tip, err := e2e.QueryTips(string(queryData), ctx, validatorI)
		require.NoError(err)
		fmt.Println("Tip: ", tip.Tips.String())

		value := layerutil.EncodeValue(rand.Float64() * 100)
		queryDatas[i].ExpectedValue = value
		txHash, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", valIAccAddress, string(queryData), value, "--keyring-dir", "/var/cosmos-chain/layer-1")
		require.NoError(err)
		fmt.Printf("TX Hash (validator submits value for query %d): %s\n", i+1, txHash)
	}

	// query available tips
	allTips, _, err := validatorI.ExecQuery(ctx, "oracle", "tipped-queries")
	require.NoError(err)
	fmt.Println("All tips: ", allTips)

	// wait for aggregates to be set
	require.NoError(testutil.WaitForBlocks(ctx, 10, validatorI))

	// query data spec
	// getSpecResponse, _, err := validatorI.ExecQuery(ctx, "registry", "data-spec", queryType)
	// require.NoError(err)
	// var getSpec e2e.GetDataSpecResponse
	// require.NoError(json.Unmarshal(getSpecResponse, &getSpec))
	// fmt.Println("Data spec: ", getSpec)

	// for _, txHash := range txHashes {
	// 	resp, err := layer.GetTransaction(txHash)
	// 	// require.NoError(err)
	// 	fmt.Println("tx response: ", resp)
	// 	fmt.Println("tx hash: ", txHash)
	// 	fmt.Println("err: ", err)
	// }

	// // get block num from tx hashes
	// blockResult, err := validatorI.Client.BlockResults(ctx, &currentHeight)
	// require.NoError(err)
	// fmt.Println("Block num: ", blockResult)

	// fmt.Println("Current tips: ", getCurrentTips.Tips.String())
	// fmt.Println("Expected tip total: ", expectedTipTotal.String())
}
