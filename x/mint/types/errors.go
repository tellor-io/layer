package types

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrInvalidSigner      = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrAlreadyInitialized = sdkerrors.Register(ModuleName, 1101, "already initialized")
	ErrInvalidRequest     = sdkerrors.Register(ModuleName, 1102, "invalid request")
)
