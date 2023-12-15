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

func NewSupportedQueryChangeProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.SupportedQueryChangeProposal:
			return handleSupportedQueryChangeProposal(ctx, k, c)

		default:
			return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized param proposal content type: %T", c)
		}
	}
}

func handleSupportedQueryChangeProposal(ctx sdk.Context, k keeper.Keeper, p *types.SupportedQueryChangeProposal) error {

	k.Logger(ctx).Info(
		fmt.Sprintf("attempt to set new supported query; query: %s", p.Changes),
	)
	k.SetSupportedQueries(ctx, p.Changes)

	return nil
}
