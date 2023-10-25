package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetReportsbyQid(goCtx context.Context, req *types.QueryGetReportsbyQidRequest) (*types.QueryGetReportsbyQidResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	qIdBytes, err := hex.DecodeString(req.QId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode query ID string: %v", err)
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReportsKey))
	reportsBytes := store.Get(qIdBytes)
	var reports types.Reports
	if err := k.cdc.Unmarshal(reportsBytes, &reports); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reports: %v", err)
	}
	return &types.QueryGetReportsbyQidResponse{Reports: reports}, nil
}

func (k Keeper) GetReportsbyReporter(goCtx context.Context, req *types.QueryGetReportsbyReporterRequest) (*types.QueryGetReportsbyReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReporterStoreKey))
	reporterKey := []byte(req.Reporter + ":")
	iterator := sdk.KVStorePrefixIterator(store, reporterKey)

	var finalReportsList []types.MicroReport
	// Iterate and fetch all reports for the reporter
	for ; iterator.Valid(); iterator.Next() {
		var report types.Reports
		if err := k.cdc.Unmarshal(iterator.Value(), &report); err != nil {
			return nil, status.Error(codes.InvalidArgument, "failed to unmarshal reports")
		}
		for _, microReport := range report.MicroReports {
			finalReportsList = append(finalReportsList, *microReport)
		}

	}
	return &types.QueryGetReportsbyReporterResponse{MicroReports: finalReportsList}, nil
}

func (k Keeper) GetReportsbyReporterQid(goCtx context.Context, req *types.QueryGetReportsbyReporterQidRequest) (*types.QueryGetReportsbyQidResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReporterStoreKey))
	reporterKey := []byte(req.Reporter + ":" + req.Qid)
	reportsBytes := store.Get(reporterKey)
	var reports types.Reports
	if err := k.cdc.Unmarshal(reportsBytes, &reports); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reports: %v", err)
	}
	return &types.QueryGetReportsbyQidResponse{Reports: reports}, nil
}
