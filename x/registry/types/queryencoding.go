package types

// import (
// 	"encoding/hex"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"math/big"
// 	"reflect"
// 	"strconv"
// 	"strings"

// 	"github.com/ethereum/go-ethereum/accounts/abi"
// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/ethereum/go-ethereum/common/hexutil"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/status"
// )

// func Take() {
// 	// Define the ABI for your contract function
// 	abiJSON := `[{"constant":true,"inputs":[{"name":"","type":"bytes32[]"}],"name":"encode","outputs":[{"name":"","type":"bytes"}],"payable":false,"stateMutability":"view","type":"function"}]`
// 	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
// 	if err != nil {
// 		fmt.Println("Error parsing ABI:", err)
// 		return
// 	}

// 	// Convert the input data into [32]byte
// 	input, _ := hexutil.Decode("0x0000000000000000000000000000000000000000000000000000000000000020")
// 	fmt.Println("input: ", input)
// 	var inputData [32]byte
// 	copy(inputData[:], input)

// 	// Encode the input data
// 	encodedData, err := parsedABI.Pack("encode", [][32]byte{inputData})
// 	if err != nil {
// 		fmt.Println("Error encoding data:", err)
// 		return
// 	}

// 	fmt.Printf("Encoded data: %x\n", encodedData)
// }

// /*
// scenario1 encode(["string"], ["hello"])
// component = [{"name":"whatever","type":"string"}]

// scenario2 encode(["string", "address"], ["hello", "0x1234"])
// component = [{"name":"whatever","type":"string"},{"name":"whatever","type":"address"}]
// */
// func MakeArg(types string, fields interface{}) (abi.Argument, interface{}) {
// 	// string, uint, address, bool, bytes, int8, int16, int32, int64, int128, int256, uint8, uint16, uint32, uint64, uint128, uint256
// 	argType, err := abi.NewType(types, types, nil)
// 	if err != nil {
// 		log.Fatalf("Failed to create new ABI type: %v", err)
// 		panic(err)
// 	}

// 	arg := abi.Argument{
// 		Type: argType,
// 	}
// 	interfaceField, err := ConvertStringToType(types, fields.(string))
// 	if err != nil {
// 		log.Fatalf("Failed to create new ABI type: %v", err)
// 		// panic(err)
// 	}
// 	return arg, interfaceField
// }

// func MakeArgTupleList(types string, fields interface{}, component []abi.ArgumentMarshaling) (abi.Argument, interface{}) {
// 	// tuple[]
// 	// convert component names to title case
// 	for i, c := range component {
// 		component[i].Name = strings.Title(c.Name)
// 	}
// 	argType, err := abi.NewType(types, types, component)
// 	if err != nil {
// 		log.Fatalf("Failed to create new ABI type: %v", err)
// 		// panic(err)
// 	}
// 	extractFields := fields.([][]interface{})

// 	var interfaceFields [][]interface{}

// 	for _, field := range extractFields {

// 		var tempField []interface{}

// 		for i, f := range field {

// 			interfaceField, err := ConvertStringToType(component[i].Type, f.(string))
// 			if err != nil {
// 				log.Fatalf("Failed to create new ABI type: %v", err)
// 				// panic(err)
// 			}

// 			tempField = append(tempField, interfaceField)
// 		}

// 		interfaceFields = append(interfaceFields, tempField)
// 	}

// 	makestruct := MakeStructs(interfaceFields, component)
// 	arg := abi.Argument{
// 		Type: argType,
// 	}
// 	return arg, makestruct.Interface()

// }
// func MakeArgTuple(types string, fields interface{}, component []abi.ArgumentMarshaling) (abi.Argument, interface{}) {
// 	// tuple[]
// 	// convert component names to title case
// 	for i, c := range component {
// 		component[i].Name = strings.Title(c.Name)
// 	}
// 	argType, err := abi.NewType(types, types, component)
// 	if err != nil {
// 		log.Fatalf("Failed to create new ABI type: %v", err)
// 		// panic(err)
// 	}
// 	extractFields := fields.([]interface{})

// 	var interfaceFields []interface{}

// 	for i, f := range extractFields {

// 		interfaceField, err := ConvertStringToType(component[i].Type, f.(string))
// 		if err != nil {
// 			log.Fatalf("Failed to create new ABI type: %v", err)
// 			// panic(err)
// 		}

// 		interfaceFields = append(interfaceFields, interfaceField)
// 	}

