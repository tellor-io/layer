package keeper

import (
	"bytes"
	"context"
	"errors"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CommitReport(ctx context.Context, msg *types.MsgCommitReport) (*types.MsgCommitReportResponse, error) {
	reporterAddr, err := msg.GetSignerAndValidateMsg()
	if err != nil {
		return nil, err
	}

	// get reporter
	reporterStake, err := k.keeper.reporterKeeper.ReporterStake(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}

	params, err := k.keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if reporterStake.LT(params.MinStakeAmount) {
		return nil, errorsmod.Wrapf(types.ErrNotEnoughStake, "reporter has %s, required amount is %s", reporterStake, params.MinStakeAmount)
	}

	// get query id bytes hash from query data
	queryId := utils.QueryIDFromData(msg.QueryData)

	// get query info by query id
	query, err := k.keeper.Query.Get(ctx, queryId)
	if err != nil {
		// if no query it means its not a cyclelist query and doesn't have tips (cyclelist queries are initialized in genesis)
		if errors.Is(err, collections.ErrNotFound) {
			// check if query is token bridge deposit
			query, err = k.keeper.tokenBridgeDepositCheck(msg.QueryData)
			if errors.Is(err, types.ErrNotTokenDeposit) {
				return nil, types.ErrNoTipsNotInCycle.Wrapf("query not part of cyclelist")
			}
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	// get current query in cycle
	cycleQuery, err := k.keeper.GetCurrentQueryInCycleList(ctx)
	if err != nil {
		return nil, err
	}
	// bool to check if query is in cycle
	incycle := bytes.Equal(msg.QueryData, cycleQuery)

	isBridgeDeposit := query.QueryType == TRBBridgeQueryType

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()
	if query.Amount.IsZero() && query.Expiration.Before(blockTime) && !incycle && !isBridgeDeposit {
		return nil, types.ErrNoTipsNotInCycle.Wrapf("query does not have tips and is not in cycle")
	}

	if query.Amount.GT(math.ZeroInt()) && query.Expiration.Before(blockTime) && !incycle && !isBridgeDeposit {
		return nil, errors.New("query's tip is expired and is not in cycle")
	}

	// if tip is zero and expired, move query forward only if in cycle
	// if tip amount is zero and query timeframe is expired, it means one of two things
	// the tip has been paid out because the query has expired and there were revealed reports
	// or the query was in cycle and expired (either revealed or not)
	// in either case move query forward by incrementing id and setting expiration
	// if the query is a bridge deposit, it should always be in cycle
	if query.Amount.IsZero() && query.Expiration.Before(blockTime) && (incycle || isBridgeDeposit) {
		nextId, err := k.keeper.QuerySequencer.Next(ctx)
		if err != nil {
			return nil, err
		}
		query.Id = nextId
		// reset query fields when generating next id
		query.HasRevealedReports = false
		query.Expiration = blockTime.Add(query.RegistrySpecTimeframe)
		err = k.keeper.Query.Set(ctx, queryId, query)
		if err != nil {
			return nil, err
		}
	}

	// if there is tip but window expired, only incycle can extend the window, otherwise requires tip vi msgTip tx
	// if tip amount is greater than zero and query timeframe is expired, it means that the query didn't have any revealed reports
	// and the tip is still there and so the time can be extended only if the query is in cycle or via a tip transaction
	// maintains the same id until the query is paid out
	if query.Amount.GT(math.ZeroInt()) && query.Expiration.Before(blockTime) && incycle || isBridgeDeposit {
		query.Expiration = blockTime.Add(query.RegistrySpecTimeframe)
		err = k.keeper.Query.Set(ctx, queryId, query)
		if err != nil {
			return nil, err
		}
	}

	// if tip is zero and not expired, this could only mean that the query is still accepting submissions
	if query.Amount.IsZero() && blockTime.Before(query.Expiration) {
		incycle = true
	}

	commit := types.Commit{
		Reporter: msg.Creator,
		QueryId:  queryId,
		Hash:     msg.Hash,
		Incycle:  incycle,
	}
	err = k.keeper.Commits.Set(ctx, collections.Join(reporterAddr.Bytes(), query.Id), commit)
	if err != nil {
		return nil, err
	}

	return &types.MsgCommitReportResponse{}, nil
}
