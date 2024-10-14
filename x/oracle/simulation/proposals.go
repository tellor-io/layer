package simulation

import (
	"fmt"
	"math/rand"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"
)

var (
	assets = []string{
		`["eth","usd"]`,
		`["btc","eth"]`,
		`["trb","btc"]`,
		`["ltc","usd"]`,
		`["bch","usd"]`,
		`["eos","eth"]`,
		`["bnb","usd"]`,
		`["usdt","usd"]`,
		`["xlm","usdt"]`,
	}
)

// SimulateMsgUpdateCyclelist returns a random MsgUpdateCyclelist
func SimulateMsgUpdateCyclelist(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	// use the default gov module account address as authority
	var authority sdk.AccAddress = address.Module("gov")
	fmt.Println("authority: ", authority.String())
	var cyclelist [][]byte
	dataspec := regtypes.DataSpec{
		AbiComponents: []*regtypes.ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currenct", FieldType: "string"},
		},
	}
	idx := r.Intn(len(assets))
	// loop a random number of times
	for i := 0; i < r.Intn(len(assets)); i++ {
		qdata, _ := dataspec.EncodeData("spotprice", assets[idx])
		cyclelist = append(cyclelist, qdata)
	}

	return &types.MsgUpdateCyclelist{
		Authority: authority.String(),
		Cyclelist: cyclelist,
	}
}

// SimulateMsgUpdateParams returns a random MsgUpdateParams
func SimulateMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	// use the default gov module account address as authority
	var authority sdk.AccAddress = address.Module("gov")

	params := types.DefaultParams()
	params.MinStakeAmount = simtypes.RandomAmount(r, math.NewInt(1000000000))

	return &types.MsgUpdateParams{
		Authority: authority.String(),
		Params:    params,
	}
}
