package types

const (
	WeightedMedian = "weighted-median"
	WeightedMode   = "weighted-mode"
)

var (
	SupportedAggregationMethod = map[string]bool{
		WeightedMedian: true,
		WeightedMode:   true,
	}
)
