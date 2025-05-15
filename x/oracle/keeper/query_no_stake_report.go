package keeper

import (
	"context"
	"encoding/hex"

	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// gets all no stake reports for a reporter
// can be paginated to return a limited number of reports
func (q Querier) GetReportersNoStakeReports(ctx context.Context, req *types.QueryGetReportersNoStakeReportsRequest) (*types.QueryGetReportersNoStakeReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporter, err := sdk.AccAddressFromBech32(req.Reporter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid reporter address")
	}
	pageRes := &query.PageResponse{
		NextKey: nil,
		Total:   uint64(0),
	}
	iter, err := q.keeper.NoStakeReports.Indexes.Reporter.MatchExact(ctx, reporter)
	if err != nil {
		return nil, err
	}
	reports := make([]*types.NoStakeMicroReport, 0)
	for ; iter.Valid(); iter.Next() {
		pk, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}
		report, err := q.keeper.NoStakeReports.Get(ctx, pk)
		if err != nil {
			return nil, err
		}
		reports = append(reports, &report)

		if req.Pagination != nil && uint64(len(reports)) >= req.Pagination.Limit {
			break
		}
	}
	pageRes.Total = uint64(len(reports))

	return &types.QueryGetReportersNoStakeReportsResponse{NoStakeReports: reports, Pagination: pageRes}, nil
}

// gets all no stake reports for a query id
// can be paginated to return a limited number of reports
func (q Querier) GetNoStakeReportsByQueryId(ctx context.Context, req *types.QueryGetNoStakeReportsByQIdRequest) (*types.QueryGetNoStakeReportsByQIdResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryIdBz, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}

	pageRes := &query.PageResponse{
		NextKey: nil,
		Total:   uint64(0),
	}
	rng := collections.NewPrefixUntilPairRange[[]byte, uint64](queryIdBz)
	iter, err := q.keeper.NoStakeReports.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}
	reports := make([]*types.NoStakeMicroReport, 0)
	for ; iter.Valid(); iter.Next() {
		pk, err := iter.Key()
		if err != nil {
			return nil, err
		}
		report, err := q.keeper.NoStakeReports.Get(ctx, pk)
		if err != nil {
			return nil, err
		}
		reports = append(reports, &report)
		if req.Pagination != nil && uint64(len(reports)) >= req.Pagination.Limit {
			break
		}
	}
	pageRes.Total = uint64(len(reports))

	return &types.QueryGetNoStakeReportsByQIdResponse{NoStakeReports: reports, Pagination: pageRes}, nil
}
