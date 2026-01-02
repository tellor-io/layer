package mint

import (
	modulev1 "github.com/tellor-io/layer/api/layer/mint"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "GetExtraRewardsRate",
					Use:       "get-extra-rewards-rate",
					Short:     "Query the effective daily extra rewards rate in loya",
					Long: `Query the effective daily extra rewards rate in loya.

If no rate has been explicitly set via governance, returns the default rate (DailyMintRate).
Use 'get-extra-rewards-pool-balance' to check if extra rewards are being distributed.`,
					Example: "$ layerd query mint get-extra-rewards-rate",
				},
				{
					RpcMethod: "GetExtraRewardsPoolBalance",
					Use:       "get-extra-rewards-pool-balance",
					Short:     "Query the extra rewards pool module account balance",
					Long: `Query the current balance of the extra_rewards_pool module account.

This pool is used to distribute additional rewards to reporters beyond the base 
time-based rewards`,
					Example: "$ layerd query mint get-extra-rewards-pool-balance",
				},
			},
		},
	}
}
