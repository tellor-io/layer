package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitValue{}, "oracle/SubmitValue", nil)
	cdc.RegisterConcrete(&MsgBatchSubmitValue{}, "oracle/BatchSubmitValue", nil)
	cdc.RegisterConcrete(&MsgTip{}, "oracle/Tip", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "oracle/UpdateParams", nil)
	cdc.RegisterConcrete(&MsgUpdateCyclelist{}, "oracle/UpdateCyclelist", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitValue{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgBatchSubmitValue{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgTip{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateCyclelist{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
