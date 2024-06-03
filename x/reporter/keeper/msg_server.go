package keeper

import (
	"context"
	"errors"

	errorsmod "cosmossdk.io/errors"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/reporter/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) CreateReporter(goCtx context.Context, msg *types.MsgCreateReporter) (*types.MsgCreateReporterResponse, error) {
	reporter := sdk.MustAccAddressFromBech32(msg.ReporterAddress)
	reporterExists, err := k.Reporters.Has(goCtx, reporter.Bytes())
	if err != nil {
		return nil, err
	}
	if reporterExists {
		return nil, errorsmod.Wrapf(types.ErrReporterExists, "cannot create reporter with address %s, it already exists", msg.ReporterAddress)
	}

	minCommRate, err := k.MinCommissionRate(goCtx)
	if err != nil {
		return nil, err
	}
	if msg.Commission.Rate.LT(minCommRate) {
		return nil, errorsmod.Wrapf(types.ErrCommissionLTMinRate, "cannot set commission to less than minimum rate of %s", minCommRate)
	}

	commission := types.NewCommissionWithTime(msg.Commission.Rate, msg.Commission.MaxRate,
		msg.Commission.MaxChangeRate, sdk.UnwrapSDKContext(goCtx).HeaderInfo().Time)

	if err := commission.Validate(); err != nil {
		return nil, err
	}
	// create a new reporter
	newReporter := types.NewOracleReporter(msg.ReporterAddress, &commission)
	if err := k.Reporters.Set(goCtx, reporter.Bytes(), newReporter); err != nil {
		return nil, err
	}
	return &types.MsgCreateReporterResponse{}, nil
}

func (k msgServer) ChangeReporter(goCtx context.Context, msg *types.MsgChangeReporter) (*types.MsgChangeReporterResponse, error) {
	newReporterAddr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)
	// get delegation
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
	delegation, err := k.Keeper.Delegators.Get(goCtx, delAddr.Bytes())
	if err != nil {
		return nil, err
	}

	reporterExists, err := k.Keeper.Reporters.Has(goCtx, newReporterAddr)
	if err != nil {
		return nil, err
	}

	if !reporterExists {
		return nil, errors.New("reporter does not exist")
	}

	iter, err := k.Keeper.Delegators.Indexes.Reporter.MatchExact(goCtx, newReporterAddr.Bytes())
	if err != nil {
		return nil, err
	}

	keys, err := iter.FullKeys()
	if err != nil {
		return nil, err
	}

	if len(keys) >= 100 {
		return nil, errors.New("reporter is at max cap")
	}

	delegation.Reporter = newReporterAddr.Bytes()
	if err := k.Keeper.Delegators.Set(goCtx, delAddr.Bytes(), delegation); err != nil {
		return nil, err
	}
	return &types.MsgChangeReporterResponse{}, nil
}

func (k msgServer) UnjailReporter(goCtx context.Context, msg *types.MsgUnjailReporter) (*types.MsgUnjailReporterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	reporterAddr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)

	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}

	if err := k.Keeper.unjailReporter(ctx, reporterAddr, reporter); err != nil {
		return nil, err
	}

	return &types.MsgUnjailReporterResponse{}, nil
}

func (k msgServer) WithdrawTip(goCtx context.Context, msg *types.MsgWithdrawTip) (*types.MsgWithdrawTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
	shares, err := k.Keeper.DelegatorTips.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}
	val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	if !val.IsBonded() {
		return nil, errors.New("chosen validator must be bonded")
	}
	_, err = k.Keeper.stakingKeeper.Delegate(ctx, delAddr, shares, val.Status, val, false)
	if err != nil {
		return nil, err
	}

	err = k.Keeper.DelegatorTips.Remove(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	// send coins
	err = k.Keeper.bankKeeper.SendCoinsFromModuleToModule(ctx, types.TipsEscrowPool, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, shares)))
	if err != nil {
		return nil, err
	}

	return &types.MsgWithdrawTipResponse{}, nil
}
