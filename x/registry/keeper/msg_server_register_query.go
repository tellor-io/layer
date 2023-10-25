package keeper

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/tellor-io/layer/x/registry/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) RegisterQuery(goCtx context.Context, msg *types.MsgRegisterQuery) (*types.MsgRegisterQueryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	storeKey := ctx.KVStore(k.storeKey)
	// check if queryType is registered
	specStore := prefix.NewStore(storeKey, types.KeyPrefix(types.SpecRegistryKey))
	if !specStore.Has([]byte(msg.QueryType)) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("query type not registered: %s", msg.QueryType))
	}
	// encode query data params
	encodedData, err := EncodeArguments(msg.DataTypes, msg.DataFields)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to encode arguments: %v", err))
	}
	// encode query data with query type
	queryData, err := EncodeArguments([]string{"string", "bytes"}, []string{msg.QueryType, string(encodedData)})
	if err != nil {
		return nil, fmt.Errorf("failed to encode query data: %v", err)
	}
	// hash query data
	queryId := crypto.Keccak256(queryData)
	queryIdHex := hex.EncodeToString(queryId)
	store := prefix.NewStore(storeKey, types.KeyPrefix(types.QueryRegistryKey))
	if store.Has(queryId) {
		return nil, fmt.Errorf("query ID %s already exists", queryIdHex)
	}
	// store query data
	store.Set(queryId, queryData)
	ctx.Logger().Info(fmt.Sprintf("Query ID: %s", queryIdHex))
	return &types.MsgRegisterQueryResponse{QueryId: queryIdHex}, nil
}

// https://github.com/ethereum/go-ethereum/blob/master/accounts/abi/argument.go
func EncodeArguments(dataTypes []string, dataFields []string) ([]byte, error) {
	var arguments abi.Arguments

	interfaceFields := make([]interface{}, len(dataFields))
	for i, dataType := range dataTypes {
		argType, err := abi.NewType(dataType, dataType, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create new ABI type: %v", err)
		}

		interfaceFields[i], err = ConvertStringToType(dataType, dataFields[i])
		if err != nil {
			return nil, err
		}

		arguments = append(arguments, abi.Argument{
			Name:    "",
			Type:    argType,
			Indexed: false,
		})
	}

	return arguments.Pack(interfaceFields...)
}

func ConvertStringToType(dataType, dataField string) (interface{}, error) {
	switch dataType {
	case "string":
		return dataField, nil
	case "bool":
		return strconv.ParseBool(dataField)
	case "address":
		// TODO: Validate address, maybe?
		return dataField, nil
	case "bytes":
		return []byte(dataField), nil
	case "int8", "int16", "int32", "int64", "int128", "int256", "uint8", "uint16", "uint32", "uint64", "uint128", "uint256":
		// https://docs.soliditylang.org/en/latest/types.html#integers
		value := new(big.Int)
		value, success := value.SetString(dataField, 10)
		if !success {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("could not set string to big.Int for value %s", dataField))
		}
		return value, nil
	default:
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported data type: %s", dataType))
	}
}
