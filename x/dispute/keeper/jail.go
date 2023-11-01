package keeper

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
)

func (k Keeper) JailValidatorUntil(ctx sdk.Context, valAddr sdk.ValAddress, jailDuration int64) error {
	val := k.stakingKeeper.Validator(ctx, valAddr)
	if val == nil {
		return fmt.Errorf("validator %s does not exist", valAddr)
	}
	consAddr, err := val.GetConsAddr()
	if err != nil {
		k.Logger(ctx).Error("error getting consensus address", "error", err)
		// panic(err)
		return err
	}

	var signingInfo slashingtypes.ValidatorSigningInfo
	signingInfo, ok := k.slashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	if !ok {
		signingInfo.Address = consAddr.String()
		signingInfo.JailedUntil = ctx.BlockTime().Add(time.Second * time.Duration(jailDuration))
	} else {
		signingInfo.JailedUntil = ctx.BlockTime().Add(time.Second * time.Duration(jailDuration))
	}
	k.slashingKeeper.SetValidatorSigningInfo(ctx, consAddr, signingInfo)
	k.stakingKeeper.Jail(ctx, consAddr)
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"jailed_validator",
			sdk.NewAttribute("validator", valAddr.String()),
			sdk.NewAttribute("duration", strconv.FormatInt(jailDuration, 10)),
		),
	})
	return nil
}
