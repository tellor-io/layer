package keeper

import (
	"context"
	"encoding/hex"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CrossChainSettlementQueryType is the queryType string for settlement
// aggregates. No registry spec exists for it, so no QueryMeta,
// tip, or reporter submission can ever be created under a settlement queryId.
// TODO: block registry from creating a query with this type string.
const CrossChainSettlementQueryType = "CrossChainSettlement"

// CrossChainSettlementQueryId replicates how th settlement queryId is computed in Solidity:
//
//	keccak256(abi.encode("CrossChainSettlement", abi.encode(uint256 chainId, address contract, uint256 escrowId)))
//
// so the escrow contract can recompute the queryId of its own settlement
// report from block.chainid, address(this), and the escrow nonce.
func CrossChainSettlementQueryId(escrowChainId uint64, escrowContract []byte, escrowId uint64) (queryId, queryData []byte, err error) {
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, nil, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, nil, err
	}
	AddressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, nil, err
	}

	innerArgs := abi.Arguments{
		{Type: Uint256Type},
		{Type: AddressType},
		{Type: Uint256Type},
	}
	innerEncoded, err := innerArgs.Pack(
		new(big.Int).SetUint64(escrowChainId),
		common.BytesToAddress(escrowContract),
		new(big.Int).SetUint64(escrowId),
	)
	if err != nil {
		return nil, nil, err
	}

	outerArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryData, err = outerArgs.Pack(CrossChainSettlementQueryType, innerEncoded)
	if err != nil {
		return nil, nil, err
	}

	return crypto.Keccak256(queryData), queryData, nil
}

// EncodeCrossChainSettlementValue replicates the Solidity encoding
//
//	abi.encode(bytes32 dataQueryId, uint256 aggregateTimestampMs, uint256 dataAggregatePower, uint256 dataPreviousTimestampMs, bytes dataValue, address[] funders, uint256[] amountsLoya)
//
// which the escrow contract abi.decodes to deliver the attested data value to
// the tipper and pay funders in order — atomic delivery-versus-payment.
// dataAggregatePower and dataPreviousTimestampMs describe the DATA report so
// the delivered record carries the same fields as a regularly relayed report
// (quality and staleness signals);
func EncodeCrossChainSettlementValue(dataQueryId []byte, aggregateTimestampMs, dataAggregatePower, dataPreviousTimestampMs uint64, dataValue []byte, escrow types.CrossChainEscrow) ([]byte, error) {
	Bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	AddressArrayType, err := abi.NewType("address[]", "", nil)
	if err != nil {
		return nil, err
	}
	Uint256ArrayType, err := abi.NewType("uint256[]", "", nil)
	if err != nil {
		return nil, err
	}

	var dataQueryId32 [32]byte
	copy(dataQueryId32[:], dataQueryId)

	funders := make([]common.Address, 0, len(escrow.Funders))
	amounts := make([]*big.Int, 0, len(escrow.Funders))
	for _, f := range escrow.Funders {
		funders = append(funders, common.BytesToAddress(f.EthPayoutAddress))
		amounts = append(amounts, f.Amount.BigInt())
	}

	valueArgs := abi.Arguments{
		{Type: Bytes32Type},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: BytesType},
		{Type: AddressArrayType},
		{Type: Uint256ArrayType},
	}
	return valueArgs.Pack(dataQueryId32, new(big.Int).SetUint64(aggregateTimestampMs), new(big.Int).SetUint64(dataAggregatePower), new(big.Int).SetUint64(dataPreviousTimestampMs), dataValue, funders, amounts)
}

// AddCrossChainFunder appends a funder for later settlement and clearance.
func (k Keeper) AddCrossChainFunder(ctx context.Context, escrowKey []byte, escrow types.CrossChainEscrow, funder types.CrossChainFunder) error {
	escrow.Funders = append(escrow.Funders, funder)
	if err := k.CrossChainEscrows.Set(ctx, escrowKey, escrow); err != nil {
		return err
	}
	return k.CrossChainEscrowsByQueryId.Set(ctx, collections.Join(escrow.QueryId, escrowKey))
}

