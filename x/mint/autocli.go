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
					RpcMethod:      "GetExtraRewardsRate",
					Use:            "get-extra-rewards-rate",
					Short:          "Query extra rewards rate (loya/day)",
					Long:           "Query the effective daily extra rewards rate in loya. Use get-extra-rewards-pool-balance to check if extra rewards are being distributed.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetExtraRewardsPoolBalance",
					Use:            "get-extra-rewards-pool-balance",
					Short:          "Query extra rewards pool balance",
					Long:           "Query the current balance of the extra_rewards_pool module account.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
			},
		},
	}
}
