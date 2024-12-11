package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	regTypes "github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetValue sets the value for a query and stores the report.
// 1. Decode queryData to get query type string
// 2. Get data spec from registry by query type
// 3. Soft validate value using value type from data spec
// 4. Create a new MicroReport object with the provided data
// 5. Set query.HasRevealedReports to true
// 6. Set the query in the store
// 7. Check if the new value changes the aggregate value and update it if needed
// 7. Emit a new_report event
// 8. Set the micro report in the store
func (k Keeper) SetValue(ctx context.Context, reporter sdk.AccAddress, query types.QueryMeta, val string, queryData []byte, power uint64, incycle bool) error {
	queryId := utils.QueryIDFromData(queryData)
	alreadyReported, err := k.Reports.Has(ctx, collections.Join3(queryId, reporter.Bytes(), query.Id))
	if err != nil {
		return err
	}
	if alreadyReported {
		return status.Error(codes.InvalidArgument, "reporter has already submitted a report for this query")
	}
	// decode query data hex to get query type, returns interface array
	queryType, _, err := regTypes.DecodeQueryType(queryData)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query type: %v", err))
	}
	dataSpec, err := k.GetDataSpec(ctx, queryType)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to get value type: %v", err))
	}
	// decode value using value type from data spec and check if decodes successfully
	// value is not used, only used to check if it decodes successfully
	if err := dataSpec.ValidateValue(val); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to validate value: %v", err))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	report := types.MicroReport{
		Reporter:        reporter.String(),
		Power:           power,
		QueryType:       queryType,
		QueryId:         queryId,
		Value:           val,
		AggregateMethod: dataSpec.AggregationMethod,
		Timestamp:       sdkCtx.BlockTime(),
		Cyclelist:       incycle,
		BlockNumber:     uint64(sdkCtx.BlockHeight()),
	}

	query.HasRevealedReports = true
	err = k.Query.Set(ctx, collections.Join(queryId, query.Id), query)
	if err != nil {
		return err
	}
	if dataSpec.AggregationMethod == "weighted-median" {
		err = k.AddReport(ctx, query.Id, report)
		if err != nil {
			return fmt.Errorf("failed to add report: %w", err)
		}
	} else {
		err = k.AddReportWeightedMode(ctx, query.Id, report)
		if err != nil {
			return fmt.Errorf("failed to add report: %w", err)
		}
	}
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"new_report",
			sdk.NewAttribute("reporter", reporter.String()),
			sdk.NewAttribute("reporter_power", fmt.Sprintf("%d", power)),
			sdk.NewAttribute("query_type", queryType),
			sdk.NewAttribute("query_id", hex.EncodeToString(queryId)),
			sdk.NewAttribute("value", val),
			sdk.NewAttribute("cyclelist", fmt.Sprintf("%t", incycle)),
			sdk.NewAttribute("aggregate_method", dataSpec.AggregationMethod),
			sdk.NewAttribute("query_data", hex.EncodeToString(queryData)),
		),
	})
	return k.Reports.Set(ctx, collections.Join3(queryId, reporter.Bytes(), query.Id), report)
}

func (k Keeper) GetDataSpec(ctx context.Context, queryType string) (regTypes.DataSpec, error) {
	// get data spec from registry by query type to validate value
	dataSpec, err := k.registryKeeper.GetSpec(ctx, queryType)
	if err != nil {
		return regTypes.DataSpec{}, err
	}
	return dataSpec, nil
}

func (k Keeper) AddReport(ctx context.Context, id uint64, report types.MicroReport) error {
	// normalize value for accurate comparison and storage
	value, err := utils.FormatUint256(report.Value)
	if err != nil {
		return fmt.Errorf("failed to format value: %w", err)
	}
	power := report.Power
	// check if same value is already reported
	existingValue, err := k.Values.Get(ctx, collections.Join(id, value))
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf("failed to get existing value: %w", err)
	}
	existingValue.CrossoverWeight += power
	existingValue.MicroReport = &report

	if err := k.Values.Set(ctx, collections.Join(id, value), existingValue); err != nil {
		return fmt.Errorf("failed to update valuesMap: %w", err)
	}
	// update total power
	totalPower, err := k.ValuesWeightSum.Get(ctx, id)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return fmt.Errorf("failed to get total power: %w", err)
		}
		totalPower = 0
	}
	totalPower += power
	if err := k.ValuesWeightSum.Set(ctx, id, totalPower); err != nil {
		return fmt.Errorf("failed to update total power: %w", err)
	}
	halfTotal := totalPower / 2
	// get current median
	currentMedian, err := k.AggregateValue.Get(ctx, id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// If no median exists, set the new value as the initial median
			if err := k.AggregateValue.Set(ctx, id, types.RunningAggregate{Value: value, CrossoverWeight: power}); err != nil {
				return fmt.Errorf("failed to set initial median: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get current median: %w", err)
	}
	crossoverPower := currentMedian.CrossoverWeight
	if value < currentMedian.Value {
		// If the new value is smaller, update cumulative weight
		crossoverPower += power
		if crossoverPower >= halfTotal {
			// Median might shift left, perform left traversal
			return k.TraverseLeft(ctx, id, currentMedian.Value, crossoverPower, halfTotal)
		}
	} else if value > currentMedian.Value {
		// If the new value is larger, perform right traversal only if needed
		if crossoverPower < halfTotal {
			return k.TraverseRight(ctx, id, currentMedian.Value, crossoverPower, halfTotal)
		}
	} else {
		// If the new value is equal to the current median, update cumulative weight
		crossoverPower += power
		return k.AggregateValue.Set(ctx, id, types.RunningAggregate{Value: currentMedian.Value, CrossoverWeight: crossoverPower})
	}

	// No changes needed if crossover weight remains valid
	return nil
}

