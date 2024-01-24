package flags

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

// A struct containing the values of all flags.
type Flags struct {
	// Existing flags
	GrpcAddress string
	GrpcEnable  bool
}

// List of CLI flags.
const (
	// Cosmos flags below. These config values can be set as flags or in config.toml.
	GrpcAddress = "grpc.address"
	GrpcEnable  = "grpc.enable"
)

// Validate checks that the flags are valid.
func (f *Flags) Validate() error {
	// Validtors must have cosmos grpc services enabled.
	if !f.GrpcEnable {
		return fmt.Errorf("grpc.enable must be set to true - validating requires gRPC server")
	}
	return nil
}

// GetFlagValuesFromOptions gets values from the `AppOptions` struct which contains values
// from the command-line flags.
func GetFlagValuesFromOptions(
	appOpts servertypes.AppOptions,
) Flags {
	// Create default result.
	result := Flags{
		// These are the default values from the Cosmos flags.
		GrpcAddress: config.DefaultGRPCAddress,
		GrpcEnable:  true,
	}

	if option := appOpts.Get(GrpcAddress); option != nil {
		if v, err := cast.ToStringE(option); err == nil {
			result.GrpcAddress = v
		}
	}

	if option := appOpts.Get(GrpcEnable); option != nil {
		if v, err := cast.ToBoolE(option); err == nil {
			result.GrpcEnable = v
		}
	}

	return result
}
