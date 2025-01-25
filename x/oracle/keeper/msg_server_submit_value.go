package keeper

import (
	"context"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// SubmitValue: allow a reporter to submit a value for a query.
// accepts QueryData (bytes), Value (string)
// QueryData undergoes multiple checks to ensure it is a valid query
// 1. Check if the queryData is decodable
// 2. Check if the queryData is a TRBBridgeQueryType
//   - If it is, check if it is a bridge withdrawal (not accepted) or deposit report
//
// 3. Check if the reporter has enough stake to submit the value
// 4. Fetch the queryMeta for the queryId (hash(queryData)) if it exists
//   - If it does not exist and it is a bridge deposit, generate a new queryMeta.
//     Note: Bridge deposit reports are always accepted and do not require to have a tip or be in the cycle list
//
// 5. Check if the queryMeta has a tip or is in the cycle list and is not expired
// 6. Further checks to validate the value is decodable to the expected spec type.
// 7. Set queryMeta.HasRevealedReports to true
// 8. Emit an event for the new report
func (k msgServer) SubmitValue(ctx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	err := validateSubmitValue(msg)
	if err != nil {
		return nil, err
	}

	reporterAddr, err := msg.GetSignerAndValidateMsg()
	if err != nil {
		return nil, err
	}

	isTokenBridgeDeposit, err := k.keeper.PreventBridgeWithdrawalReport(msg.QueryData)
	if err != nil {
		return nil, err
	}
	queryId := utils.QueryIDFromData(msg.QueryData)
	// get reporter
	reporterStake, err := k.keeper.reporterKeeper.ReporterStake(ctx, reporterAddr, queryId)
	if err != nil {
		return nil, err
	}
	params, err := k.keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if reporterStake.LT(params.MinStakeAmount) {
		return nil, errorsmod.Wrapf(types.ErrNotEnoughStake, "reporter has %s, required %s", reporterStake, params.MinStakeAmount)
	}

	reportingPower := reporterStake.Quo(layertypes.PowerReduction).Uint64()

	query, err := k.keeper.CurrentQuery(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
		if !isTokenBridgeDeposit {
			return nil, types.ErrNotTokenDeposit
		}
		query, err = k.keeper.TokenBridgeDepositQuery(ctx, msg.QueryData)
		if err != nil {
			return nil, err
		}

		err = k.keeper.Query.Set(ctx, collections.Join(queryId, query.Id), query)
		if err != nil {
			return nil, err
		}
		err = k.keeper.HandleBridgeDepositDirectReveal(ctx, query, msg.QueryData, reporterAddr, msg.Value, reportingPower)
		if err != nil {
			return nil, err
		}
		return &types.MsgSubmitValueResponse{}, nil
	}

	err = k.keeper.DirectReveal(ctx, query, msg.QueryData, msg.Value, reporterAddr, reportingPower, isTokenBridgeDeposit)
	if err != nil {
		return nil, err
	}
	return &types.MsgSubmitValueResponse{}, nil
}

func (k Keeper) DirectReveal(ctx context.Context,
	query types.QueryMeta,
	qDataBytes []byte,
	value string,
	reporterAddr sdk.AccAddress,
	votingPower uint64,
	bridgeDeposit bool,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	if bridgeDeposit {
		return k.HandleBridgeDepositDirectReveal(ctx, query, qDataBytes, reporterAddr, value, votingPower)
	}

	if query.Amount.IsZero() && !query.CycleList {
		return types.ErrNoTipsNotInCycle
	}

	if query.Expiration < uint64(blockHeight) {
		return types.ErrSubmissionWindowExpired
	}

	return k.SetValue(ctx, reporterAddr, query, value, qDataBytes, votingPower, query.CycleList)
}

// replacement for ValidateBasic
func validateSubmitValue(msg *types.MsgSubmitValue) error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	// make sure query data is not empty
	if len(msg.QueryData) == 0 {
		return errors.New("MsgSubmitValue query data cannot be empty (%s)")
	}
	// make sure value is not empty
	if msg.Value == "" {
		return errors.New("MsgSubmitValue value field cannot be empty (%s)")
	}
	return nil
}
