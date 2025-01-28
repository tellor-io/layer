package types

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	Tuplenoarray = "tuple"
	Tuplearray   = "tuple[]"
)

// genesis spot price data spec
func GenesisDataSpec() []DataSpec {
	var dataspecs []DataSpec
	// spotprice data spec
	dataspecs = append(dataspecs, DataSpec{
		DocumentHash:      "",
		ResponseValueType: "uint256",
		AbiComponents: []*ABIComponent{
			{
				Name:            "asset",
				FieldType:       "string",
				NestedComponent: []*ABIComponent{},
			},
			{
				Name:            "currency",
				FieldType:       "string",
				NestedComponent: []*ABIComponent{},
			},
		},
		AggregationMethod: "weighted-median",
		Registrar:         "genesis",
		ReportBlockWindow: 2,
		QueryType:         "spotprice",
	})
	// trbbridge data spec
	dataspecs = append(dataspecs, DataSpec{
		DocumentHash:      "",
		ResponseValueType: "address, string, uint256",
		AbiComponents: []*ABIComponent{
			{
				Name:            "toLayer",
				FieldType:       "bool",
				NestedComponent: []*ABIComponent{},
			},
			{
				Name:            "depositId",
				FieldType:       "uint256",
				NestedComponent: []*ABIComponent{},
			},
		},
		AggregationMethod: "weighted-mode",
		Registrar:         "genesis",
		ReportBlockWindow: 2000,
		QueryType:         "trbbridge",
	})

	return dataspecs
}

func (d DataSpec) EncodeData(querytype, datafields string) ([]byte, error) {
	argMarshller := d.MakeArgMarshaller()
	args := MakeArguments(argMarshller)
	interfacefields := MakePackdata(datafields, argMarshller)
	encodedBytes, err := args.Pack(interfacefields...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack arguments: %w", err)
	}

	querydataBytes, err := EncodeWithQuerytype(querytype, encodedBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments: %w", err)
	}

	return querydataBytes, nil
}

func (d DataSpec) ValidateValue(value string) error {
	return IsValueDecodable(value, d.ResponseValueType)
}

func (d DataSpec) DecodeValue(value string) (string, error) {
	valueInterface, err := DecodeValue(value, d.ResponseValueType)
	if err != nil {
		return "", fmt.Errorf("failed to decode value: %w", err)
	}
	valueJson, err := ConvertToJSON(valueInterface)
	if err != nil {
		return "", fmt.Errorf("failed to convert to JSON: %w", err)
	}
	return valueJson, nil
}

func (d DataSpec) MakeArgMarshaller() []abi.ArgumentMarshaling {
	argMrsh := []abi.ArgumentMarshaling{}
	for _, cmp := range d.AbiComponents {
		_argMrsh := abi.ArgumentMarshaling{
			Name: cmp.Name,
			Type: cmp.FieldType,
		}
		if cmp.FieldType == Tuplenoarray || cmp.FieldType == Tuplearray {
			for _, cmp2 := range cmp.NestedComponent {
				_argMrsh_ := abi.ArgumentMarshaling{
					Name: cmp2.Name,
					Type: cmp2.FieldType,
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
		panic(fmt.Errorf("error unmarshalling JSON: %w", err))
	}
	list, ok := data.([]interface{})
	if !ok {
		panic(fmt.Errorf("error asserting data to a slice of interfaces"))
	}
	// check length should be the same
	if len(list) != len(argMrsh) {
		panic(fmt.Errorf("length of fields and argMrsh should be the same"))
	}
	var interfaceFields []interface{}
	for i, item := range argMrsh {
		if len(item.Components) == 0 {
			interfaceField, err := ConvertStringToType(item.Type, list[i].(string))
			if err != nil {
				panic(fmt.Errorf("failed to create new ABI type: %w", err))
			}
			interfaceFields = append(interfaceFields, interfaceField)
		} else {
			// crete structs and populate
			if item.Type == Tuplenoarray {
				interfaceField := createAndPopulateStruct(list[i].([]interface{}), item.Components)
				interfaceFields = append(interfaceFields, interfaceField.Interface())
			}
			if item.Type == Tuplearray {
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
		caser := cases.Title(language.English, cases.NoLower)
		compName := caser.String(c.Name)
		fieldType, err := ConvertTypeToReflectType(c.Type)
		if err != nil {
			panic(fmt.Errorf("failed to create new ABI type: %w", err))
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
			panic(fmt.Errorf("failed to convert string to type: %w", err))
		}
		fieldVal := newStruct.Field(i)
		if !fieldVal.IsValid() {
			panic(fmt.Errorf("no such field: %s", components[i].Name))
		}
		val := reflect.ValueOf(fieldValue)
		if !val.Type().AssignableTo(fieldVal.Type()) {
			panic(fmt.Errorf("type mismatch for field: %s", components[i].Name))
		}
		fieldVal.Set(val)
	}
	return newStruct
}
