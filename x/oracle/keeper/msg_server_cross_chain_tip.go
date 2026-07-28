package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strconv"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CrossChainTip fronts a tip on Layer on behalf of an Ethereum escrow tip.
// It behaves exactly like Tip (same validation, 2% burn, QueryMeta lifecycle)
// and additionally records the funder's Ethereum payout address and gross
// amount under the escrow's settlement queryId, to be repaid from the
// Ethereum escrow once the data aggregate is settled and attested.
func (k msgServer) CrossChainTip(goCtx context.Context, msg *types.MsgCrossChainTip) (*types.MsgCrossChainTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	tipper, err := sdk.AccAddressFromBech32(msg.Tipper)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid tipper address (%s)", err)
	}
	if err := validateTipFields(msg.Amount, msg.QueryData); err != nil {
		return nil, err
	}
	if len(msg.EscrowContract) != 20 {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow contract must be a 20-byte EVM address")
	}
	if len(msg.EthPayoutAddress) != 20 {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "eth payout address must be a 20-byte EVM address")
	}
	if msg.EscrowTotal.IsNil() || msg.EscrowTotal.LT(msg.Amount.Amount) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow total must be at least the tip amount")
	}

	escrowKey, _, err := CrossChainSettlementQueryId(msg.EscrowChainId, msg.EscrowContract, msg.EscrowId)
	if err != nil {
		return nil, err
	}
	queryId := utils.QueryIDFromData(msg.QueryData)

	escrow, err := k.keeper.CrossChainEscrows.Get(ctx, escrowKey)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
		// first funder escrow record
		escrow = types.CrossChainEscrow{
			QueryId:        queryId,
			EscrowChainId:  msg.EscrowChainId,
			EscrowContract: msg.EscrowContract,
			EscrowId:       msg.EscrowId,
			DeclaredTotal:  msg.EscrowTotal,
			Funders:        nil,
			Settled:        false,
		}
	} else {
		if escrow.Settled {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow %s already settled", hex.EncodeToString(escrowKey))
		}
		if !bytes.Equal(escrow.QueryId, queryId) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query data does not match escrow's bound query id %s", hex.EncodeToString(escrow.QueryId))
		}
		if !escrow.DeclaredTotal.Equal(msg.EscrowTotal) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow total %s does not match declared total %s", msg.EscrowTotal, escrow.DeclaredTotal)
		}
	}

	// reject if funding cap reached
	funded := math.ZeroInt()
	for _, f := range escrow.Funders {
		funded = funded.Add(f.Amount)
	}
	if funded.Add(msg.Amount.Amount).GT(escrow.DeclaredTotal) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"tip would exceed escrow funding cap: funded %s + amount %s > declared total %s",
			funded, msg.Amount.Amount, escrow.DeclaredTotal)
	}

	// normal tip flow
	tip, _, query, err := k.keeper.processTipCore(ctx, tipper, msg.QueryData, msg.Amount)
	if err != nil {
		return nil, err
	}

	// record the funder gross (pre-burn) so they are made whole on Ethereum
	funder := types.CrossChainFunder{
		EthPayoutAddress: msg.EthPayoutAddress,
		Amount:           msg.Amount.Amount,
	}
	if err := k.keeper.AddCrossChainFunder(ctx, escrowKey, escrow, funder); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"cross_chain_tip",
			sdk.NewAttribute("settlement_query_id", hex.EncodeToString(escrowKey)),
			sdk.NewAttribute("query_id", hex.EncodeToString(queryId)),
			sdk.NewAttribute("tipper", tipper.String()),
			sdk.NewAttribute("eth_payout_address", hex.EncodeToString(msg.EthPayoutAddress)),
			sdk.NewAttribute("amount_gross", msg.Amount.Amount.String()),
			sdk.NewAttribute("amount_net", tip.Amount.String()),
			sdk.NewAttribute("querymeta_id", strconv.Itoa(int(query.Id))),
		),
	})

	return &types.MsgCrossChainTipResponse{SettlementQueryId: hex.EncodeToString(escrowKey)}, nil
}
