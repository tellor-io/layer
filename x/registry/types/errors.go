package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/registry module sentinel errors
var (
	ErrSample        = sdkerrors.Register(ModuleName, 1100, "sample error")
	ErrInvalidSigner = sdkerrors.Register(ModuleName, 1101, "expected gov account as only signer for proposal message")
	ErrInvalidSpec   = sdkerrors.Register(ModuleName, 1102, "invalid data specification")
)
