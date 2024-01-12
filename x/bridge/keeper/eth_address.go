package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

// UInt64Bytes uses the SDK byte marshaling to encode a uint64.
func UInt64Bytes(n uint64) []byte {
	return sdk.Uint64ToBigEndian(n)
}

func DefaultEVMAddress(addr sdk.ValAddress) gethcommon.Address {
	return gethcommon.BytesToAddress(addr)
}

// func GetValidatorAddresses(goContext context.Context, b *bridgeServer, height int64) (map[string]gethcommon.Address, error) {
// 	ctx := sdk.UnwrapSDKContext(goContext)
// 	validators := b.stakingKeeper.GetValidators(ctx, uint32(height))
// 	addresses := make(map[string]gethcommon.Address, len(validators))
// 	for _, validator := range validators {
// 		addresses[validator.GetOperator().String()] = DefaultEVMAddress(validator.GetOperator())
// 	}
// 	// print validator eth addresses to log
// 	for _, validator := range validators {
// 		b.Logger(ctx).Info("validator", "validator", validator.GetOperator(), "eth_address", DefaultEVMAddress(validator.GetOperator()).String())
// 	}

// 	return addresses, nil
// }
