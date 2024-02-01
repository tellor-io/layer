package types

var SupportedValueTypes = map[string]bool{
	"int8":      true,
	"int16":     true,
	"int32":     true,
	"int64":     true,
	"int128":    true,
	"int256":    true,
	"int[]":     true,
	"int8[]":    true,
	"int16[]":   true,
	"int32[]":   true,
	"int64[]":   true,
	"int128[]":  true,
	"int256[]":  true,
	"uint8":     true,
	"uint16":    true,
	"uint32":    true,
	"uint64":    true,
	"uint128":   true,
	"uint256":   true,
	"uint[]":    true,
	"uint8[]":   true,
	"uint16[]":  true,
	"uint32[]":  true,
	"uint64[]":  true,
	"uint128[]": true,
	"uint256[]": true,
	"bytes":     true,
	"string":    true,
	"bool":      true,
	"address":   true,
	"bytes[]":   true,
	"string[]":  true,
	"bool[]":    true,
	"address[]": true,
}

func SupportedType(dataType string) bool {
	_, exists := SupportedValueTypes[dataType]
	return exists
}
