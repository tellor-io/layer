package keeper

import (
	"context"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) GetReportsbyQid(goCtx context.Context, req *types.QueryGetReportsbyQidRequest) (*types.QueryGetReportsbyQidResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: add index and use that
	reports := types.Reports{
		MicroReports: []*types.MicroReport{},
	}

	k.Reports.Walk(ctx, nil, func(key collections.Triple[[]byte, []byte, int64], value types.MicroReport) (stop bool, err error) {
		if value.QueryId == req.QueryId {
			reports.MicroReports = append(reports.MicroReports, &value)
		}
		return false, nil
	})

	return &types.QueryGetReportsbyQidResponse{Reports: reports}, nil
}

func (k Querier) GetReportsbyReporter(goCtx context.Context, req *types.QueryGetReportsbyReporterRequest) (*types.QueryGetReportsbyReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporter := sdk.MustAccAddressFromBech32(req.Reporter)

	// Retrieve the stored reports for the current block height.
	iter, err := k.Reports.Indexes.Reporter.MatchExact(goCtx, reporter.Bytes())
	if err != nil {
		return nil, err
	}

	reports, err := indexes.CollectValues(goCtx, k.Reports, iter)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetReportsbyReporterResponse{MicroReports: reports}, nil
}

func (k Querier) GetReportsbyReporterQid(goCtx context.Context, req *types.QueryGetReportsbyReporterQidRequest) (*types.QueryGetReportsbyQidResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporterAdd, err := sdk.AccAddressFromBech32(req.Reporter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to decode reporter address")
	}

	qId, err := utils.QueryIDFromString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to decode query ID")
	}

	microReports := []*types.MicroReport{}
	rng := collections.NewSuperPrefixedTripleRange[[]byte, []byte, int64](qId, reporterAdd.Bytes())
	k.Reports.Walk(goCtx, rng, func(key collections.Triple[[]byte, []byte, int64], value types.MicroReport) (stop bool, err error) {
		microReports = append(microReports, &value)
		return false, nil
	})

	return &types.QueryGetReportsbyQidResponse{Reports: types.Reports{
		MicroReports: microReports,
	}}, nil
}
