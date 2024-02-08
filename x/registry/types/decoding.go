package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func IsValueDecodable(value, datatype string) error {
	result, err := DecodeValue(value, datatype)
	if err != nil {
		return fmt.Errorf("failed to unpack value: %v", err)
	}
	fmt.Println("Decoded value: ", result[0])
	return nil
}

func DecodeValue(value, datatype string) ([]interface{}, error) {
	value = Remove0xPrefix(value)
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode value string: %v", err)
	}

	comp := []abi.ArgumentMarshaling{}
	if strings.Contains(datatype, "(") && strings.Contains(datatype, ")") {
		untrimmed := datatype
		if strings.HasSuffix(datatype, "[]") {
			datatype = "tuple[]"
			untrimmed = untrimmed[:len(untrimmed)-2]
		} else {
			datatype = "tuple"
		}
		types := strings.Split(strings.Trim(untrimmed, "()"), ",")

		_comp := abi.ArgumentMarshaling{Type: datatype, Name: "Main"}

		for i, element := range types {
			_comp.Components = append(_comp.Components, abi.ArgumentMarshaling{
				Type: element, Name: "Value" + fmt.Sprintf("%d", i),
			})
		}
		comp = append(comp, _comp)
		args := MakeArguments(comp)

		return args.Unpack(valueBytes)
	}

	argType, err := abi.NewType(datatype, datatype, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ABI type when decoding value: %v", err)
	}
	arg := abi.Argument{
		Type: argType,
	}

	args := abi.Arguments{arg}

	return args.Unpack(valueBytes)
}

// Decodes query data bytes to query type and data fields
func DecodeQueryType(data []byte) (string, []byte, error) {
	// Create an ABI arguments object based on the types
	strArg, err := abi.NewType("string", "string", nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create string ABI type when decoding query type: %v", err)
	}
	bytesArg, err := abi.NewType("bytes", "bytes", nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create bytes ABI type when decoding query type: %v", err)
	}
	args := abi.Arguments{
		abi.Argument{Type: strArg},
		abi.Argument{Type: bytesArg},
	}
	result, err := args.Unpack(data)
	if err != nil {
		return "", nil, fmt.Errorf("failed to unpack query type: %v", err)
	}
	return result[0].(string), result[1].([]byte), nil
}

func DecodeParamtypes(data []byte, component []*ABIComponent) (string, error) {
	var args abi.Arguments
	for _, comp := range component {
		marshaling := convertToArgumentMarshaling(*comp)
		argType, err := abi.NewType(marshaling.Type, marshaling.Type, marshaling.Components)
		if err != nil {
			return "", err
		}

		args = append(args, abi.Argument{
			Name: marshaling.Name,
			Type: argType,
		})
	}

	result, err := args.Unpack(data)
	if err != nil {
		return "", fmt.Errorf("failed to unpack query data into its fields: %v", err)
	}

	return convertToJSON(result)

}

func convertToArgumentMarshaling(comp ABIComponent) abi.ArgumentMarshaling {
	var nestedArgMarshallings []abi.ArgumentMarshaling
	for _, nestedComp := range comp.NestedComponent {
		nestedArgMarshaling := convertToArgumentMarshaling(*nestedComp)
		nestedArgMarshallings = append(nestedArgMarshallings, nestedArgMarshaling)
	}

	return abi.ArgumentMarshaling{
		Name:       comp.Name,
		Type:       comp.Type,
		Components: nestedArgMarshallings,
	}
}