func (k Keeper) TraverseLeft(
	ctx context.Context,
	id uint64, medianValue string, crossoverWeight, halfTotal uint64,
) error {
	currentValue := medianValue
	rng := collections.NewPrefixedPairRange[uint64, string](id).EndInclusive(medianValue).Descending() // inclusive to get the current median value's actual power
	iter, err := k.Values.Iterate(ctx, rng)
	if err != nil {
		return err
	}

	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		leftkey, err := iter.Key()
		if err != nil {
			return err
		}
		leftValue, err := k.Values.Get(ctx, leftkey)
		if err != nil {
			return err
		}
		leftPower := leftValue.CrossoverWeight
		if leftkey.K2() == currentValue {
			crossoverWeight -= leftPower
			// check if you should keep going left
			if crossoverWeight < halfTotal {
				// don't go anymore left
				// return the median
				// put the power back
				crossoverWeight += leftPower
				return k.AggregateValue.Set(ctx, id, types.RunningAggregate{CrossoverWeight: crossoverWeight, Value: medianValue})
			}
			continue
		}

		// you reduce by this item's power to see if you should keep going left
		crossoverWeight -= leftPower

		if crossoverWeight < halfTotal {
			// don't go anymore left
			// put the power back
			crossoverWeight += leftPower
			return k.AggregateValue.Set(ctx, id, types.RunningAggregate{CrossoverWeight: crossoverWeight, Value: leftkey.K2()})
		}
	}

	return k.AggregateValue.Set(ctx, id, types.RunningAggregate{CrossoverWeight: crossoverWeight, Value: medianValue})
}

func (k Keeper) TraverseRight(
	ctx context.Context,
	id uint64, medianValue string, crossoverWeight, halfTotal uint64,
) error {
	incomingWeight := crossoverWeight
	rng := collections.NewPrefixedPairRange[uint64, string](id).StartExclusive(medianValue)
	iter, err := k.Values.Iterate(ctx, rng)
	if err != nil {
		return err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		rightkey, err := iter.Key()
		if err != nil {
			return err
		}
		rightValue, err := k.Values.Get(ctx, rightkey)
		if err != nil {
			return err
		}
		rightPower := rightValue.CrossoverWeight
		crossoverWeight += rightPower

		if crossoverWeight < halfTotal {
			medianValue = rightkey.K2()
		} else {
			if incomingWeight == crossoverWeight {
				crossoverWeight = incomingWeight
				break
			}
			medianValue = rightkey.K2()
			break
		}
	}

	return k.AggregateValue.Set(ctx, id, types.RunningAggregate{CrossoverWeight: crossoverWeight, Value: medianValue})
}

func (k Keeper) AddReportWeightedMode(ctx context.Context, id uint64, report types.MicroReport) error {
	value := report.Value
	power := report.Power
	existingValue, err := k.Values.Get(ctx, collections.Join(id, value))
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf("failed to get existing value: %w", err)
	}
	existingValue.CrossoverWeight += power
	existingValue.MicroReport = &report
	if err := k.Values.Set(ctx, collections.Join(id, value), existingValue); err != nil {
		return fmt.Errorf("failed to update valuesMap: %w", err)
	}
	// update total power
	totalPower, err := k.ValuesWeightSum.Get(ctx, id)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return fmt.Errorf("failed to get total power: %w", err)
		}
		totalPower = 0
	}
	totalPower += power
	if err := k.ValuesWeightSum.Set(ctx, id, totalPower); err != nil {
		return fmt.Errorf("failed to update total power: %w", err)
	}
	// get current median
	currentMedian, err := k.AggregateValue.Get(ctx, id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// If no median exists, set the new value as the initial median
			if err := k.AggregateValue.Set(ctx, id, types.RunningAggregate{Value: value, CrossoverWeight: power}); err != nil {
				return fmt.Errorf("failed to set initial median: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get current median: %w", err)
	}
	// check if the new value power is greater than the current median power
	// if it is not do nothing and return else update the median
	if currentMedian.CrossoverWeight > power {
		return nil
	}
	rng := collections.NewPrefixedPairRange[collections.Pair[uint64, uint64], collections.Pair[uint64, string]](collections.Join(id, uint64(math.MaxUint64))).Descending()
	iter, err := k.Values.Indexes.Power.Iterate(ctx, rng)
	if err != nil {
		return err
	}
	defer iter.Close()
	if iter.Valid() {
		pk, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		maxValue, err := k.Values.Get(ctx, pk)
		if err != nil {
			return err
		}
		return k.AggregateValue.Set(ctx, id, types.RunningAggregate{Value: maxValue.MicroReport.Value, CrossoverWeight: maxValue.CrossoverWeight})
	}
	return nil
}
