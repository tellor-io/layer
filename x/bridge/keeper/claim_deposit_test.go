package keeper_test

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	math "cosmossdk.io/math"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	amountAggregate := big.NewInt(1 * 1e12) // 1 loya, 0.00001 trb
	reportValueArgsEncoded, err := reportValueArgs.Pack(ethAddress, layerAddressString, amountAggregate)
	require.NoError(t, err)
	reportValueString := hex.EncodeToString(reportValueArgsEncoded)

	recipient, amount, err := k.DecodeDepositReportValue(ctx, reportValueString)
	require.Equal(t, recipient.String(), layerAddressString)
	require.Equal(t, amount.AmountOf("loya").BigInt(), amountAggregate.Div(amountAggregate, big.NewInt(1e12)))
	require.NoError(t, err)

	// decode big numbers
	amountAggregate = big.NewInt(1).Mul(big.NewInt(10), big.NewInt(1e18)) // 10 trb
	reportValueArgsEncoded, err = reportValueArgs.Pack(ethAddress, layerAddressString, amountAggregate)
	require.NoError(t, err)
	reportValueString = hex.EncodeToString(reportValueArgsEncoded)

	recipient, amount, err = k.DecodeDepositReportValue(ctx, reportValueString)
	require.Equal(t, recipient.String(), layerAddressString)
	require.Equal(t, amount.AmountOf("loya").BigInt(), amountAggregate.Div(amountAggregate, big.NewInt(1e12)))
	require.NoError(t, err)

	amountAggregate = big.NewInt(1).Mul(big.NewInt(1_000), big.NewInt(1e18)) // 1,000 trb
	reportValueArgsEncoded, err = reportValueArgs.Pack(ethAddress, layerAddressString, amountAggregate)
	require.NoError(t, err)
	reportValueString = hex.EncodeToString(reportValueArgsEncoded)

	recipient, amount, err = k.DecodeDepositReportValue(ctx, reportValueString)
	require.Equal(t, recipient.String(), layerAddressString)
	require.Equal(t, amount.AmountOf("loya").BigInt(), amountAggregate.Div(amountAggregate, big.NewInt(1e12)))
	require.NoError(t, err)

	amountAggregate = big.NewInt(1).Mul(big.NewInt(10_000), big.NewInt(1e18)) // 10,000 trb
	reportValueArgsEncoded, err = reportValueArgs.Pack(ethAddress, layerAddressString, amountAggregate)
	require.NoError(t, err)
	reportValueString = hex.EncodeToString(reportValueArgsEncoded)

	recipient, amount, err = k.DecodeDepositReportValue(ctx, reportValueString)
	require.Equal(t, recipient.String(), layerAddressString)
	require.Equal(t, amount.AmountOf("loya").BigInt(), amountAggregate.Div(amountAggregate, big.NewInt(1e12)))
	require.NoError(t, err)

	amountAggregate = big.NewInt(1).Mul(big.NewInt(1_000_000), big.NewInt(1e18)) // 1,000,000 trb
	reportValueArgsEncoded, err = reportValueArgs.Pack(ethAddress, layerAddressString, amountAggregate)
	require.NoError(t, err)
	reportValueString = hex.EncodeToString(reportValueArgsEncoded)

	recipient, amount, err = k.DecodeDepositReportValue(ctx, reportValueString)
	require.Equal(t, recipient.String(), layerAddressString)
	require.Equal(t, amount.AmountOf("loya").BigInt(), amountAggregate.Div(amountAggregate, big.NewInt(1e12)))
	require.NoError(t, err)
}

func TestDecodeDepositReportValueInvalidReport(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	badString := "0x"
	_, _, err := k.DecodeDepositReportValue(ctx, badString)
	require.Error(t, err)

	emptyString := ""
	_, _, err = k.DecodeDepositReportValue(ctx, emptyString)
	require.Error(t, err)

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
	_, err = reportValueArgs.Pack(layerAddressString, ethAddress, amountUint64)
	require.Error(t, err)
	_, err = reportValueArgs.Pack(layerAddressString, amountUint64, ethAddress)
	require.Error(t, err)
	_, err = reportValueArgs.Pack(ethAddress, amountUint64, layerAddressString)
	require.Error(t, err)
	_, err = reportValueArgs.Pack(amountUint64, layerAddressString, ethAddress)
	require.Error(t, err)
	_, err = reportValueArgs.Pack(amountUint64, ethAddress, layerAddressString)
	require.Error(t, err)
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

func TestClaimDeposit(t *testing.T) {
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
		ReporterPower:        int64(67),
	}
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(13 * time.Hour))
	recipient, amount, err := k.DecodeDepositReportValue(ctx, reportValueString)
	totalBondedTokens := math.NewInt(100)
	ok.On("GetAggregateByIndex", sdkCtx, queryId, uint64(aggregate.AggregateReportIndex)).Return(aggregate, aggregateTimestamp, err)
	rk.On("TotalReporterPower", sdkCtx).Return(totalBondedTokens, err)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, amount).Return(err)

	depositId := uint64(0)
	reportIndex := uint64(0)
	err = k.ClaimDeposit(sdkCtx, depositId, reportIndex)
	require.NoError(t, err)
	depositClaimedResult, err := k.DepositIdClaimedMap.Get(sdkCtx, depositId)
	require.NoError(t, err)
	require.Equal(t, depositClaimedResult.Claimed, true)
}

