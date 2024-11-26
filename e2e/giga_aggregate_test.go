package e2e_test

import (
	"context"
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
)

// Hold user addr and tip amount
type UserTip struct {
	Address string
	Tip     sdk.Coin
}

func TestLotsOfAggregatesInSameBlock(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	layer := e2e.LayerSpinup(t)

	// get a validator
	validators, valAccAddresses, _, err := e2e.GetValAddresses(ctx, layer)
	require.NoError(err)
	validatorI := validators[0]
	// validatorII := validators[1]

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, layer, validatorI))

	// Create and fund n users that are tipping t amount
	n := 10
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

	// create data specs that expire every 2 blocks
	// create a data spec
	registrar := valAccAddresses[0]
	queryType := "TWAP"
	spec := e2e.RegisterDataSpec(ctx, layer, validatorI, []byte(queryType))
	txHash, err := validatorI.ExecTx(ctx, "validator", "registry", "register-spec", registrar, queryType, spec, "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(err)
	fmt.Println("TX Hash (create data spec): ", txHash)

	var currentHeight int64
	var txHashes []string
	for i := 0; i < len(userTips); i++ {
		// Get the current block height
		currentHeight, err = validatorI.Height(ctx)
		require.NoError(err)
		fmt.Println("Current height: ", currentHeight)

		// Send a tip from user i
		userAddr := userTips[i].Address
		randomTip := userTips[i].Tip
		userName := fmt.Sprintf("user%d", i+1)

		txHash, err := validatorI.ExecTx(ctx, userName, "oracle", "tip", userAddr, ltcusdQData, randomTip.String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
		require.NoError(err)
		txHashes = append(txHashes, txHash)
		fmt.Printf("TX Hash (user %d tip %s): %s\n", i+1, randomTip.String(), txHash)
	}

	// query available tips
	require.NoError(testutil.WaitForBlocks(ctx, 4, validatorI))
	availableTips, _, err := validatorI.ExecQuery(ctx, "oracle", "get-current-tip", ltcusdQData)
	require.NoError(err)
	var getCurrentTips e2e.CurrentTipsResponse
	require.NoError(json.Unmarshal(availableTips, &getCurrentTips))

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

	fmt.Println("Current tips: ", getCurrentTips.Tips.String())
	fmt.Println("Expected tip total: ", expectedTipTotal.String())
}
