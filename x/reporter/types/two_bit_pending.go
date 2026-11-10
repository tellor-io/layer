package types

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	collcodec "cosmossdk.io/collections/codec"
)

// TwoBitPendingRequestExpiryBlocks is how long a 2bit registration request
// stays acceptable. A request made at height H can be accepted through height
// H+TwoBitPendingRequestExpiryBlocks and is rejected after that.
const TwoBitPendingRequestExpiryBlocks uint64 = 2000

// TwoBitPendingRequest is the one outstanding invite from a reporter.
// Replacing or canceling it removes this value, so the previous address can no longer accept.
type TwoBitPendingRequest struct {
	TwoBitReporter []byte
	ExpiryHeight   uint64
}

type twoBitPendingRequestValueCodec struct{}

func (twoBitPendingRequestValueCodec) Encode(value TwoBitPendingRequest) ([]byte, error) {
	if len(value.TwoBitReporter) == 0 {
		return nil, fmt.Errorf("pending 2bit reporter is empty")
	}
	out := make([]byte, 8+len(value.TwoBitReporter))
	binary.BigEndian.PutUint64(out[:8], value.ExpiryHeight)
	copy(out[8:], value.TwoBitReporter)
	return out, nil
}

func (twoBitPendingRequestValueCodec) Decode(b []byte) (TwoBitPendingRequest, error) {
	if len(b) <= 8 {
		return TwoBitPendingRequest{}, fmt.Errorf("2bit pending request is too short")
	}
	twoBitReporter := make([]byte, len(b)-8)
	copy(twoBitReporter, b[8:])
	return TwoBitPendingRequest{
		ExpiryHeight:   binary.BigEndian.Uint64(b[:8]),
		TwoBitReporter: twoBitReporter,
	}, nil
}

func (c twoBitPendingRequestValueCodec) EncodeJSON(value TwoBitPendingRequest) ([]byte, error) {
	return json.Marshal(twoBitPendingRequestJSON{
		TwoBitReporter: hex.EncodeToString(value.TwoBitReporter),
		ExpiryHeight:   value.ExpiryHeight,
	})
}

func (c twoBitPendingRequestValueCodec) DecodeJSON(b []byte) (TwoBitPendingRequest, error) {
	var raw twoBitPendingRequestJSON
	if err := json.Unmarshal(b, &raw); err != nil {
		return TwoBitPendingRequest{}, err
	}
	twoBitReporter, err := hex.DecodeString(raw.TwoBitReporter)
	if err != nil {
		return TwoBitPendingRequest{}, err
	}
	return TwoBitPendingRequest{TwoBitReporter: twoBitReporter, ExpiryHeight: raw.ExpiryHeight}, nil
}

func (twoBitPendingRequestValueCodec) Stringify(value TwoBitPendingRequest) string {
	return fmt.Sprintf("%d:%s", value.ExpiryHeight, hex.EncodeToString(value.TwoBitReporter))
}

func (twoBitPendingRequestValueCodec) ValueType() string {
	return "TwoBitPendingRequest"
}

type twoBitPendingRequestJSON struct {
	TwoBitReporter string `json:"two_bit_reporter"`
	ExpiryHeight   uint64 `json:"expiry_height"`
}

// TwoBitPendingRequestValue encodes a pending 2bit registration request.
var TwoBitPendingRequestValue collcodec.ValueCodec[TwoBitPendingRequest] = twoBitPendingRequestValueCodec{}
