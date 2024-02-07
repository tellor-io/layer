package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// genesis spot price data spec
func GenesisDataSpec() DataSpec {
	return DataSpec{
		DocumentHash:      "",
		ResponseValueType: "uint256",
		AbiComponents: []*ABIComponent{
			{Name: "asset", Type: "string"},
			{Name: "currency", Type: "string"},
		},
		AggregationMethod: "weighted-median",
		Registrar:         "genesis",
	}
}

func (d DataSpec) EncodeData(querytype, datafields string) (string, error) {
	argMarshller := d.MakeArgMarshaller()
	args := MakeArguments(argMarshller)
	interfacefields := MakePackdata(datafields, argMarshller)
	encodedBytes, err := args.Pack(interfacefields...)
	if err != nil {
		return "", fmt.Errorf("Failed to pack arguments: %v", err)
	}

	querydataBytes, err := EncodeWithQuerytype(querytype, encodedBytes)
	if err != nil {
		return "", fmt.Errorf("Failed to encode arguments: %v", err)
	}

	return hex.EncodeToString(querydataBytes), nil
}

func (d DataSpec) ValidateValue(value string) error {
	return IsValueDecodable(value, d.ResponseValueType)
}

func (d DataSpec) DecodeValue(value string) (string, error) {
	valueInterface, err := DecodeValue(value, d.ResponseValueType)
	if err != nil {
		return "", fmt.Errorf("Failed to decode value: %v", err)
	}
	valueBytes, err := convertToJSON(valueInterface)
	if err != nil {
		return "", fmt.Errorf("Failed to convert to JSON: %v", err)
	}
	return string(valueBytes), nil
}

func (d DataSpec) MakeArgMarshaller() []abi.ArgumentMarshaling {
	argMrsh := []abi.ArgumentMarshaling{}
	for _, cmp := range d.AbiComponents {
		_argMrsh := abi.ArgumentMarshaling{
			Name: cmp.Name,
			Type: cmp.Type,
		}
		if cmp.Type == "tuple" || cmp.Type == "tuple[]" {
			for _, cmp2 := range cmp.NestedComponent {
				_argMrsh_ := abi.ArgumentMarshaling{
					Name: cmp2.Name,
					Type: cmp2.Type,
				}
				_argMrsh.Components = append(_argMrsh.Components, _argMrsh_)
			}
		}
		argMrsh = append(argMrsh, _argMrsh)
	}
	return argMrsh
}

func MakeArguments(argMrsh []abi.ArgumentMarshaling) abi.Arguments {
	var arguments abi.Arguments
	for _, arg := range argMrsh {
		// make struct for each
		argType, err := abi.NewType(arg.Type, arg.Type, arg.Components)
		if err != nil {
			panic(err)
		}
		arguments = append(arguments, abi.Argument{
			Name: arg.Name,
			Type: argType,
		})
	}
	return arguments
}

func MakePackdata(fields string, argMrsh []abi.ArgumentMarshaling) []interface{} {
	var data interface{}
	err := json.Unmarshal([]byte(fields), &data)
	if err != nil {
		panic(fmt.Errorf("Error unmarshalling JSON: %v", err))
	}
	list, ok := data.([]interface{})
	if !ok {
		panic(fmt.Errorf("Error asserting data to a slice of interfaces"))
	}
	// check length should be the same
	if len(list) != len(argMrsh) {
		panic(fmt.Errorf("Length of fields and argMrsh should be the same"))
	}
	var interfaceFields []interface{}
	for i, item := range argMrsh {
		if len(item.Components) == 0 {
			interfaceField, err := ConvertStringToType(item.Type, list[i].(string))
			if err != nil {
				panic(fmt.Errorf("Failed to create new ABI type: %v", err))
			}
			interfaceFields = append(interfaceFields, interfaceField)
		} else {
			// crete structs and populate
			if item.Type == "tuple" {
				interfaceField := createAndPopulateStruct(list[i].([]interface{}), item.Components)
				interfaceFields = append(interfaceFields, interfaceField.Interface())
			}
			if item.Type == "tuple[]" {
				interfaceField := createAndPopulateStructSlice(list[i], item.Components)
				interfaceFields = append(interfaceFields, interfaceField.Interface())

			}
		}
	}
	return interfaceFields
}

// createAndPopulateStructSlice creates a slice of structs from interface fields and ABI components.
func createAndPopulateStructSlice(interfaceFields interface{}, components []abi.ArgumentMarshaling) reflect.Value {
	nestedInterface := interfaceFields.([]interface{})
	structType := reflect.TypeOf(createAndPopulateStruct([]interface{}{}, components).Interface())
	structSlice := reflect.MakeSlice(reflect.SliceOf(structType), 0, len(nestedInterface))

	for _, instanceFields := range nestedInterface {
		newStruct := createAndPopulateStruct(instanceFields.([]interface{}), components)
		structSlice = reflect.Append(structSlice, newStruct)
	}

	return structSlice
}

// Convert component names to title case and create a single struct populated with field values.
func createAndPopulateStruct(fields []interface{}, components []abi.ArgumentMarshaling) reflect.Value {
	structFields := make([]reflect.StructField, len(components))
	for i, c := range components {
		// Ensure component names are in title case
		compName := strings.Title(c.Name)
		fieldType, err := ConvertTypeToReflectType(c.Type)
		if err != nil {
			panic(fmt.Errorf("Failed to create new ABI type: %v", err))
		}
		structFields[i] = reflect.StructField{
			Name: compName,
			Type: fieldType,
		}
	}
	structType := reflect.StructOf(structFields)
	newStruct := reflect.New(structType).Elem()

	for i, fieldValue := range fields {
		fieldValue, err := ConvertStringToType(components[i].Type, fieldValue.(string))
		if err != nil {
			panic(fmt.Errorf("Failed to convert string to type: %v", err))
		}
		fieldVal := newStruct.Field(i)
		if !fieldVal.IsValid() {
			panic(fmt.Errorf("No such field: %s", components[i].Name))
		}
		val := reflect.ValueOf(fieldValue)
		if !val.Type().AssignableTo(fieldVal.Type()) {
			panic(fmt.Errorf("Type mismatch for field: %s", components[i].Name))
		}
		fieldVal.Set(val)
	}
	return newStruct
}