// 	makestruct := MakeStruct(interfaceFields, component)
// 	arg := abi.Argument{
// 		Type: argType,
// 	}
// 	return arg, makestruct.Interface()

// }

// // func Encodes(types []string, fields []interface{}, components []abi.ArgumentMarshaling) ([]byte, error) {
// // 	var arguments abi.Arguments
// // 	var inter []interface{}
// // 	for i, t := range types {
// // 		if t == "tuple[]" {
// // 			arg, interfaceFields := MakeArgTupleList(t, fields[i], components[i])
// // 			arguments = append(arguments, arg)
// // 			inter = append(inter, interfaceFields)
// // 			continue
// // 		}
// // 		if t == "tuple" {
// // 			arg, interfaceFields := MakeArgTuple(t, fields, components[i])
// // 			arguments = append(arguments, arg)
// // 			inter = append(inter, interfaceFields)
// // 			continue
// // 		}
// // 		arg, interfaceField := MakeArg(t, fields[i])

// // 		arguments = append(arguments, arg)
// // 		inter = append(inter, interfaceField)
// // 	}
// // 	return arguments.Pack(inter...)
// // }

// // https://github.com/ethereum/go-ethereum/blob/master/accounts/abi/argument.go
// func EncodeArguments(dataTypes []string, dataFields []string) ([]byte, error) {
// 	var arguments abi.Arguments

// 	interfaceFields := make([]interface{}, len(dataFields))
// 	for i, dataType := range dataTypes {
// 		argType, err := abi.NewType(dataType, dataType, nil)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to create new ABI type: %v", err)
// 		}

// 		interfaceFields[i], err = ConvertStringToType(dataType, dataFields[i])
// 		if err != nil {
// 			return nil, err
// 		}

// 		arguments = append(arguments, abi.Argument{
// 			Type: argType,
// 		})
// 	}

// 	return arguments.Pack(interfaceFields...)
// }

// func ConvertStringToType(dataType, dataField string) (interface{}, error) {
// 	// TODO: make more robust and handle multidimensional arrays
// 	if strings.Contains(dataType, "int") {
// 		if strings.HasSuffix(dataType, "[]") {
// 			dataType = "int[]"
// 		} else {
// 			dataType = "int"
// 		}
// 	}
// 	// handle bytes and fixed bytes and single arrays
// 	// if strings.Contains(dataType, "bytes") {
// 	// 	byt := []byte(dataField)
// 	// 	if dataType == "bytes" {
// 	// 		return byt, nil
// 	// 	} else {
// 	// 		// trim bytes
// 	// 		if strings.HasSuffix(dataType, "[]") {
// 	// 			dataField = strings.Trim(dataField, "[]")
// 	// 			bytesStrings := strings.Split(dataField, ",")
// 	// 			if dataType == "bytes[]" {
// 	// 				bytesSlice := make([][]byte, 0, len(bytesStrings))
// 	// 				for _, bytesStr := range bytesStrings {
// 	// 					bytesSlice = append(bytesSlice, []byte(bytesStr))
// 	// 				}
// 	// 				return bytesSlice, nil
// 	// 			} else {
// 	// 				// extract bytes size
// 	// 				dataType = strings.TrimPrefix(dataType, "bytes")
// 	// 				dataType = strings.TrimSuffix(dataType, "[]")
// 	// 				// string to int
// 	// 				size, err := strconv.Atoi(dataType)
// 	// 				if err != nil {
// 	// 					return nil, fmt.Errorf("could not parse bytes size: %s", dataType)
// 	// 				}
// 	// 				bytesSlice := make([][]byte, 0, size)
// 	// 				for _, bytesStr := range bytesStrings {
// 	// 					bytesSlice = append(bytesSlice, []byte(bytesStr))
// 	// 				}
// 	// 				return bytesSlice, nil
// 	// 			}
// 	// 		} else {
// 	// 			dataType = strings.TrimPrefix(dataType, "bytes")
// 	// 			fmt.Println("___________________")
// 	// 			// string to int
// 	// 			size, err := strconv.Atoi(dataType)
// 	// 			if err != nil {
// 	// 				return nil, fmt.Errorf("could not parse bytes size: %s", dataType)
// 	// 			}
// 	// 			fmt.Println("size: ", size)
// 	// 			b := make([]byte, size)
// 	// 			copy(b[:], byt)
// 	// 			return b, nil
// 	// 		}
// 	// 	}
// 	// }

