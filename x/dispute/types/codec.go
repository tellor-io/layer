package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgProposeDispute{}, "dispute/ProposeDispute", nil)
	cdc.RegisterConcrete(&MsgAddFeeToDispute{}, "dispute/AddFeeToDispute", nil)
	cdc.RegisterConcrete(&MsgVote{}, "dispute/Vote", nil)
	cdc.RegisterConcrete(&MsgWithdrawFeeRefund{}, "dispute/WithdrawFeeRefund", nil)
	cdc.RegisterConcrete(&MsgClaimReward{}, "dispute/ClaimReward", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgProposeDispute{},
		&MsgAddFeeToDispute{},
		&MsgVote{},
		&MsgUpdateTeam{},
		&MsgWithdrawFeeRefund{},
		&MsgClaimReward{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
