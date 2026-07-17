package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgCancelTipUnlock{}

func NewMsgCancelTipUnlock(selectorAddress string, unlockID uint64) *MsgCancelTipUnlock {
	return &MsgCancelTipUnlock{
		SelectorAddress: selectorAddress,
		UnlockId:        unlockID,
	}
}

func (msg *MsgCancelTipUnlock) ValidateBasic() error {
	return nil
}
