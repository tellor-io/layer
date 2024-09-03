package keeper_test

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMsgClaimDeposit(t *testing.T) {
	k, _, bk, ok, _, _, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	require.Panics(t, func() {
		_, err := msgServer.ClaimDeposit(ctx, nil)
		require.Error(t, err)
	})

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
		{Type: Uint256Type},
	}
	ethAddress := common.HexToAddress("0x3386518F7ab3eb51591571adBE62CF94540EAd29")
	layerAddressString := simtestutil.CreateIncrementalAccounts(1)[0].String()
	amountUint64 := big.NewInt(100 * 1e12)
	tipAmountUint64 := big.NewInt(1 * 1e12)
	reportValueArgsEncoded, err := reportValueArgs.Pack(ethAddress, layerAddressString, amountUint64, tipAmountUint64)
	require.NoError(t, err)
	reportValueString := hex.EncodeToString(reportValueArgsEncoded)
	queryId, err := k.GetDepositQueryId(0)
	require.NoError(t, err)
	aggregate := &oracletypes.Aggregate{
		QueryId:              queryId,
		AggregateValue:       reportValueString,
		AggregateReportIndex: int64(0),
		ReporterPower:        int64(68),
	}
	powerThreshold := uint64(67)
	validatorTimestamp := uint64(aggregateTimestamp.UnixMilli() - 1)
	valSetHash := []byte("valSetHash")
	_, err = k.CalculateValidatorSetCheckpoint(ctx, powerThreshold, validatorTimestamp, valSetHash)
	require.NoError(t, err)
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(13 * time.Hour))
	// convert ethAddress (common.Address) to sdk.Address
	msgSender := sdk.AccAddress(ethAddress.Bytes())
	recipient, amount, tip, err := k.DecodeDepositReportValue(ctx, reportValueString)
	claimAmount := amount.Sub(tip...)
	ok.On("GetAggregateByIndex", sdkCtx, queryId, uint64(aggregate.AggregateReportIndex)).Return(aggregate, aggregateTimestamp, err)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, claimAmount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, msgSender, tip).Return(err)

	depositId := uint64(0)
	reportIndex := uint64(0)

	result, err := msgServer.ClaimDeposit(sdkCtx, &bridgetypes.MsgClaimDepositRequest{
		Creator:   msgSender.String(),
		DepositId: depositId,
		Index:     reportIndex,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	depositClaimedResult, err := k.DepositIdClaimedMap.Get(sdkCtx, depositId)
	require.NoError(t, err)
	require.Equal(t, depositClaimedResult.Claimed, true)
}
