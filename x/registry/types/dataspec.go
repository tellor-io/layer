package types

// genesis spot price data spec
func GenesisDataSpec() DataSpec {
	return DataSpec{
		DocumentHash:      "",
		ValueType:         "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         "genesis",
	}
}
