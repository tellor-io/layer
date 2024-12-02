package types

import (
	collcodec "cosmossdk.io/collections/codec"
	"cosmossdk.io/math"
)

type legacyDecValueCodec struct{}

func (i legacyDecValueCodec) Encode(value math.LegacyDec) ([]byte, error) {
	return value.Marshal()
}

func (i legacyDecValueCodec) Decode(b []byte) (math.LegacyDec, error) {
	v := new(math.LegacyDec)
	err := v.Unmarshal(b)
	if err != nil {
		return math.LegacyDec{}, err
	}
	return *v, nil
}

func (i legacyDecValueCodec) EncodeJSON(value math.LegacyDec) ([]byte, error) {
	return value.MarshalJSON()
}

func (i legacyDecValueCodec) DecodeJSON(b []byte) (math.LegacyDec, error) {
	v := new(math.LegacyDec)
	err := v.UnmarshalJSON(b)
	if err != nil {
		return math.LegacyDec{}, err
	}
	return *v, nil
}

func (i legacyDecValueCodec) Stringify(value math.LegacyDec) string {
	return value.String()
}

func (i legacyDecValueCodec) ValueType() string {
	return "math.LegacyDec"
}

var LegacyDecValue collcodec.ValueCodec[math.LegacyDec] = legacyDecValueCodec{}
