package keeper

import (
	"context"
	"encoding/binary"

	"github.com/tellor-io/layer/x/dispute/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

func (k Querier) Disputes(ctx context.Context, req *types.QueryDisputesRequest) (*types.QueryDisputesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	disputeStore := prefix.NewStore(store, types.DisputesPrefix)
	disputes := make([]*types.Disputes, 0)
	pageRes, err := query.Paginate(disputeStore, req.Pagination, func(disputeID, value []byte) error {
		var dispute types.Dispute
		err := k.cdc.Unmarshal(value, &dispute)
		if err != nil {
			return err
		}
		id := binary.BigEndian.Uint64(disputeID)
		disputes = append(disputes, &types.Disputes{
			DisputeId: id,
			Metadata:  &dispute,
		})
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDisputesResponse{Disputes: disputes, Pagination: pageRes}, nil
}

func (k Querier) OpenDisputes(ctx context.Context, req *types.QueryOpenDisputesRequest) (*types.QueryOpenDisputesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	openDisputes, err := k.Keeper.GetOpenDisputes(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var openDisputesArray types.OpenDisputes
	openDisputesArray.Ids = openDisputes
	return &types.QueryOpenDisputesResponse{OpenDisputes: &openDisputesArray}, nil
}

func (k Querier) TeamVote(ctx context.Context, req *types.QueryTeamVoteRequest) (*types.QueryTeamVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	teamAddr, err := k.Keeper.GetTeamAddress(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	vote, err := k.Keeper.Voter.Get(ctx, collections.Join(req.DisputeId, teamAddr.Bytes()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryTeamVoteResponse{TeamVote: vote}, nil
}

func (k Querier) TeamAddress(ctx context.Context, req *types.QueryTeamAddressRequest) (*types.QueryTeamAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	teamAddr, err := k.Keeper.GetTeamAddress(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryTeamAddressResponse{TeamAddress: teamAddr.String()}, nil
}