// 	switch dataType {
// 	case "string":
// 		return dataField, nil
// 	case "string[]":
// 		dataField = strings.Trim(dataField, "[]")
// 		return []string{dataField}, nil
// 	case "bool":
// 		return strconv.ParseBool(dataField)
// 	case "bool[]":
// 		dataField = strings.Trim(dataField, "[]")
// 		// Bool
// 		boolStrings := strings.Split(dataField, ",")
// 		boolSlice := make([]bool, 0, len(boolStrings))
// 		for _, boolStr := range boolStrings {
// 			boolVal, err := strconv.ParseBool(boolStr)
// 			if err != nil {
// 				return nil, fmt.Errorf("could not parse bool string %s", boolStr)
// 			}
// 			boolSlice = append(boolSlice, boolVal)
// 		}
// 		return boolSlice, nil
// 	case "address":
// 		return common.HexToAddress(dataField), nil
// 	case "address[]":
// 		dataField = strings.Trim(dataField, "[]")
// 		// Address
// 		addressStrings := strings.Split(dataField, ",")
// 		addressSlice := make([]common.Address, 0, len(addressStrings))
// 		for _, addressStr := range addressStrings {
// 			addressSlice = append(addressSlice, common.HexToAddress(addressStr))
// 		}
// 		return addressSlice, nil
// 	case "bytes", "bytes[]":
// 		// TODO: decode bytes properly
// 		return []byte(dataField), nil
// 	case "bytes32", "bytes32[]":
// 		var b [32]byte
// 		if dataType == "bytes32" {
// 			byt := []byte(dataField)
// 			copy(b[:], byt)
// 			return b, nil
// 		} else {
// 			copy(b[:], []byte(dataField))
// 			return [][32]byte{b}, nil
// 		}
// 	case "int":
// 		// https://docs.soliditylang.org/en/latest/types.html#integers
// 		value := new(big.Int)
// 		value, success := value.SetString(dataField, 10)
// 		if !success {
// 			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("could not set string to big.Int for value %s", dataField))
// 		}
// 		return value, nil
// 	case "int[]":
// 		// Remove the brackets
// 		dataField = strings.Trim(dataField, "[]")

// 		// Split the string by commas
// 		numberStrings := strings.Split(dataField, ",")

// 		// Convert each string number to a big.Int
// 		bigIntSlice := make([]*big.Int, 0, len(numberStrings))
// 		for _, numberStr := range numberStrings {
// 			numberStr = strings.TrimSpace(numberStr) // Remove any whitespace
// 			num := new(big.Int)
// 			_, success := num.SetString(numberStr, 10) // Base 10 for decimal
// 			if !success {
// 				fmt.Printf("Error converting '%s' to big.Int\n", numberStr)
// 				continue
// 			}
// 			bigIntSlice = append(bigIntSlice, num)
// 		}
// 		return bigIntSlice, nil
// 	default:
// 		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported data type: %s", dataType))
// 	}
// }

// func VerifyDataTypeFields(querytype string, datatypes, datafields []string) error {
// 	// check if data fields is empty
// 	if len(datafields) == 0 {
// 		return status.Error(codes.InvalidArgument, fmt.Sprintf("data field mapping is empty"))
// 	}
// 	// encode query data params
// 	encodedDatafields, err := EncodeArguments(datatypes, datafields)
// 	if err != nil {
// 		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to encode data field mapping, data_type_to_field_name: %v", err))
// 	}
// 	// encode query data with query type
// 	_, err = EncodeArguments([]string{"string", "bytes"}, []string{querytype, string(encodedDatafields)})

// 	if err != nil {
// 		return fmt.Errorf("failed to encode query data: %v", err)
// 	}
// 	return nil
// }

// // has0xPrefix validates str begins with '0x' or '0X'.
// // From: https://github.com/ethereum/go-ethereum/blob/5c6f4b9f0d4270fcc56df681bf003e6a74f11a6b/common/bytes.go#L51
// func Has0xPrefix(str string) bool {
// 	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
// }

// // check queryId is valid ie 32 bytes
// func IsQueryId64chars(queryId string) bool {
// 	hasPrefix := Has0xPrefix(queryId)
// 	if hasPrefix {
// 		queryId = queryId[2:]
// 	}
// 	if len(queryId) != 64 {
// 		return false
// 	}
// 	return true
// }

