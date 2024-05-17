package keeper_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	math "cosmossdk.io/math"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func TestDecodeDepositReportValue(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	AddressType, err := abi.NewType("address", "", nil)
	require.NoError(t, err)
	Uint256Type, err := abi.NewType("uint256", "", nil)
	require.NoError(t, err)
	StringType, err := abi.NewType("string", "", nil)
	require.NoError(t, err)
	reportValueArgs := abi.Arguments{
		{Type: AddressType},
		{Type: StringType},
		{Type: Uint256Type},
	}
	ethAddress := common.HexToAddress("0x3386518F7ab3eb51591571adBE62CF94540EAd29")
	layerAddressString := simtestutil.CreateIncrementalAccounts(1)[0].String()
	amountUint64 := big.NewInt(100 * 1e12)
	reportValueArgsEncoded, err := reportValueArgs.Pack(ethAddress, layerAddressString, amountUint64)
	require.NoError(t, err)
	reportValueString := hex.EncodeToString(reportValueArgsEncoded)

	recipient, amount, err := k.DecodeDepositReportValue(ctx, reportValueString)
	fmt.Println("recipient.StrinG: ", recipient.String())
	fmt.Println("amount: ", amount)
	require.NoError(t, err)

}

func TestGetDepositQueryId(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	queryId, err := k.GetDepositQueryId(0)
	require.NoError(t, err)
	require.NotNil(t, queryId)

	queryId, err = k.GetDepositQueryId(1)
	require.NoError(t, err)
	require.NotNil(t, queryId)

	maxUint64 := ^uint64(0)
	queryId, err = k.GetDepositQueryId(maxUint64)
	require.NoError(t, err)
	require.NotNil(t, queryId)

	maxPlusOne := maxUint64 + 1
	queryId, err = k.GetDepositQueryId(maxPlusOne)
	require.NoError(t, err)
	require.NotNil(t, queryId)
}

func TestClaimDepositHelper(t *testing.T) {
	k, _, bk, ok, rk, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	aggregateTimestamp := sdkCtx.BlockTime()

	AddressType, err := abi.NewType("address", "", nil)
	require.NoError(t, err)
	Uint256Type, err := abi.NewType("uint256", "", nil)
	require.NoError(t, err)
	StringType, err := abi.NewType("string", "", nil)
	require.NoError(t, err)
	reportValueArgs := abi.Arguments{
		{Type: AddressType},
		{Type: StringType},
		{Type: Uint256Type},
	}
	ethAddress := common.HexToAddress("0x3386518F7ab3eb51591571adBE62CF94540EAd29")
	layerAddressString := simtestutil.CreateIncrementalAccounts(1)[0].String()
	amountUint64 := big.NewInt(100 * 1e12)
	reportValueArgsEncoded, err := reportValueArgs.Pack(ethAddress, layerAddressString, amountUint64)
	require.NoError(t, err)
	reportValueString := hex.EncodeToString(reportValueArgsEncoded)
	queryId, err := k.GetDepositQueryId(0)
	require.NoError(t, err)
	aggregate := &oracletypes.Aggregate{
		QueryId:              queryId,
		AggregateValue:       reportValueString,
		AggregateReportIndex: int64(0),
	}
	recipient, amount, err := k.DecodeDepositReportValue(ctx, reportValueString)
	//advance 2 seconds
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(2 * time.Second))
	totalBondedTokens := math.NewInt(0 * 1e6)
	ok.On("GetAggregateByIndex", sdkCtx, queryId, uint64(aggregate.AggregateReportIndex)).Return(aggregate, aggregateTimestamp, err)
	rk.On("TotalReporterPower", sdkCtx).Return(totalBondedTokens, err)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, amount).Return(err)

	depositId := uint64(0)
	reportIndex := uint64(0)

	err = k.ClaimDepositHelper(sdkCtx, depositId, reportIndex)
	require.NoError(t, err)

}
