package keeper

import (
	"bytes"
	"context"
	"errors"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// SetTwoBitReporter registers agent as the account that may submit 2bitAgentic
// reports for reporter. An empty agent clears the registration. A reporter has
// one agent; registering again replaces the previous address.
func (k Keeper) SetTwoBitReporter(ctx context.Context, reporter, agent sdk.AccAddress) error {
	exists, err := k.Reporters.Has(ctx, reporter.Bytes())
	if err != nil {
		return err
	}
	if !exists {
		return types.ErrReporterDoesNotExist.Wrapf("reporter %s does not exist", reporter.String())
	}
	if len(agent) == 0 {
		return k.clearTwoBitReporter(ctx, reporter.Bytes())
	}
	if bytes.Equal(reporter.Bytes(), agent.Bytes()) {
		return types.ErrInvalidReporter.Wrap("2bit address cannot be the reporter address")
	}
	isReporter, err := k.Reporters.Has(ctx, agent.Bytes())
	if err != nil {
		return err
	}
	if isReporter {
		return types.ErrTwoBitAgentIsReporter.Wrapf("%s is a reporter", agent.String())
	}
	return k.setTwoBitMapping(ctx, reporter.Bytes(), agent.Bytes())
}

// SetTwoBitMapping writes both 2bit indexes. The caller has already validated the pair.
// A previous agent for this reporter is deleted so the old key cannot keep reporting.
func (k Keeper) SetTwoBitMapping(ctx context.Context, reporter, agent []byte) error {
	return k.setTwoBitMapping(ctx, reporter, agent)
}

// TwoBitReporter returns the reporter registered for agent.
func (k Keeper) TwoBitReporter(ctx context.Context, agent sdk.AccAddress) (sdk.AccAddress, error) {
	reporter, err := k.TwoBitAgentToReporter.Get(ctx, agent.Bytes())
	if err != nil {
		return nil, err
	}
	return append(sdk.AccAddress(nil), reporter...), nil
}

// removeReporter deletes a reporter row and its 2bit registration.
func (k Keeper) removeReporter(ctx context.Context, reporter []byte) error {
	if err := k.clearTwoBitReporter(ctx, reporter); err != nil {
		return err
	}
	if err := k.Reporters.Remove(ctx, reporter); err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	return nil
}

func (k Keeper) setTwoBitMapping(ctx context.Context, reporter, agent []byte) error {
	reporter = append([]byte(nil), reporter...)
	agent = append([]byte(nil), agent...)

	prev, err := k.TwoBitReporterToAgent.Get(ctx, reporter)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err == nil && !bytes.Equal(prev, agent) {
		if err := k.TwoBitAgentToReporter.Remove(ctx, prev); err != nil && !errors.Is(err, collections.ErrNotFound) {
			return err
		}
	}

	owner, err := k.TwoBitAgentToReporter.Get(ctx, agent)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err == nil && !bytes.Equal(owner, reporter) {
		return types.ErrTwoBitAgentTaken.Wrapf("2bit address already registered to %s", sdk.AccAddress(owner).String())
	}

	if err := k.TwoBitAgentToReporter.Set(ctx, agent, reporter); err != nil {
		return err
	}
	return k.TwoBitReporterToAgent.Set(ctx, reporter, agent)
}

func (k Keeper) clearTwoBitReporter(ctx context.Context, reporter []byte) error {
	agent, err := k.TwoBitReporterToAgent.Get(ctx, reporter)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}
	if err := k.TwoBitAgentToReporter.Remove(ctx, agent); err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err := k.TwoBitReporterToAgent.Remove(ctx, reporter); err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	return nil
}

func (k msgServer) RegisterTwoBitReporter(goCtx context.Context, msg *types.MsgRegisterTwoBitReporter) (*types.MsgRegisterTwoBitReporterResponse, error) {
	reporter, err := sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}
	var agent sdk.AccAddress
	if msg.TwoBitAddress != "" {
		agent, err = sdk.AccAddressFromBech32(msg.TwoBitAddress)
		if err != nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid 2bit address (%s)", err)
		}
	}
	if err := k.SetTwoBitReporter(goCtx, reporter, agent); err != nil {
		return nil, err
	}

	eventType := "two_bit_reporter_registered"
	if len(agent) == 0 {
		eventType = "two_bit_reporter_cleared"
	}
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvent(sdk.NewEvent(
		eventType,
		sdk.NewAttribute("reporter", reporter.String()),
		sdk.NewAttribute("two_bit_address", agent.String()),
	))
	return &types.MsgRegisterTwoBitReporterResponse{}, nil
}