// func RemoveHexPrefix(hexString string) string {
// 	if Has0xPrefix(hexString) {
// 		hexString = hexString[2:]
// 	}
// 	return hexString
// }

// // Decodes query data bytes to query type and data fields
// func DecodeQueryType(data []byte) (string, []byte, error) {
// 	// Create an ABI arguments object based on the types
// 	strArg, err := abi.NewType("string", "string", nil)
// 	if err != nil {
// 		return "", nil, fmt.Errorf("failed to create new ABI type when decoding query type: %v", err)
// 	}
// 	bytesArg, err := abi.NewType("bytes", "bytes", nil)
// 	if err != nil {
// 		return "", nil, fmt.Errorf("failed to create new ABI type when decoding query type: %v", err)
// 	}
// 	args := abi.Arguments{
// 		abi.Argument{Type: strArg},
// 		abi.Argument{Type: bytesArg},
// 	}
// 	result, err := args.UnpackValues(data)
// 	if err != nil {
// 		return "", nil, fmt.Errorf("failed to unpack query type: %v", err)
// 	}
// 	return result[0].(string), result[1].([]byte), nil
// }

// // Decodes query data bytes to query type and data fields
// func DecodeParamtypes(data []byte, types []*ABIComponent) (string, error) {
// 	var args abi.Arguments
// 	for _, t := range types {
// 		fmt.Println(types)
// 		argType, err := abi.NewType(t.Type, t.Type, nil)
// 		if err != nil {
// 			return "", fmt.Errorf("failed to create new ABI type: %v", err)
// 		}
// 		args = append(args, abi.Argument{Type: argType})
// 	}

// 	result, err := args.UnpackValues(data)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to unpack query type: %v", err)
// 	}

// 	return convertToJSON(result, types)
// }

// // convertToJSON converts a slice of interfaces into a JSON string.
// func convertToJSON(slice []interface{}, types []*ABIComponent) (string, error) {
// 	fmt.Println("slice: ", slice, " types: ", types)
// 	var items []map[string]interface{}
// 	for i, item := range slice {
// 		itemName := types[i].Name

// 		itemMap := map[string]interface{}{
// 			"name":  itemName,
// 			"value": item,
// 		}
// 		items = append(items, itemMap)
// 	}

// 	jsonResult, err := json.Marshal(items)
// 	if err != nil {
// 		return "", err
// 	}

// 	return string(jsonResult), nil
// }

// // func GenerateQuerydata(querytype string, parameters []interface{}, queryParameterTypes []string, abi []abi.ArgumentMarshaling) (string, error) {
// // 	if len(parameters) == 0 {
// // 		return "", status.Error(codes.InvalidArgument, fmt.Sprintf("data field mapping is empty"))
// // 	}
// // 	// encode query data params
// // 	encodedDatafields, err := Encodes(queryParameterTypes, parameters, abi)
// // 	if err != nil {
// // 		return "", status.Error(codes.InvalidArgument, fmt.Sprintf("failed to encode data arguments: %v", err))
// // 	}

// // 	// encode query data with query type
// // 	encodedQuerydata, err := EncodeArguments([]string{"string", "bytes"}, []string{querytype, string(encodedDatafields)})
// // 	if err != nil {
// // 		return "", fmt.Errorf("failed to encode query data: %v", err)
// // 	}
// // 	return hex.EncodeToString(encodedQuerydata), nil
// // }

// func IsValueDecodable(value, datatype string) error {
// 	argType, err := abi.NewType(datatype, datatype, nil)
// 	if err != nil {
// 		return fmt.Errorf("failed to create new ABI type when decoding value: %v", err)
// 	}
// 	arg := abi.Argument{
// 		Type: argType,
// 	}
// 	args := abi.Arguments{arg}
// 	valueBytes, err := hex.DecodeString(value)
// 	if err != nil {
// 		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode value string: %v", err))
// 	}
// 	var result []interface{}
// 	result, err = args.Unpack(valueBytes)
// 	if err != nil {
// 		return fmt.Errorf("failed to unpack value: %v", err)
// 	}
// 	fmt.Println("Decoded value: ", result[0])
// 	return nil
// }

// func MakeStructs(interfaceFields interface{}, components []abi.ArgumentMarshaling) reflect.Value {
// 	// Define the struct fields based on the ABI components
// 	structFields := make([]reflect.StructField, len(components))
// 	for i, comp := range components {
// 		fieldType, err := ConvertABIToReflectType(comp.Type)
// 		if err != nil {
// 			log.Fatalf("Failed to convert ABI type to reflect.Type: %v", err)
// 		}

