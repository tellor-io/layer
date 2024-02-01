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
		DocumentHash: "",
		ValueType:    "uint256",
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
	interfacefields := Makedata(datafields, argMarshller)
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
	return IsValueDecodable(value, d.ValueType)
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

func Makedata(fields string, argMrsh []abi.ArgumentMarshaling) []interface{} {
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
			// fields here should be a slice of strings
			// MakeStruct takes a list of strings and components which is a list
			if item.Type == "tuple" {
				interfaceField := MakeNewStruct(list[i].([]interface{}), item.Components)
				interfaceFields = append(interfaceFields, interfaceField.Interface())
			}
			if item.Type == "tuple[]" {
				interfaceField := MakeStructs(list[i], item.Components)
				interfaceFields = append(interfaceFields, interfaceField.Interface())

			}
		}
	}
	return interfaceFields
}

func MakeNewStruct(fields interface{}, component []abi.ArgumentMarshaling) reflect.Value {
	structFields := make([]reflect.StructField, len(component))
	// convert component names to title case
	for i, c := range component {
		component[i].Name = strings.Title(c.Name)
		fieldType, err := ConvertABIToReflectType(c.Type)
		if err != nil {
			panic(fmt.Errorf("Failed to create new ABI type: %v", err))
		}
		structFields[i] = reflect.StructField{
			Name: strings.Title(c.Name),
			Type: fieldType,
		}
	}
	structType := reflect.StructOf(structFields)
	newStruct := reflect.New(structType).Elem()

	for i, fieldValue := range fields.([]interface{}) {
		fieldValue, err := ConvertStringToType(component[i].Type, fieldValue.(string))
		if err != nil {
			panic(fmt.Errorf("Failed to create new ABI type: %v", err))
		}
		fieldVal := newStruct.Field(i)

		if !fieldVal.IsValid() {
			panic(fmt.Errorf("No such field at index: %d", i))
		}

		val := reflect.ValueOf(fieldValue)
		if !val.Type().AssignableTo(fieldVal.Type()) {
			panic(fmt.Errorf("Type mismatch for field at index %d", i))
		}

		fieldVal.Set(val)
	}
	return newStruct
}

func MakeStructs(interfaceFields interface{}, components []abi.ArgumentMarshaling) reflect.Value {
	// Define the struct fields based on the ABI components
	structFields := make([]reflect.StructField, len(components))
	for i, comp := range components {
		fieldType, err := ConvertABIToReflectType(comp.Type)
		if err != nil {
			panic(fmt.Errorf("Failed to convert ABI type to reflect.Type: %v", err))
		}

		structFields[i] = reflect.StructField{
			Name: strings.Title(comp.Name),
			Type: fieldType,
		}
	}

	// Create the struct type
	structType := reflect.StructOf(structFields)
	nestedInterface := interfaceFields.([]interface{})
	// Create a slice to hold the structs
	structSlice := reflect.MakeSlice(reflect.SliceOf(structType), 0, len(nestedInterface))

	// Iterate over the interface fields and populate the struct slice
	for _, instanceFields := range nestedInterface {
		if len(instanceFields.([]interface{})) != len(components) {
			panic(fmt.Errorf("The number of instance fields must match the number of components"))
		}
		newStruct := reflect.New(structType).Elem()
		for i, fieldValue := range instanceFields.([]interface{}) {
			fieldValue, err := ConvertStringToType(components[i].Type, fieldValue.(string))
			if err != nil {
				panic(fmt.Errorf("Failed to create new ABI type: %v", err))
			}
			fieldVal := newStruct.Field(i)
			if !fieldVal.IsValid() {
				panic(fmt.Errorf("No such field at index: %d", i))
			}

			val := reflect.ValueOf(fieldValue)

			if !val.Type().AssignableTo(fieldVal.Type()) {
				panic(fmt.Errorf("Type mismatch for field at index %d", i))
			}

			fieldVal.Set(val)
		}

		structSlice = reflect.Append(structSlice, newStruct)
	}

	return structSlice
}
