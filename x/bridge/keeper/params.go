package keeper

import (
	"context"
	"time"

	"cosmossdk.io/math"
)

// GetAttestSlashPercentage returns the attestation slash percentage as a decimal
func (k Keeper) GetAttestSlashPercentage(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return math.LegacyZeroDec(), err
	}
	return params.AttestSlashPercentage, nil
}

// GetAttestRateLimitWindow returns the attestation rate limit window in milliseconds
func (k Keeper) GetAttestRateLimitWindow(ctx context.Context) (uint64, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return 0, err
	}
	return params.AttestRateLimitWindow, nil
}

// GetAttestRateLimitWindowDuration returns the attestation rate limit window as a time.Duration
func (k Keeper) GetAttestRateLimitWindowDuration(ctx context.Context) (time.Duration, error) {
	windowMillis, err := k.GetAttestRateLimitWindow(ctx)
	if err != nil {
		return 0, err
	}
	return time.Duration(windowMillis) * time.Millisecond, nil
}

// GetValsetSlashPercentage returns the validator set signature slash percentage as a decimal
func (k Keeper) GetValsetSlashPercentage(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return math.LegacyZeroDec(), err
	}
	return params.ValsetSlashPercentage, nil
}

// GetValsetRateLimitWindow returns the validator set signature rate limit window in milliseconds
func (k Keeper) GetValsetRateLimitWindow(ctx context.Context) (uint64, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return 0, err
	}
	return params.ValsetRateLimitWindow, nil
}

// GetValsetRateLimitWindowDuration returns the validator set signature rate limit window as a time.Duration
func (k Keeper) GetValsetRateLimitWindowDuration(ctx context.Context) (time.Duration, error) {
	windowMillis, err := k.GetValsetRateLimitWindow(ctx)
	if err != nil {
		return 0, err
	}
	return time.Duration(windowMillis) * time.Millisecond, nil
}

// GetAttestPenaltyTimeCutoff returns the attestation penalty time cutoff timestamp
func (k Keeper) GetAttestPenaltyTimeCutoff(ctx context.Context) (uint64, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return 0, err
	}
	return params.AttestPenaltyTimeCutoff, nil
}

// GetMainnetChainId returns the mainnet chain ID for domain separator encoding
func (k Keeper) GetMainnetChainId(ctx context.Context) (string, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return "", err
	}
	return params.MainnetChainId, nil
}
