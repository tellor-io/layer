package types

import "github.com/tellor-io/layer/lib"

// genesis spot price data spec
func GenesisDataSpec() DataSpec {
	return DataSpec{
		DocumentHash:        "",
		ValueType:           "uint256",
		QueryParameterTypes: []string{"string", "string"},
		AggregationMethod:   "weighted-median",
		Registrar:           "genesis",
	}
}

func (d DataSpec) GenerateQuerydata(querytype string, parameters []string) (string, error) {
	return lib.GenerateQuerydata(querytype, parameters, d.QueryParameterTypes)
}
