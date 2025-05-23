package customquery

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
)

// AbiNumberEncoder encodes float64 values to any Ethereum ABI numeric type
func AbiNumberEncoder(value float64, abiType string) (string, error) {
	// Parse the ABI type
	typeInfo, err := parseAbiType(abiType)
	if err != nil {
		return "", err
	}

	// Convert to exact representation using big.Rational
	rat, err := floatToScaledRat(value, typeInfo.decimals)
	if err != nil {
		return "", err
	}

	// Convert to integer
	intVal := new(big.Int)
	intVal.Div(rat.Num(), rat.Denom())

	// Range check
	if err := validateRange(intVal, typeInfo); err != nil {
		return "", err
	}

	// Handle negative values for signed types
	if typeInfo.signed && intVal.Sign() < 0 {
		// Two's complement for negative numbers
		// 2^bitSize + value
		twosPower := new(big.Int).Lsh(big.NewInt(1), uint(typeInfo.bitSize))
		intVal.Add(intVal, twosPower)
	}

	// Convert to hex with proper padding
	return intToHex(intVal, typeInfo.bitSize/8), nil // bitSize is in bits, so divide by 8 for bytes
}

type abiTypeInfo struct {
	category string // "int", "uint", "fixed", "ufixed"
	bitSize  int    // 8, 16, 24, ..., 256
	decimals int
	signed   bool // true for int/fixed, false for uint/ufixed
}

// Parse ABI type string into struct abiTypeInfo
func parseAbiType(abiType string) (abiTypeInfo, error) {
	// Define patterns for different types
	intPattern := regexp.MustCompile(`^(u?)int(\d+)$`)
	fixedPattern := regexp.MustCompile(`^(u?)fixed(\d+)x(\d+)$`)

	var info abiTypeInfo

	// Check if it's an integer type
	if matches := intPattern.FindStringSubmatch(abiType); len(matches) > 0 {
		info.category = "int"
		if matches[1] == "u" {
			info.category = "uint"
			info.signed = false
		} else {
			info.signed = true
		}

		bitSize, err := strconv.Atoi(matches[2])
		if err != nil {
			return info, fmt.Errorf("invalid bit size: %s", matches[2])
		}
		info.bitSize = bitSize
		info.decimals = 0
	} else if matches := fixedPattern.FindStringSubmatch(abiType); len(matches) > 0 {
		// It's a fixed-point type
		info.category = "fixed"
		if matches[1] == "u" {
			info.category = "ufixed"
			info.signed = false
		} else {
			info.signed = true
		}

		bitSize, err := strconv.Atoi(matches[2])
		if err != nil {
			return info, fmt.Errorf("invalid bit size: %s", matches[2])
		}
		info.bitSize = bitSize

		decimals, err := strconv.Atoi(matches[3])
		if err != nil {
			return info, fmt.Errorf("invalid decimals: %s", matches[3])
		}
		info.decimals = decimals
	} else {
		return info, fmt.Errorf("unsupported ABI type: %s", abiType)
	}

	// Validate bit size
	if info.bitSize <= 0 || info.bitSize > 256 || info.bitSize%8 != 0 {
		return info, fmt.Errorf("invalid bit size: %d (must be multiple of 8, ≤ 256)", info.bitSize)
	}

	return info, nil
}

// Convert float64 to scaled big.Rat
func floatToScaledRat(value float64, decimals int) (*big.Rat, error) {
	// Convert to string first to preserve precision, for some reason it's off slightly
	// when using the float64 directly
	valueStr := strconv.FormatFloat(value, 'f', -1, 64)
	// Parse into a rational number, better for precision than float64
	rat := new(big.Rat)
	_, ok := rat.SetString(valueStr)
	if !ok {
		return nil, fmt.Errorf("failed to convert %s to rational", valueStr)
	}

	// If decimals > 0, multiply by 10^decimals
	if decimals > 0 {
		scale := new(big.Rat).SetInt(new(big.Int).Exp(
			big.NewInt(10), big.NewInt(int64(decimals)), nil))
		rat.Mul(rat, scale)
	}

	return rat, nil
}

// Validate value is within range for the type
func validateRange(value *big.Int, typeInfo abiTypeInfo) error {
	if typeInfo.signed {
		// For signed types: -2^(bitSize-1) to 2^(bitSize-1)-1
		maxPositive := new(big.Int).Lsh(big.NewInt(1), uint(typeInfo.bitSize-1))
		maxPositive.Sub(maxPositive, big.NewInt(1))

		maxNegative := new(big.Int).Lsh(big.NewInt(1), uint(typeInfo.bitSize-1))
		maxNegative.Neg(maxNegative)

		if value.Cmp(maxPositive) > 0 || value.Cmp(maxNegative) < 0 {
			return fmt.Errorf("value overflow for type %s (value: %s, range: [%s, %s])",
				typeInfo.category, value.String(), maxNegative.String(), maxPositive.String())
		}
	} else {
		// For unsigned types: 0 to 2^bitSize-1
		if value.Sign() < 0 {
			return fmt.Errorf("negative value not allowed for unsigned type %s", typeInfo.category)
		}

		maxValue := new(big.Int).Lsh(big.NewInt(1), uint(typeInfo.bitSize))
		maxValue.Sub(maxValue, big.NewInt(1))

		if value.Cmp(maxValue) > 0 {
			return fmt.Errorf("value overflow for type %s (value: %s, max: %s)",
				typeInfo.category, value.String(), maxValue.String())
		}
	}

	return nil
}

// Convert big.Int to hex string with proper padding
func intToHex(value *big.Int, byteSize int) string {
	// Get the bytes in big-endian order
	bytes := value.Bytes()

	// Pad to required length
	paddedBytes := make([]byte, byteSize)
	copy(paddedBytes[byteSize-len(bytes):], bytes)

	// Convert to hex string
	return hex.EncodeToString(paddedBytes)
}

func MedianInHex(values []float64, responseType string) (string, error) {
	if len(values) == 0 {
		return "", fmt.Errorf("cannot calculate median of empty slice")
	}

	// Sort the float values
	sort.Float64s(values)

	// Calculate median
	middle := len(values) / 2
	var medianValue float64

	if len(values)%2 == 1 {
		// For odd number of values, take the middle value
		medianValue = values[middle]
		fmt.Println("Odd count, middle index:", middle, "value:", medianValue)
	} else {
		// For even number of values use the average of the two middle values
		medianValue = (values[middle-1] + values[middle]) / 2.0
	}

	fmt.Println("Final Median Value:", medianValue)

	// Convert median to 32-byte hex string
	return AbiNumberEncoder(medianValue, responseType)
}
