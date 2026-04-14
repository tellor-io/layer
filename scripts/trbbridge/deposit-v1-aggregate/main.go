// Command deposit-v1-aggregate uses the genesis trbbridge DataSpec (x/registry/types/dataspec.go):
//   - Decode aggregate values (ResponseValueType: address, string, uint256, uint256).
//   - Encode deposit query_data and query_id from toLayer + deposit id (AbiComponents on that spec).
//
// Decode:
//
//	go run ./scripts/trbbridge/deposit-v1-aggregate/ -value 0x...
//
// Encode query data + query id (keccak256(query_data)):
//
//	go run ./scripts/trbbridge/deposit-v1-aggregate/ -deposit-id 7 -to-layer=true
//
// You may pass both -value and -deposit-id in one run (encode lines first, then decoded JSON).
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/x/registry/types"
)

func main() {
	valueHex := flag.String("value", "", "hex aggregate value (Aggregate.aggregate_value); optional if -deposit-id is set")
	depositIDStr := flag.String("deposit-id", "", "deposit id (decimal); encodes query_data and query_id via genesis trbbridge spec")
	toLayer := flag.Bool("to-layer", true, "toLayer field for query data (deposits use true)")
	queryTypeOuter := flag.String("query-type", "TRBBridge", "outer ABI string for EncodeWithQuerytype (EVM v1 uses TRBBridge)")
	flag.Parse()

	if strings.TrimSpace(*valueHex) == "" && strings.TrimSpace(*depositIDStr) == "" {
		fmt.Fprintln(os.Stderr, "provide -value and/or -deposit-id")
		os.Exit(2)
	}

	spec, ok := trbBridgeSpec()
	if !ok {
		fmt.Fprintln(os.Stderr, "trbbridge spec not found in genesis dataspecs")
		os.Exit(1)
	}

	if strings.TrimSpace(*depositIDStr) != "" {
		depositID, err := strconv.ParseUint(strings.TrimSpace(*depositIDStr), 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "deposit-id:", err)
			os.Exit(1)
		}
		datafields := fmt.Sprintf("[%q,%q]", strconv.FormatBool(*toLayer), strconv.FormatUint(depositID, 10))
		queryData, err := spec.EncodeData(*queryTypeOuter, datafields)
		if err != nil {
			fmt.Fprintln(os.Stderr, "encode query data:", err)
			os.Exit(1)
		}
		queryID := crypto.Keccak256(queryData)
		fmt.Printf("query_data_hex=0x%s\n", hex.EncodeToString(queryData))
		fmt.Printf("query_id_hex=0x%s\n", hex.EncodeToString(queryID))
	}

	if strings.TrimSpace(*valueHex) != "" {
		decoded, err := decodeByResponseValueType(*valueHex, spec.ResponseValueType)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		out, err := types.ConvertToJSON(decoded)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(out)
	}
}

func trbBridgeSpec() (types.DataSpec, bool) {
	for _, s := range types.GenesisDataSpec() {
		if strings.EqualFold(s.QueryType, "trbbridge") {
			return s, true
		}
	}
	return types.DataSpec{}, false
}

// decodeByResponseValueType matches Solidity abi.encode(arg0, arg1, …): comma-separated
// ResponseValueType values are unpacked as a flat argument list (not a single parenthesized tuple).
func decodeByResponseValueType(valueHex string, responseValueType string) ([]interface{}, error) {
	raw, err := hex.DecodeString(types.Remove0xPrefix(valueHex))
	if err != nil {
		return nil, fmt.Errorf("value hex: %w", err)
	}

	rv := strings.TrimSpace(responseValueType)
	if strings.Contains(rv, "(") && strings.Contains(rv, ")") {
		return types.DecodeValue(valueHex, rv)
	}

	parts := strings.Split(rv, ",")
	var typeList []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			typeList = append(typeList, p)
		}
	}
	if len(typeList) == 0 {
		return nil, fmt.Errorf("empty response_value_type %q", responseValueType)
	}
	if len(typeList) == 1 {
		return types.DecodeValue(valueHex, typeList[0])
	}

	var args abi.Arguments
	for _, p := range typeList {
		t, err := abi.NewType(p, p, nil)
		if err != nil {
			return nil, fmt.Errorf("abi type %q: %w", p, err)
		}
		args = append(args, abi.Argument{Type: t})
	}
	return args.Unpack(raw)
}