// 		structFields[i] = reflect.StructField{
// 			Name: strings.Title(comp.Name),
// 			Type: fieldType,
// 		}
// 	}

// 	// Create the struct type
// 	structType := reflect.StructOf(structFields)
// 	nestedInterface := interfaceFields.([]interface{})
// 	// Create a slice to hold the structs
// 	structSlice := reflect.MakeSlice(reflect.SliceOf(structType), 0, len(nestedInterface))

// 	// Iterate over the interface fields and populate the struct slice
// 	for _, instanceFields := range nestedInterface {
// 		if len(instanceFields.([]interface{})) != len(components) {
// 			log.Fatalf("The number of instance fields must match the number of components")
// 		}
// 		newStruct := reflect.New(structType).Elem()
// 		for i, fieldValue := range instanceFields.([]interface{}) {
// 			fieldValue, err := ConvertStringToType(components[i].Type, fieldValue.(string))
// 			if err != nil {
// 				panic(fmt.Errorf("Failed to create new ABI type: %v", err))
// 			}
// 			fieldVal := newStruct.Field(i)
// 			if !fieldVal.IsValid() {
// 				log.Fatalf("No such field at index: %d", i)
// 			}

// 			val := reflect.ValueOf(fieldValue)
// 			fmt.Println(val, fieldValue, fieldVal, components[i].Type)
// 			if !val.Type().AssignableTo(fieldVal.Type()) {
// 				log.Fatalf("Type mismatch for field at index %d", i)
// 			}

// 			fieldVal.Set(val)
// 		}

// 		structSlice = reflect.Append(structSlice, newStruct)
// 	}

// 	return structSlice
// }

// func MakeStruct(interfaceFields []interface{}, components []abi.ArgumentMarshaling) reflect.Value {
// 	// Define the struct fields based on the ABI components
// 	structFields := make([]reflect.StructField, len(components))
// 	for i, comp := range components {
// 		fieldType, err := ConvertABIToReflectType(comp.Type)
// 		if err != nil {
// 			log.Fatalf("Failed to convert ABI type to reflect.Type: %v", err)
// 		}

// 		structFields[i] = reflect.StructField{
// 			Name: comp.Name,
// 			Type: fieldType,
// 		}
// 	}

// 	// Create the struct type
// 	structType := reflect.StructOf(structFields)

// 	// Iterate over the interface fields and populate the struct slice
// 	if len(interfaceFields) != len(components) {
// 		log.Fatalf("The number of instance fields must match the number of components")
// 	}
// 	newStruct := reflect.New(structType).Elem()
// 	for i, fieldValue := range interfaceFields {

// 		fieldVal := newStruct.Field(i)

// 		if !fieldVal.IsValid() {
// 			log.Fatalf("No such field at index: %d", i)
// 		}

// 		val := reflect.ValueOf(fieldValue)

// 		if !val.Type().AssignableTo(fieldVal.Type()) {
// 			log.Fatalf("Type mismatch for field at index %d", i)
// 		}

// 		fieldVal.Set(val)
// 	}

// 	return newStruct
// }

// // ConvertABIToReflectType converts ABI type strings to reflect.Type
// func ConvertABIToReflectType(abiType string) (reflect.Type, error) {
// 	if strings.Contains(abiType, "int") {
// 		abiType = "int"
// 	}
// 	switch abiType {
// 	case "string":
// 		return reflect.TypeOf(""), nil
// 	case "address":
// 		return reflect.TypeOf(common.Address{}), nil
// 	case "int":
// 		bigint := big.NewInt(1)
// 		return reflect.TypeOf(bigint), nil
// 	case "bool":
// 		return reflect.TypeOf(false), nil
// 	case "bytes":
// 		return reflect.TypeOf([]byte{}), nil
// 	default:
// 		return nil, fmt.Errorf("unsupported ABI type: %s", abiType)
// 	}
// }

// // convert after recieving from user
// func returndatafield(typ, data string) string {
// 	switch typ {
// 	case "bytes":
// 		// remove 0x prefix
// 		data = RemoveHexPrefix(data)
// 		dataBytes, err := hex.DecodeString(data)
// 		if err != nil {
// 			panic(fmt.Errorf("failed to decode bytes: %v", err))
// 		}
// 		return string(dataBytes)
// 	default:
// 		return ""
// 	}
// }
