package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/types"
)

func TestTwoBitPendingRequestValueCodec(t *testing.T) {
	codec := types.TwoBitPendingRequestValue
	original := types.TwoBitPendingRequest{
		TwoBitReporter: []byte{0x01, 0x02, 0xff},
		ExpiryHeight:   42,
	}

	encoded, err := codec.Encode(original)
	require.NoError(t, err)
	decoded, err := codec.Decode(encoded)
	require.NoError(t, err)
	require.Equal(t, original, decoded)

	_, err = codec.Encode(types.TwoBitPendingRequest{ExpiryHeight: 1})
	require.Error(t, err)
	_, err = codec.Decode([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01})
	require.Error(t, err)
	_, err = codec.Decode(nil)
	require.Error(t, err)

	jsonBz, err := codec.EncodeJSON(original)
	require.NoError(t, err)
	fromJSON, err := codec.DecodeJSON(jsonBz)
	require.NoError(t, err)
	require.Equal(t, original, fromJSON)
	_, err = codec.DecodeJSON([]byte(`{"two_bit_reporter":"zz","expiry_height":1}`))
	require.Error(t, err)
	_, err = codec.DecodeJSON([]byte("{"))
	require.Error(t, err)

	require.Equal(t, "42:01ff", codec.Stringify(types.TwoBitPendingRequest{
		TwoBitReporter: []byte{0x01, 0xff},
		ExpiryHeight:   42,
	}))
	require.Equal(t, "TwoBitPendingRequest", codec.ValueType())
}
