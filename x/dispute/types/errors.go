package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/dispute module sentinel errors
var (
	INSUFFICIENT_BALANCE    = sdkerrors.Register(ModuleName, 2, "insufficient balance")
	ErrDisputeDoesNotExist  = sdkerrors.Register(ModuleName, 3, "dispute not found")
	ErrDisputeTimeExpired   = sdkerrors.Register(ModuleName, 4, "dispute time expired")
	ErrDisputeFeeAlreadyMet = sdkerrors.Register(ModuleName, 5, "dispute fee already met")
)
