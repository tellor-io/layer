package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(mint cdctypes.InterfaceRegistry) {
	mint.RegisterImplementations((*sdk.Msg)(nil),
		&MsgInit{},
	)
	// this line is used by starport scaffolding # 3
	msgservice.RegisterMsgServiceDesc(mint, &_Msg_serviceDesc)
}
