package e2e_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	_ "github.com/tellor-io/layer/x/dispute"
	_ "github.com/tellor-io/layer/x/mint"
	_ "github.com/tellor-io/layer/x/oracle"
	_ "github.com/tellor-io/layer/x/registry/module"
	_ "github.com/tellor-io/layer/x/reporter/module"

	sdk "github.com/cosmos/cosmos-sdk/types"

	_ "github.com/cosmos/cosmos-sdk/x/auth"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"

	_ "github.com/cosmos/cosmos-sdk/x/consensus"
	_ "github.com/cosmos/cosmos-sdk/x/distribution"
	_ "github.com/cosmos/cosmos-sdk/x/genutil"
	_ "github.com/cosmos/cosmos-sdk/x/gov"
	_ "github.com/cosmos/cosmos-sdk/x/mint"
	_ "github.com/cosmos/cosmos-sdk/x/params"
	_ "github.com/cosmos/cosmos-sdk/x/slashing"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	setup "github.com/tellor-io/layer/tests"
)

const (
	ethQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	btcQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	trbQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
)

type E2ETestSuite struct {
	suite.Suite
	Setup *setup.SharedSetup
}

func (s *E2ETestSuite) SetupTest() {
	s.Setup = &setup.SharedSetup{}
	s.Setup.SetupTest(s.T())
}

func JailValidator(Ctx sdk.Context, consensusAddress sdk.ConsAddress, validatorAddress sdk.ValAddress, k stakingkeeper.Keeper) error {
	validator, err := k.GetValidator(Ctx, validatorAddress)
	if err != nil {
		return fmt.Errorf("validator %s not found", validatorAddress)
	}

	if validator.Jailed {
		return fmt.Errorf("validator %s is already jailed", validatorAddress)
	}

	err = k.Jail(Ctx, consensusAddress)
	if err != nil {
		return err
	}

	return nil
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
