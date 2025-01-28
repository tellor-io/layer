package types

import (
	"fmt"

	"gopkg.in/yaml.v2"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyTeamAddress     = []byte("TeamAddress")
	DefaultTeamAddress = authtypes.NewModuleAddress("trbFakeAddress") // TODO: Determine the default value
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	team sdk.AccAddress,
) Params {
	return Params{
		TeamAddress: team.Bytes(),
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultTeamAddress,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyTeamAddress, &p.TeamAddress, validateTeamAddress),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateTeamAddress(p.TeamAddress); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

func validateTeamAddress(v interface{}) error {
	_, ok := v.([]byte)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}
