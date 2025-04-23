package lib

import (
	"errors"
	"sort"

	"golang.org/x/exp/constraints"
)

// Median is a generic median calculator.
// If the input has an even number of elements, then the average of the two middle numbers is rounded away from zero.
func Median[V uint64 | uint32 | int64 | int32](input []V) (V, error) {
	l := len(input)
	if l == 0 {
		return 0, errors.New("input cannot be empty")
	}

	inputCopy := make([]V, l)
	copy(inputCopy, input)
	sort.Slice(inputCopy, func(i, j int) bool { return inputCopy[i] < inputCopy[j] })

	midIdx := l / 2

	if l%2 == 1 {
		return inputCopy[midIdx], nil
	}

	// The median is an average of the two middle numbers. It's rounded away from zero
	// to the nearest integer.
	// Note x <= y since `inputCopy` is sorted.
	x := inputCopy[midIdx-1]
	y := inputCopy[midIdx]

	if x <= 0 && y >= 0 {
		// x and y have different signs, so x+y cannot overflow.
		sum := x + y
		return sum/2 + sum%2, nil
	}

	if y > 0 {
		// x and y are both positive.
		return y - (y-x)/2, nil
	}

	// x and y are both negative.
	return x + (y-x)/2, nil
}

func AbsInt32(i int32) uint32 {
	if i < 0 {
		return uint32(0 - i)
	}

	return uint32(i)
}

func Min[T constraints.Ordered](x, y T) T {
	if x > y {
		return y
	}
	return x
}
