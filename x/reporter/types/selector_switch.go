package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/collections/codec"
)

// SelectorReporterSwitch is one reporter change for a selector.
// Switches are stored newest first. CompletedHeight is 0 until the pending
// switch finalizes and Selection.Reporter actually moves. CancelledHeight is 0
// when the switch was not cancelled. A cancelled row keeps CompletedHeight at 0
// and stays until prune. An open row has both heights at 0.
type SelectorReporterSwitch struct {
	OriginalReporter []byte `json:"original_reporter"`
	BeganHeight      uint64 `json:"began_height"`
	CompletedHeight  uint64 `json:"completed_height"`
	CancelledHeight  uint64 `json:"cancelled_height"`
}

// SelectorReporterSwitches is the full history for one selector, newest first.
type SelectorReporterSwitches struct {
	Switches []SelectorReporterSwitch `json:"switches"`
}

// SelectorReporterSwitchesValue encodes a selector's switch history.
var SelectorReporterSwitchesValue codec.ValueCodec[SelectorReporterSwitches] = selectorSwitchesCodec{}

type selectorSwitchesCodec struct{}

func (selectorSwitchesCodec) Encode(value SelectorReporterSwitches) ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(value.Switches))); err != nil {
		return nil, err
	}
	for _, sw := range value.Switches {
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(sw.OriginalReporter))); err != nil {
			return nil, err
		}
		if _, err := buf.Write(sw.OriginalReporter); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, sw.BeganHeight); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, sw.CompletedHeight); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, sw.CancelledHeight); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (selectorSwitchesCodec) Decode(b []byte) (SelectorReporterSwitches, error) {
	var out SelectorReporterSwitches
	if len(b) == 0 {
		return out, nil
	}
	reader := bytes.NewReader(b)
	var n uint32
	if err := binary.Read(reader, binary.BigEndian, &n); err != nil {
		return out, err
	}
	out.Switches = make([]SelectorReporterSwitch, 0, n)
	for i := uint32(0); i < n; i++ {
		var addrLen uint32
		if err := binary.Read(reader, binary.BigEndian, &addrLen); err != nil {
			return SelectorReporterSwitches{}, err
		}
		addr := make([]byte, addrLen)
		if _, err := reader.Read(addr); err != nil {
			return SelectorReporterSwitches{}, err
		}
		var began, completed, cancelled uint64
		if err := binary.Read(reader, binary.BigEndian, &began); err != nil {
			return SelectorReporterSwitches{}, err
		}
		if err := binary.Read(reader, binary.BigEndian, &completed); err != nil {
			return SelectorReporterSwitches{}, err
		}
		if err := binary.Read(reader, binary.BigEndian, &cancelled); err != nil {
			return SelectorReporterSwitches{}, err
		}
		out.Switches = append(out.Switches, SelectorReporterSwitch{
			OriginalReporter: addr,
			BeganHeight:      began,
			CompletedHeight:  completed,
			CancelledHeight:  cancelled,
		})
	}
	return out, nil
}

func (selectorSwitchesCodec) EncodeJSON(value SelectorReporterSwitches) ([]byte, error) {
	return json.Marshal(value)
}

func (c selectorSwitchesCodec) DecodeJSON(b []byte) (SelectorReporterSwitches, error) {
	var out SelectorReporterSwitches
	err := json.Unmarshal(b, &out)
	return out, err
}

func (selectorSwitchesCodec) Stringify(value SelectorReporterSwitches) string {
	return fmt.Sprintf("%+v", value)
}

func (selectorSwitchesCodec) ValueType() string {
	return "selector_reporter_switches"
}
