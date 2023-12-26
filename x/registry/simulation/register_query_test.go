package simulation_test

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/codec"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

type SimTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	accountKeeper types.AccountKeeper
	bankKeeper    keeper.Keeper
	cdc           codec.Codec
	txConfig      client.TxConfig
	app           *runtime.App
}

