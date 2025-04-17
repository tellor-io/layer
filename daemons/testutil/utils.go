package testutil

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GenerateReports(reporters []sdk.AccAddress, values []string, powers []uint64, queryId []byte) []oracletypes.MicroReport {
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

func EncodeValue(number float64) string {
	strNumber := fmt.Sprintf("%.18f", number)

	parts := strings.Split(strNumber, ".")
	if len(parts[1]) > 18 {
		parts[1] = parts[1][:18]
	}
	truncatedStr := parts[0] + parts[1]

	bigIntNumber := new(big.Int)
	bigIntNumber.SetString(truncatedStr, 10)

	uint256ABIType, _ := abi.NewType("uint256", "", nil)

	arguments := abi.Arguments{{Type: uint256ABIType}}
	encodedBytes, _ := arguments.Pack(bigIntNumber)

	encodedString := hex.EncodeToString(encodedBytes)
	return encodedString
}

// EncodeStringValue encodes a string for Ethereum ABI
func EncodeStringValue(value string) string {
	// Create a string ABI type
	stringABIType, _ := abi.NewType("string", "", nil)

	// Create the arguments and pack the string
	arguments := abi.Arguments{{Type: stringABIType}}
	encodedBytes, _ := arguments.Pack(value)

	// Convert to hex string
	encodedString := hex.EncodeToString(encodedBytes)
	return encodedString
}
