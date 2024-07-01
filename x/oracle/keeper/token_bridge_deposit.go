package keeper

import (
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TRBBridgeQueryType = "TRBBridge"
)

func (k Keeper) tokenBridgeDepositCheck(currentTime time.Time, queryData []byte) (types.QueryMeta, error) {
	// decode query data partial
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return types.QueryMeta{}, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return types.QueryMeta{}, err
	}
	initialArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryDataDecodedPartial, err := initialArgs.Unpack(queryData)
	if err != nil {
		return types.QueryMeta{}, err
	}
	if len(queryDataDecodedPartial) != 2 {
		return types.QueryMeta{}, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data length")
	}
	// check if first arg is a string
	if reflect.TypeOf(queryDataDecodedPartial[0]).Kind() != reflect.String {
		return types.QueryMeta{}, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data type")
	}
	if queryDataDecodedPartial[0].(string) != TRBBridgeQueryType {
		return types.QueryMeta{}, types.ErrNotTokenDeposit
	}
	// decode query data arguments
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return types.QueryMeta{}, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return types.QueryMeta{}, err
	}

	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}

	queryDataArgsDecoded, err := queryDataArgs.Unpack(queryDataDecodedPartial[1].([]byte))
	if err != nil {
		return types.QueryMeta{}, err
	}

	if len(queryDataArgsDecoded) != 2 {
		return types.QueryMeta{}, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data arguments length")
	}

	// check if first arg is a bool
	if reflect.TypeOf(queryDataArgsDecoded[0]).Kind() != reflect.Bool {
		return types.QueryMeta{}, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data arguments type")
	}

	if !queryDataArgsDecoded[0].(bool) {
		return types.QueryMeta{}, types.ErrNotTokenDeposit
	}

	queryId := utils.QueryIDFromData(queryData)
	query := types.QueryMeta{
		RegistrySpecTimeframe: time.Second,
		Expiration:            currentTime.Add(-48 * time.Hour),
		QueryType:             TRBBridgeQueryType,
		QueryId:               queryId,
		Amount:                math.NewInt(0),
	}

	return query, nil
}

// type QueryMeta struct {
//     // unique id of the query that changes after query's lifecycle ends
//     Id uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
//     // amount of tokens that was tipped
//     Amount cosmossdk_io_math.Int `protobuf:"bytes,2,opt,name=amount,proto3,customtype=cosmossdk.io/math.Int" json:"amount"`
//     // expiration time of the query
//     Expiration time.Time `protobuf:"bytes,3,opt,name=expiration,proto3,stdtime" json:"expiration"`
//     // timeframe of the query according to the data spec
//     RegistrySpecTimeframe time.Duration `protobuf:"bytes,4,opt,name=registry_spec_timeframe,json=registrySpecTimeframe,proto3,stdduration" json:"registry_spec_timeframe"`
//     // indicates whether query has revealed reports
//     HasRevealedReports bool `protobuf:"varint,5,opt,name=has_revealed_reports,json=hasRevealedReports,proto3" json:"has_revealed_reports,omitempty"`
//     // unique id of the query according to the data spec
//     QueryId []byte `protobuf:"bytes,6,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
//     // string identifier of the data spec
//     QueryType string `protobuf:"bytes,7,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
// }
