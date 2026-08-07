package enrich

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

const (
	// BridgeChannel is the Discord channel for bridge-related weak aggregates.
	BridgeChannel = "bridge"

	// DefaultTipAmount is the tip amount suggested in weak bridge-deposit alerts.
	DefaultTipAmount = "1500loya"

	trbBridgeQueryType   = "TRBBridge"
	trbBridgeV2QueryType = "TRBBridgeV2"
)

// BridgeDeposit is a decoded TRBBridge / TRBBridgeV2 query.
type BridgeDeposit struct {
	QueryType string
	ToLayer   bool
	DepositID uint64
}

// IsBridgeDeposit reports whether d is a deposit (toLayer=true) for a known bridge query type.
func (d BridgeDeposit) IsBridgeDeposit() bool {
	return d.ToLayer && IsBridgeQueryType(d.QueryType)
}

// IsBridgeQueryType reports whether queryType is TRBBridge or TRBBridgeV2.
func IsBridgeQueryType(queryType string) bool {
	return strings.EqualFold(queryType, trbBridgeQueryType) ||
		strings.EqualFold(queryType, trbBridgeV2QueryType)
}

// QueryTypeFromQueryData extracts the ABI-encoded query type string from
// query_data hex (abi.encode(string, bytes)). Empty when undecodable.
func QueryTypeFromQueryData(queryDataHex string) string {
	raw, err := decodeHex(queryDataHex)
	if err != nil || len(raw) == 0 {
		return ""
	}
	queryType, _, err := decodeQueryType(raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(queryType)
}

// DecodeBridgeDepositQueryData unpacks abi.encode(string queryType, bytes args)
// where args is abi.encode(bool toLayer, uint256 depositId).
// Returns ok=false when query_data is empty, not bridge-shaped, or not a deposit.
func DecodeBridgeDepositQueryData(queryDataHex string) (BridgeDeposit, bool) {
	raw, err := decodeHex(queryDataHex)
	if err != nil || len(raw) == 0 {
		return BridgeDeposit{}, false
	}
	queryType, args, err := decodeQueryType(raw)
	if err != nil || !IsBridgeQueryType(queryType) {
		return BridgeDeposit{}, false
	}
	toLayer, depositID, err := decodeBridgeArgs(args)
	if err != nil || !toLayer {
		return BridgeDeposit{}, false
	}
	return BridgeDeposit{
		QueryType: queryType,
		ToLayer:   true,
		DepositID: depositID,
	}, true
}

// TipCommand builds a copy-pasteable layerd tip CLI for the given query data.
// queryDataHex should be the on-chain event value (hex, optional 0x prefix).
func TipCommand(queryDataHex, chainID string) string {
	qd := strings.TrimSpace(queryDataHex)
	qd = strings.TrimPrefix(qd, "0x")
	qd = strings.TrimPrefix(qd, "0X")
	chainID = strings.TrimSpace(chainID)
	if qd == "" || chainID == "" {
		return ""
	}
	return fmt.Sprintf("./layerd tx oracle tip %s %s --chain-id %s", qd, DefaultTipAmount, chainID)
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if s == "" {
		return nil, fmt.Errorf("empty hex")
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

// decodeQueryType unpacks abi.encode(string, bytes).
func decodeQueryType(data []byte) (string, []byte, error) {
	if len(data) < 64 {
		return "", nil, fmt.Errorf("query data too short")
	}
	strOff, err := readUint64Word(data, 0)
	if err != nil {
		return "", nil, err
	}
	bytesOff, err := readUint64Word(data, 32)
	if err != nil {
		return "", nil, err
	}
	str, err := readDynamicBytes(data, strOff)
	if err != nil {
		return "", nil, fmt.Errorf("query type string: %w", err)
	}
	args, err := readDynamicBytes(data, bytesOff)
	if err != nil {
		return "", nil, fmt.Errorf("query args: %w", err)
	}
	return string(str), args, nil
}

// decodeBridgeArgs unpacks abi.encode(bool toLayer, uint256 depositId).
func decodeBridgeArgs(args []byte) (toLayer bool, depositID uint64, err error) {
	if len(args) < 64 {
		return false, 0, fmt.Errorf("bridge args too short")
	}
	toLayerWord, err := readBigIntWord(args, 0)
	if err != nil {
		return false, 0, err
	}
	if toLayerWord.Cmp(big.NewInt(0)) != 0 && toLayerWord.Cmp(big.NewInt(1)) != 0 {
		return false, 0, fmt.Errorf("invalid bool word")
	}
	toLayer = toLayerWord.Cmp(big.NewInt(1)) == 0
	idWord, err := readBigIntWord(args, 32)
	if err != nil {
		return false, 0, err
	}
	if !idWord.IsUint64() {
		return false, 0, fmt.Errorf("deposit id overflows uint64")
	}
	return toLayer, idWord.Uint64(), nil
}

func readUint64Word(data []byte, offset int) (uint64, error) {
	n, err := readBigIntWord(data, offset)
	if err != nil {
		return 0, err
	}
	if !n.IsUint64() {
		return 0, fmt.Errorf("word at %d overflows uint64", offset)
	}
	return n.Uint64(), nil
}

func readBigIntWord(data []byte, offset int) (*big.Int, error) {
	if offset < 0 || offset+32 > len(data) {
		return nil, fmt.Errorf("word out of range at %d", offset)
	}
	return new(big.Int).SetBytes(data[offset : offset+32]), nil
}

func readDynamicBytes(data []byte, offset uint64) ([]byte, error) {
	if offset > uint64(len(data)) {
		return nil, fmt.Errorf("dynamic offset %d past end", offset)
	}
	off := int(offset)
	if off+32 > len(data) {
		return nil, fmt.Errorf("dynamic length word past end")
	}
	length := binary.BigEndian.Uint64(data[off+24 : off+32])
	start := off + 32
	end := start + int(length)
	if length > uint64(len(data)) || end > len(data) {
		return nil, fmt.Errorf("dynamic payload past end")
	}
	return data[start:end], nil
}
