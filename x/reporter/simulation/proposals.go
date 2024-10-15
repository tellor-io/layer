package simulation

import (
	"math/rand"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// SimulateMsgUpdateParams returns a random MsgUpdateParams
func SimulateMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	// use the default gov module account address as authority
	var authority sdk.AccAddress = address.Module("gov")

	params := types.DefaultParams()
	params.MaxSelectors = r.Uint64()
	params.MinCommissionRate = simtypes.RandomDecAmount(r, math.LegacyOneDec())
	params.MinTrb = simtypes.RandomAmount(r, math.NewInt(1000000000))

	return &types.MsgUpdateParams{
		Authority: authority.String(),
		Params:    params,
	}
}
