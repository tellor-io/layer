package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitValue{}, "oracle/SubmitValue", nil)
	cdc.RegisterConcrete(&MsgCommitReport{}, "oracle/CommitReport", nil)
	cdc.RegisterConcrete(&MsgTip{}, "oracle/Tip", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitValue{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCommitReport{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgTip{},
	)
	registry.RegisterImplementations((*govtypes.Content)(nil),
		&CycleListChangeProposal{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
