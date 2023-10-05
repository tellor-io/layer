package bridge

import (
	"encoding/binary"
	"time"

	gogotypes "github.com/cosmos/gogoproto/types"

	"reflect"

	"github.com/cometbft/cometbft/libs/bytes"
)

// cdcEncode returns nil if the input is nil, otherwise returns
// proto.Marshal(<type>Value{Value: item})
func cdcEncode(item interface{}) []byte {
	if item != nil && !isTypedNil(item) && !isEmpty(item) {
		switch item := item.(type) {
		case string:
			i := gogotypes.StringValue{
				Value: item,
			}
			bz, err := i.Marshal()
			if err != nil {
				return nil
			}
			return bz
		case int64:
			i := gogotypes.Int64Value{
				Value: item,
			}
			bz, err := i.Marshal()
			if err != nil {
				return nil
			}
			return bz
		case bytes.HexBytes:
			i := gogotypes.BytesValue{
				Value: item,
			}
			bz, err := i.Marshal()
			if err != nil {
				return nil
			}
			return bz
		default:
			return nil
		}
	}

	return nil
}

// Go lacks a simple and safe way to see if something is a typed nil.
// See:
//   - https://dave.cheney.net/2017/08/09/typed-nils-in-go-2
//   - https://groups.google.com/forum/#!topic/golang-nuts/wnH302gBa4I/discussion
//   - https://github.com/golang/go/issues/21538
func isTypedNil(o interface{}) bool {
	rv := reflect.ValueOf(o)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// Returns true if it has zero length.
func isEmpty(o interface{}) bool {
	rv := reflect.ValueOf(o)
	switch rv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return rv.Len() == 0
	default:
		return false
	}
}

func encodeTime(t time.Time) []byte {
	bz := []byte{}
	s := t.Unix()
	if s != 0 {
		bz = append(bz, encodeFieldNumberAndTyp3(1, 0)...)
		bz = append(bz, encodeUvarint(uint64(s))...)
	}
	ns := int32(t.Nanosecond())
	if ns != 0 {
		bz = append(bz, encodeFieldNumberAndTyp3(2, 0)...)
		bz = append(bz, encodeUvarint(uint64(ns))...)
	}
	return bz
}

func encodeFieldNumberAndTyp3(num uint32, typ uint8) []byte {
	return encodeUvarint((uint64(num) << 3) | uint64(typ))
}

func encodeUvarint(value uint64) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, value)
	return buf[:n]
}
