package testutil

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func GenerateReports(reporters []sdk.AccAddress, values []string, powers []int64, queryId []byte) []oracletypes.MicroReport {
	var reports []oracletypes.MicroReport

	for i, reporter := range reporters {
		reports = append(reports, oracletypes.MicroReport{
			Reporter: reporter.String(),
			Value:    values[i],
			Power:    powers[i],
			QueryId:  queryId,
		})
	}
	return reports
}

func SumArray(arr []int64) int64 {
	sum := int64(0)
	for _, value := range arr {
		sum += value
	}
	return sum
}

func CalculateWeightedMean(values []int, powers []int64) float64 {
	var totalWeight, weightedSum float64
	for i, value := range values {
		weightedSum += float64(value) * float64(powers[i])
		totalWeight += float64(powers[i])
	}
	return weightedSum / totalWeight
}

func CalculateStandardDeviation(values []int, powers []int64, mean float64) float64 {
	var sum float64
	totalWeight := float64(SumArray(powers))

	for i, value := range values {
		deviation := float64(value) - mean
		sum += float64(powers[i]) * deviation * deviation
	}

	return math.Sqrt(sum / totalWeight)
}

func IntToHex(values []int) []string {
	var hexValues []string
	for _, value := range values {
		hexValues = append(hexValues, fmt.Sprintf("%x", value))
	}
	return hexValues
}
