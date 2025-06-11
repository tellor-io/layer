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
	// key is Bytes (reporter address) with bytes encoded max uint64 concatenated (reporterAddr...fff...)
	// timestamp is the last 8 bytes of the key so we can sort by timestamp
	buffer := make([]byte, 8)
	_, err = collections.Uint64Key.Encode(buffer, ^uint64(0))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode start value")
	}

	// construct key: reporter_address + encoded_uint64
	key := append(reporter.Bytes(), buffer...)
	rng := collections.NewPrefixUntilPairRange[[]byte, collections.Pair[[]byte, uint64]](key)
	if req.Pagination != nil && req.Pagination.Reverse {
		rng.Descending()
	}
	pairKeyCodec := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)
	if req.Pagination != nil && len(req.Pagination.Key) > 0 {
		_, startKey, err := pairKeyCodec.Decode(req.Pagination.Key)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid pagination key")
		}
		rng.StartInclusive(startKey)
	}

	iter, err := q.keeper.NoStakeReports.Indexes.Reporter.Iterate(ctx, rng)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	reports := make([]*types.NoStakeMicroReportStrings, 0)

	for ; iter.Valid(); iter.Next() {
		pk, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}

		if req.Pagination != nil && uint64(len(reports)) >= req.Pagination.Limit {
			buffer := make([]byte, pairKeyCodec.Size(pk))
			_, err = pairKeyCodec.Encode(buffer, pk)
			if err != nil {
				return nil, status.Error(codes.Internal, "failed to encode pagination key")
			}
			pageRes.NextKey = buffer
			break
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
