package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

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
// 7. Emit a new_report event
// 8. Set the micro report in the store
func (k Keeper) SetValue(ctx context.Context, reporter sdk.AccAddress, query types.QueryMeta, val string, queryData []byte, power uint64, incycle bool) error {
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
	queryId := utils.QueryIDFromData(queryData)
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
		if err == nil {
			fmt.Println("report added successfully!!!")
		}
		if err != nil {
			fmt.Printf("failed to add report: %v", err)
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
	// check if same value is already reported
	value := report.Value
	fmt.Println("value", value)
	power := report.Power
	existingValue, err := k.Values.Get(ctx, collections.Join(id, value))
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf("failed to get existing value: %w", err)
	}
	existingValue.CumulativePower += power
	existingValue.Report = &report
	// todo: update reporter
	newPower := existingValue.CumulativePower + power

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
	currentMedian, err := k.Median.Get(ctx, id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			fmt.Println("no median found")
			// If no median exists, set the new value as the initial median
			if err := k.Median.Set(ctx, id, types.Median{Value: value, CrossoverWeight: newPower}); err != nil {
				return fmt.Errorf("failed to set initial median: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get current median: %w", err)
	}
	cumulativeWeight := currentMedian.CrossoverWeight
	if value < currentMedian.Value {
		// If the new value is smaller, update cumulative weight
		cumulativeWeight += power
		fmt.Println("new cumulative weight", cumulativeWeight)
		if cumulativeWeight >= halfTotal {
			// Median might shift left, perform left traversal
			return k.TraverseLeft(ctx, id, currentMedian.Value, cumulativeWeight, halfTotal)
		}
	} else if value > currentMedian.Value {
		// If the new value is larger, perform right traversal only if needed
		if cumulativeWeight < halfTotal {
			return k.TraverseRight(ctx, id, currentMedian.Value, cumulativeWeight, halfTotal)
		}
		return k.TraverseLeft(ctx, id, currentMedian.Value, cumulativeWeight, halfTotal)
	} else {
		// If the new value is equal to the current median, update cumulative weight
		cumulativeWeight += power
		return k.Median.Set(ctx, id, types.Median{Value: currentMedian.Value, CrossoverWeight: cumulativeWeight})
	}

	// No changes needed if crossover weight remains valid
	return nil
}

func (k Keeper) TraverseLeft(
	ctx context.Context,
	// valuesMap collections.Map[collections.Pair[uint64, string], uint64],
	id uint64, medianValue string, crossoverWeight, halfTotal uint64,
	//  cweightMap collections.Map[uint64, types.Median]
) error {
	fmt.Println("traversing left")
	currentValue := medianValue
	rng := collections.NewPrefixedPairRange[uint64, string](id).EndInclusive(medianValue).Descending() // inclusive to get the current median value's actual power
	iter, err := k.Values.Iterate(ctx, rng)
	if err != nil {
		return err
	}

	// [1,2,3,4,5,6]
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
		leftPower := leftValue.CumulativePower
		if leftkey.K2() == currentValue {
			fmt.Println("cross over weight after current value subtraction", crossoverWeight-leftPower, halfTotal)
			crossoverWeight -= leftPower
			// check if you should keep going left
			if crossoverWeight < halfTotal {
				// don't go anymore left
				// return the median
				return k.Median.Set(ctx, id, types.Median{CrossoverWeight: crossoverWeight, Value: medianValue})
			}
			continue
		}
		fmt.Println("cross over weight", crossoverWeight)
		// you reduce by this item's power to see if you should keep going left
		crossoverWeight -= leftPower
		fmt.Println("new left crossover weight", crossoverWeight)
		if crossoverWeight < halfTotal {
			// don't go anymore left
			// put the power back
			crossoverWeight += leftPower
			return k.Median.Set(ctx, id, types.Median{CrossoverWeight: crossoverWeight, Value: leftkey.K2()})
		}
	}
	fmt.Println("transversing left", crossoverWeight, medianValue)
	return k.Median.Set(ctx, id, types.Median{CrossoverWeight: crossoverWeight, Value: medianValue})
}

func (k Keeper) TraverseRight(
	ctx context.Context,
	// valuesMap collections.Map[collections.Pair[uint64, string], uint64],
	id uint64, medianValue string, crossoverWeight, halfTotal uint64,
	// cweightMap collections.Map[uint64, types.Median]
) error {
	fmt.Println("traversing right")
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
		rightPower := rightValue.CumulativePower
		crossoverWeight += rightPower
		fmt.Println("new right crossover weight", crossoverWeight, rightkey.K2())
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
	fmt.Println("is value changing", crossoverWeight, medianValue)
	return k.Median.Set(ctx, id, types.Median{CrossoverWeight: crossoverWeight, Value: medianValue})
}