// SettleCrossChainEscrows runs in the oracle EndBlocker after
// SetAggregatedReport: for every data aggregate finalized at the current
// height, it synthesizes one settlement aggregate per pending escrow on that
// queryId. The settlement aggregate flows through the existing bridge
// snapshot/attestation (bridge EndBlocker runs after oracle's), so
// validators sign it next block and the Ethereum escrow can verify it.
func (k Keeper) SettleCrossChainEscrows(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	aggs, err := k.GetAggregatedReportsByHeight(ctx, uint64(sdkCtx.BlockHeight()))
	if err != nil {
		return err
	}
	if len(aggs) == 0 {
		return nil
	}
	blockTimeMs := uint64(sdkCtx.BlockTime().UnixMilli())
	for i := range aggs {
		rng := collections.NewPrefixedPairRange[[]byte, []byte](aggs[i].QueryId)
		iter, err := k.CrossChainEscrowsByQueryId.Iterate(ctx, rng)
		if err != nil {
			return err
		}

		keys, err := iter.Keys()
		iter.Close()
		if err != nil {
			return err
		}
		for _, key := range keys {
			escrowKey := key.K2()
			// a corrupt escrow record must not halt consensus: log and move on
			if err := k.settleOneCrossChainEscrow(ctx, escrowKey, aggs[i], blockTimeMs); err != nil {
				k.Logger(ctx).Error("cross-chain escrow settlement failed",
					"escrow_key", hex.EncodeToString(escrowKey), "error", err)
			}
		}
	}
	return nil
}

func (k Keeper) settleOneCrossChainEscrow(ctx context.Context, escrowKey []byte, dataAgg types.Aggregate, blockTimeMs uint64) error {
	escrow, err := k.CrossChainEscrows.Get(ctx, escrowKey)
	if err != nil {
		return err
	}
	if escrow.Settled {
		// stale index entry; clear it
		return k.CrossChainEscrowsByQueryId.Remove(ctx, collections.Join(escrow.QueryId, escrowKey))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	_, settlementQueryData, err := CrossChainSettlementQueryId(escrow.EscrowChainId, escrow.EscrowContract, escrow.EscrowId)
	if err != nil {
		return err
	}
	// carry the attested data value inside the settlement so one relay both
	// pays the funders and delivers the data to the escrow (the tipper's side)
	dataValue, err := hex.DecodeString(dataAgg.AggregateValue)
	if err != nil {
		return err
	}

	dataPrevTsMs := uint64(0)
	if prevTs, err := k.GetTimestampBefore(ctx, dataAgg.QueryId, sdkCtx.BlockTime()); err == nil {
		dataPrevTsMs = uint64(prevTs.UnixMilli())
	}
	value, err := EncodeCrossChainSettlementValue(dataAgg.QueryId, blockTimeMs, dataAgg.AggregatePower, dataPrevTsMs, dataValue, escrow)
	if err != nil {
		return err
	}

	totalBonded, err := k.reporterKeeper.TotalReporterPower(ctx)
	if err != nil {
		return err
	}

	settlementAgg := &types.Aggregate{
		QueryId:           escrowKey,
		AggregateValue:    hex.EncodeToString(value),
		AggregateReporter: dataAgg.AggregateReporter,
		AggregatePower:    totalBonded.Quo(layertypes.PowerReduction).Uint64(),
		Flagged:           false,
		Height:            uint64(sdkCtx.BlockHeight()),
		MicroHeight:       uint64(sdkCtx.BlockHeight()),
		MetaId:            dataAgg.MetaId,
	}
	if err := k.SetAggregate(ctx, settlementAgg, settlementQueryData, CrossChainSettlementQueryType); err != nil {
		return err
	}

	escrow.Settled = true
	escrow.SettlementTimestamp = blockTimeMs
	if err := k.CrossChainEscrows.Set(ctx, escrowKey, escrow); err != nil {
		return err
	}
	if err := k.CrossChainEscrowsByQueryId.Remove(ctx, collections.Join(escrow.QueryId, escrowKey)); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"cross_chain_settlement",
			sdk.NewAttribute("settlement_query_id", hex.EncodeToString(escrowKey)),
			sdk.NewAttribute("data_query_id", hex.EncodeToString(dataAgg.QueryId)),
			sdk.NewAttribute("timestamp", strconv.FormatUint(blockTimeMs, 10)),
			sdk.NewAttribute("funder_count", strconv.Itoa(len(escrow.Funders))),
		),
	})
	return nil
}
