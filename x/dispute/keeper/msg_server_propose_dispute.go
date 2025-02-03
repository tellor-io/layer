package keeper

import (
	"context"
	"errors"
	"strings"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) ProposeDispute(goCtx context.Context, msg *types.MsgProposeDispute) (*types.MsgProposeDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, disputed_reporter, err := validateProposeDispute(msg)
	if err != nil {
		return nil, err
	}
	ctx.Logger().Info("Proposing dispute", "reporter", msg.Report.Reporter, "report", msg.Report, "disputeCategory", msg.DisputeCategory, "fee", msg.Fee.Amount)

	qId, err := utils.QueryBytesFromString(msg.ReportQueryId)
	if err != nil {
		return nil, err
	}

	report, exists, err := k.oracleKeeper.ValidateMicroReportExists(ctx, disputed_reporter, msg.ReportMetaId, qId)
	if !exists {
		return nil, types.ErrDisputedReportDoesNotExist
	}
	if err != nil {
		return nil, err
	}

	if msg.Fee.Amount.LT(layer.OnePercent) {
		return nil, types.ErrMinimumTRBrequired.Wrapf("fee %s doesn't meet minimum fee required", msg.Fee.Amount)
	}
	// return an error if the proposer attempts to create a dispute on themselves while paying from their bond
	if msg.PayFromBond && strings.EqualFold(msg.Creator, msg.DisputedReporter) {
		return nil, types.ErrSelfDisputeFromBond.Wrapf("proposer cannot pay from their bond when creating a dispute on themselves")
	}

	dispute, err := k.GetDisputeByReporter(ctx, *report, msg.DisputeCategory)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// event gets emitted in SetNewDispute
			if err := k.Keeper.SetNewDispute(ctx, sender, *msg, report); err != nil {
				return nil, err
			}
			return &types.MsgProposeDisputeResponse{}, nil
		}
		return nil, err
	}
	// Add round to Existing Dispute - emits event
	if err := k.Keeper.AddDisputeRound(ctx, sender, dispute, *msg); err != nil {
		return nil, err
	}
	return &types.MsgProposeDisputeResponse{}, nil
}

func validateProposeDispute(msg *types.MsgProposeDispute) (creator sdk.AccAddress, disputed_reporter sdk.AccAddress, err error) {
	creator, err = sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	disputed_reporter, err = sdk.AccAddressFromBech32(msg.DisputedReporter)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid disputed reporter address (%s)", err)
	}
	// ensure that the fee matches the layer.BondDenom and the amount is a positive number
	if msg.Fee.Denom != layer.BondDenom || msg.Fee.Amount.IsZero() || msg.Fee.Amount.IsNegative() {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid fee amount (%s)", msg.Fee.Amount.String())
	}
	if msg.ReportQueryId == "" {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query id should not be nil")
	}
	if msg.DisputeCategory != types.Warning && msg.DisputeCategory != types.Minor && msg.DisputeCategory != types.Major {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "dispute category should be either Warning, Minor, or Major")
	}
	return creator, disputed_reporter, nil
}
