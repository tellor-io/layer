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

// SetTwoBitReporter stores a request for agent to submit 2bitAgentic reports
// for reporter. The mapping is written when agent accepts. The request can be
// accepted through the next TwoBitPendingRequestExpiryBlocks blocks.
// Passing the reporter's own address clears the active registration and deletes
// any pending request. A reporter has one pending request. Requesting again
// deletes the previous request before writing the new one, so the previous
// address can no longer accept. The active agent stays until the new address accepts.
func (k Keeper) SetTwoBitReporter(ctx context.Context, reporter, agent sdk.AccAddress) error {
	exists, err := k.Reporters.Has(ctx, reporter.Bytes())
	if err != nil {
		return err
	}
	if !exists {
		return types.ErrReporterDoesNotExist.Wrapf("reporter %s does not exist", reporter.String())
	}
	if len(agent) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "2bit address cannot be empty")
	}
	// Own address clears. This has to come before the "agent is a reporter" check,
	// because the reporter address is itself a reporter.
	if bytes.Equal(reporter.Bytes(), agent.Bytes()) {
		if err := k.clearTwoBitReporter(ctx, reporter.Bytes()); err != nil {
			return err
		}
		return k.clearTwoBitRequest(ctx, reporter.Bytes())
	}
	isReporter, err := k.Reporters.Has(ctx, agent.Bytes())
	if err != nil {
		return err
	}
	if isReporter {
		return types.ErrTwoBitAgentIsReporter.Wrapf("%s is a reporter", agent.String())
	}
	owner, err := k.TwoBitAgentToReporter.Get(ctx, agent.Bytes())
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err == nil && !bytes.Equal(owner, reporter.Bytes()) {
		return types.ErrTwoBitAgentTaken.Wrapf("2bit address already registered to %s", sdk.AccAddress(owner).String())
	}
	// Delete the previous invite before writing the new one. One slot per reporter,
	// so the address named in the old request can no longer accept it.
	if err := k.clearTwoBitRequest(ctx, reporter.Bytes()); err != nil {
		return err
	}
	expiry, err := twoBitRequestExpiryHeight(ctx)
	if err != nil {
		return err
	}
	return k.TwoBitPendingRequests.Set(ctx, append([]byte(nil), reporter.Bytes()...), types.TwoBitPendingRequest{
		TwoBitReporter: append([]byte(nil), agent.Bytes()...),
		ExpiryHeight:   expiry,
	})
}

// AcceptTwoBitRegistration writes the 2bit mapping for reporter's pending request.
// agent must be the account named in that request.
func (k Keeper) AcceptTwoBitRegistration(ctx context.Context, reporter, agent sdk.AccAddress) error {
	if len(reporter) == 0 || len(agent) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "2bit address cannot be empty")
	}
	exists, err := k.Reporters.Has(ctx, reporter.Bytes())
	if err != nil {
		return err
	}
	if !exists {
		return types.ErrReporterDoesNotExist.Wrapf("reporter %s does not exist", reporter.String())
	}
	pending, err := k.TwoBitPendingRequests.Get(ctx, reporter.Bytes())
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.ErrTwoBitRegistrationNotFound.Wrapf("reporter %s has no 2bit registration request", reporter.String())
		}
		return err
	}
	if !bytes.Equal(pending.TwoBitReporter, agent.Bytes()) {
		return types.ErrTwoBitRegistrationNotFound.Wrapf("reporter %s did not request %s", reporter.String(), agent.String())
	}
	height, err := twoBitBlockHeight(ctx)
	if err != nil {
		return err
	}
	if height > pending.ExpiryHeight {
		return types.ErrTwoBitRegistrationExpired.Wrapf("reporter %s request for %s expired at height %d", reporter.String(), agent.String(), pending.ExpiryHeight)
	}
	isReporter, err := k.Reporters.Has(ctx, agent.Bytes())
	if err != nil {
		return err
	}
	if isReporter {
		return types.ErrTwoBitAgentIsReporter.Wrapf("%s is a reporter", agent.String())
	}
	if err := k.setTwoBitMapping(ctx, reporter.Bytes(), agent.Bytes()); err != nil {
		return err
	}
	return k.clearTwoBitRequest(ctx, reporter.Bytes())
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

// removeReporter deletes a reporter row, its 2bit registration, and any pending request.
func (k Keeper) removeReporter(ctx context.Context, reporter []byte) error {
	if err := k.clearTwoBitReporter(ctx, reporter); err != nil {
		return err
	}
	if err := k.clearTwoBitRequest(ctx, reporter); err != nil {
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

func (k Keeper) clearTwoBitRequest(ctx context.Context, reporter []byte) error {
	if err := k.TwoBitPendingRequests.Remove(ctx, reporter); err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	return nil
}

func twoBitBlockHeight(ctx context.Context) (uint64, error) {
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if height < 0 {
		return 0, errors.New("negative block height")
	}
	return uint64(height), nil
}

// twoBitRequestExpiryHeight is the last height at which a request made in this block can be accepted.
func twoBitRequestExpiryHeight(ctx context.Context) (uint64, error) {
	height, err := twoBitBlockHeight(ctx)
	if err != nil {
		return 0, err
	}
	return height + types.TwoBitPendingRequestExpiryBlocks, nil
}
