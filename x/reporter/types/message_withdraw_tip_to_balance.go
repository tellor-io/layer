package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgWithdrawTipToBalance{}

func NewMsgWithdrawTipToBalance(selectorAddress string) *MsgWithdrawTipToBalance {
	return &MsgWithdrawTipToBalance{
		SelectorAddress: selectorAddress,
	}
}

func (msg *MsgWithdrawTipToBalance) ValidateBasic() error {
	return nil
}
