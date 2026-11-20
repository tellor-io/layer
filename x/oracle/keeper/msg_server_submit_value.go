package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/tellor-io/layer/lib/metrics"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// TwoBitAgenticQueryType is the prefix of every query type an aliased 2bit agent
// may report. The reporting window is part of the type, so 2bitAgentic,
// 2bitAgentic90, and 2bitAgentic500 are distinct queries that share this prefix.
const TwoBitAgenticQueryType = "2bitAgentic"

// SubmitValue: allow a reporter to submit a value for a query.
// accepts QueryData (bytes), Value (string)
// QueryData undergoes multiple checks to ensure it is a valid query
// 1. Check if the queryData is decodable
// 2. Check if the queryData is a TRBBridgeV2QueryType
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
func (k msgServer) SubmitValue(ctx context.Context, msg *types.MsgSubmitValue) (res *types.MsgSubmitValueResponse, err error) {
	valueBytes, err := hex.DecodeString(registrytypes.Remove0xPrefix(msg.Value))
	if err != nil {
		return nil, err
	}
	msg.Value = hex.EncodeToString(valueBytes)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err = validateSubmitValue(msg)
	if err != nil {
		return nil, err
	}

	reporterAddr, err := msg.GetSignerAndValidateMsg()
	if err != nil {
		return nil, err
	}

	isTokenBridgeDeposit, err := k.keeper.PreventBridgeWithdrawalReport(ctx, msg.QueryData)
	if err != nil {
		return nil, err
	}
	queryId := utils.QueryIDFromData(msg.QueryData)
	// get reporter. A signer with no reporter row can still report a query whose
	// type starts with 2bitAgentic when that address was registered as another
	// reporter's agent. The stored report uses the registered reporter, including
	// that reporter's power and jail state.
	reporterStake, stakeSnapshot, err := k.keeper.reporterKeeper.ReporterStakeWithSnapshot(ctx, reporterAddr)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
		resolved, resolveErr := k.twoBitReporter(ctx, reporterAddr, msg.QueryData)
		if resolveErr != nil {
			return nil, err
		}
		reporterAddr = resolved
		reporterStake, stakeSnapshot, err = k.keeper.reporterKeeper.ReporterStakeWithSnapshot(ctx, reporterAddr)
		if err != nil {
			return nil, err
		}
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
	defer func() {
		if err == nil {
			telemetry.SetGaugeWithLabels([]string{"submit_value_gas_consumed"}, float32(sdkCtx.GasMeter().GasConsumed()), []metrics.Label{{Name: "chain_id", Value: sdkCtx.ChainID()}, {Name: "query_type", Value: query.QueryType}})
		}
	}()
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
		err = k.keeper.HandleBridgeDepositDirectReveal(ctx, query, msg.QueryData, reporterAddr, msg.Value, reportingPower, stakeSnapshot)
		if err != nil {
			return nil, err
		}
		return &types.MsgSubmitValueResponse{}, nil
	}

	err = k.keeper.DirectReveal(ctx, query, msg.QueryData, msg.Value, reporterAddr, reportingPower, stakeSnapshot, isTokenBridgeDeposit)
	if err != nil {
		return nil, err
	}

	return &types.MsgSubmitValueResponse{Id: query.Id}, nil
}

func (k Keeper) DirectReveal(ctx context.Context,
	query types.QueryMeta,
	qDataBytes []byte,
	value string,
	reporterAddr sdk.AccAddress,
	votingPower uint64,
	stakeSnapshot reportertypes.StakeSnapshotRef,
	bridgeDeposit bool,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	if bridgeDeposit {
		return k.HandleBridgeDepositDirectReveal(ctx, query, qDataBytes, reporterAddr, value, votingPower, stakeSnapshot)
	}

	if query.Amount.IsZero() && !query.CycleList {
		return types.ErrNoTipsNotInCycle
	}

	if query.Expiration < uint64(blockHeight) {
		return types.ErrSubmissionWindowExpired
	}

	return k.SetValue(ctx, reporterAddr, query, value, qDataBytes, votingPower, stakeSnapshot, query.CycleList)
}

// replacement for ValidateBasic
func validateSubmitValue(msg *types.MsgSubmitValue) error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	// make sure query data is not empty
	if len(msg.QueryData) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "query data cannot be empty")
	}
	// make sure value is not empty
	if msg.Value == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "value cannot be empty")
	}
	return nil
}

// twoBitReporter resolves a signer that is not a reporter to the reporter that
// registered it, and only for a query type that starts with 2bitAgentic. Any
// other case returns an error so the caller keeps the original reporter-not-found result.
func (k msgServer) twoBitReporter(ctx context.Context, signer sdk.AccAddress, queryData []byte) (sdk.AccAddress, error) {
	queryType, _, err := registrytypes.DecodeQueryType(queryData)
	if err != nil {
		return nil, err
	}
	if !hasTwoBitAgenticPrefix(queryType) {
		return nil, collections.ErrNotFound
	}
	return k.keeper.reporterKeeper.TwoBitReporter(ctx, signer)
}

// hasTwoBitAgenticPrefix reports whether queryType is 2bitAgentic or a
// window-specific variant such as 2bitAgentic500. Matching ignores case.
func hasTwoBitAgenticPrefix(queryType string) bool {
	if len(queryType) < len(TwoBitAgenticQueryType) {
		return false
	}
	return strings.EqualFold(queryType[:len(TwoBitAgenticQueryType)], TwoBitAgenticQueryType)
}
