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

	q.keeper.Logger(ctx).Info("GetReportersNoStakeReports req.Reporter: ", req.Reporter)
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
	reports := make([]*types.NoStakeMicroReportStrings, 0)
	for ; iter.Valid(); iter.Next() {
		pk, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}
		report, err := q.keeper.NoStakeReports.Get(ctx, pk)
		if err != nil {
			return nil, err
		}
		stringReport := types.NoStakeMicroReportStrings{
			Reporter:    sdk.AccAddress(report.Reporter).String(),
			Value:       report.Value,
			Timestamp:   uint64(report.Timestamp.UnixMilli()),
			BlockNumber: report.BlockNumber,
		}
		reports = append(reports, &stringReport)
		if req.Pagination != nil && uint64(len(reports)) >= req.Pagination.Limit {
			break
		}
	}
	pageRes.Total = uint64(len(reports))

	return &types.QueryGetReportersNoStakeReportsResponse{NoStakeReports: reports, Pagination: pageRes}, nil
}

// gets all no stake reports for a query id
// can be paginated to return a limited number of reports
func (q Querier) GetNoStakeReportsByQueryId(ctx context.Context, req *types.QueryGetNoStakeReportsByQueryIdRequest) (*types.QueryGetNoStakeReportsByQueryIdResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryIdBz, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}

	microreports := make([]*types.NoStakeMicroReportStrings, 0)
	_, pageRes, err := query.CollectionPaginate(
		ctx, q.keeper.NoStakeReports, req.Pagination, func(_ collections.Pair[[]byte, uint64], report types.NoStakeMicroReport) (types.NoStakeMicroReport, error) {
			microReport := types.NoStakeMicroReportStrings{
				Reporter:    sdk.AccAddress(report.Reporter).String(),
				Value:       report.Value,
				Timestamp:   uint64(report.Timestamp.UnixMilli()),
				BlockNumber: report.BlockNumber,
			}
			microreports = append(microreports, &microReport)
			return report, nil
		}, query.WithCollectionPaginationPairPrefix[[]byte, uint64](queryIdBz),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetNoStakeReportsByQueryIdResponse{NoStakeReports: microreports, Pagination: pageRes}, nil
}
