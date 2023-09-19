package keeper

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"layer/x/querydatastorage/types"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/cosmos-sdk/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
)

func (k msgServer) AddQueryToStorage(goCtx context.Context, msg *types.MsgAddQueryToStorage) (*types.MsgAddQueryToStorageResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	queryType := msg.QueryType
	dataTypes := msg.DataTypes
	dataFields := msg.DataFields

	encodedData, err := EncodeArguments(dataTypes, dataFields)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments: %v", err)
	}

	queryData, err := EncodeArguments([]string{"string", "bytes"}, []string{queryType, string(encodedData)})
	if err != nil {
		return nil, fmt.Errorf("failed to encode query data: %v", err)
	}
	queryId := crypto.Keccak256(queryData)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.QueryDataKey))
	if store.Has(queryId) {
		return nil, fmt.Errorf("query ID %s already exists", bytes.HexBytes(queryId).String())
	}
	store.Set(queryId, queryData)
	ctx.Logger().Info(fmt.Sprintf("Query ID: %s", bytes.HexBytes(queryId).String()))
	return &types.MsgAddQueryToStorageResponse{QueryId: string(queryId)}, nil
}

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
			return nil, fmt.Errorf("could not set string to big.Int for value %s", dataField)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}
