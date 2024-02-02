package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func EncodeWithQuerytype(querytype string, databytes []byte) ([]byte, error) {
	strArg, err := abi.NewType("string", "string", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ABI type when encoding query type: %v", err)
	}
	bytesArg, err := abi.NewType("bytes", "bytes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ABI type when encoding with query type: %v", err)
	}
	args := abi.Arguments{
		abi.Argument{Type: strArg},
		abi.Argument{Type: bytesArg},
	}
	return args.Pack(querytype, databytes)
}