func TestClaimDepositNilAggregate(t *testing.T) {
	k, _, _, ok, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	queryId, _ := k.GetDepositQueryId(0)
	currentTime := time.Now()
	ok.On("GetAggregateByIndex", sdkCtx, queryId, uint64(0)).Return(nil, currentTime, nil)

	err := k.ClaimDeposit(ctx, 0, 0)
	require.ErrorContains(t, err, "no aggregate found")
}

func TestClaimDepositReportFlaggedAggregate(t *testing.T) {
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
		ReporterPower:        int64(90 * 1e6),
		Flagged:              true,
	}
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(13 * time.Hour))
	recipient, amount, err := k.DecodeDepositReportValue(ctx, reportValueString)
	totalBondedTokens := math.NewInt(100 * 1e6)
	ok.On("GetAggregateByIndex", sdkCtx, queryId, uint64(aggregate.AggregateReportIndex)).Return(aggregate, aggregateTimestamp, err)
	rk.On("TotalReporterPower", sdkCtx).Return(totalBondedTokens, err)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, amount).Return(err)

	depositId := uint64(0)
	reportIndex := uint64(0)
	err = k.ClaimDeposit(sdkCtx, depositId, reportIndex)
	require.ErrorContains(t, err, "aggregate flagged")
}

func TestClaimDepositTotalReporterPowerErr(t *testing.T) {
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
		ReporterPower:        int64(90 * 1e6),
	}
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(13 * time.Hour))
	recipient, amount, err := k.DecodeDepositReportValue(ctx, reportValueString)
	totalBondedTokens := math.NewInt(100 * 1e6)
	ok.On("GetAggregateByIndex", sdkCtx, queryId, uint64(aggregate.AggregateReportIndex)).Return(aggregate, aggregateTimestamp, err)
	rk.On("TotalReporterPower", sdkCtx).Return(totalBondedTokens, err)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, amount).Return(err)

	depositId := uint64(0)
	reportIndex := uint64(0)
	err = k.ClaimDeposit(sdkCtx, depositId, reportIndex)
	require.NoError(t, err)
	depositClaimedResult, err := k.DepositIdClaimedMap.Get(sdkCtx, depositId)
	require.NoError(t, err)
	require.Equal(t, depositClaimedResult.Claimed, true)
}

func TestClaimDepositNotEnoughPower(t *testing.T) {
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
	// 66/100
	aggregate := &oracletypes.Aggregate{
		QueryId:              queryId,
		AggregateValue:       reportValueString,
		AggregateReportIndex: int64(0),
		ReporterPower:        int64(66),
	}
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(13 * time.Hour))
	recipient, amount, err := k.DecodeDepositReportValue(ctx, reportValueString)
	totalBondedTokens := math.NewInt(100)
	ok.On("GetAggregateByIndex", sdkCtx, queryId, uint64(aggregate.AggregateReportIndex)).Return(aggregate, aggregateTimestamp, err)
	rk.On("TotalReporterPower", sdkCtx).Return(totalBondedTokens, err)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, amount).Return(err)

	depositId := uint64(0)
	reportIndex := uint64(0)
	err = k.ClaimDeposit(sdkCtx, depositId, reportIndex)
	require.ErrorContains(t, err, "insufficient reporter power")
}

func TestClaimDepositReportTooYoung(t *testing.T) {
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
		ReporterPower:        int64(90 * 1e6),
	}
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(11 * time.Hour))
	recipient, amount, err := k.DecodeDepositReportValue(ctx, reportValueString)
	totalBondedTokens := math.NewInt(100 * 1e6)
	ok.On("GetAggregateByIndex", sdkCtx, queryId, uint64(aggregate.AggregateReportIndex)).Return(aggregate, aggregateTimestamp, err)
	rk.On("TotalReporterPower", sdkCtx).Return(totalBondedTokens, err)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, amount).Return(err)

	depositId := uint64(0)
	reportIndex := uint64(0)
	err = k.ClaimDeposit(sdkCtx, depositId, reportIndex)
	require.ErrorContains(t, err, "report too young")
}

func TestClaimDepositSpam(t *testing.T) {
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
		ReporterPower:        int64(67),
	}
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(13 * time.Hour))
	recipient, amount, err := k.DecodeDepositReportValue(ctx, reportValueString)
	totalBondedTokens := math.NewInt(100)
	ok.On("GetAggregateByIndex", sdkCtx, queryId, uint64(aggregate.AggregateReportIndex)).Return(aggregate, aggregateTimestamp, err)
	rk.On("TotalReporterPower", sdkCtx).Return(totalBondedTokens, err)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, amount).Return(err)

	depositId := uint64(0)
	reportIndex := uint64(0)
	err = k.ClaimDeposit(sdkCtx, depositId, reportIndex)
	require.NoError(t, err)
	depositClaimedResult, err := k.DepositIdClaimedMap.Get(sdkCtx, depositId)
	require.NoError(t, err)
	require.Equal(t, depositClaimedResult.Claimed, true)

	attempts := 0
	for attempts < 100 {
		attempts++
		err = k.ClaimDeposit(sdkCtx, depositId, reportIndex)
		require.ErrorContains(t, err, "deposit already claimed")
	}
}
