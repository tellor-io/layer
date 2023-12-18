package oracle

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func NewCycleListChangeProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.CycleListChangeProposal:
			return handleCycleListChangeProposal(ctx, k, c)

		default:
			return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized param proposal content type: %T", c)
		}
	}
}

func handleCycleListChangeProposal(ctx sdk.Context, k keeper.Keeper, p *types.CycleListChangeProposal) error {

	k.Logger(ctx).Info(
		fmt.Sprintf("attempt to set new cycle list: %s", p.NewList),
	)
	k.SetCycleList(ctx, p.NewList)

	return nil
}
