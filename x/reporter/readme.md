# Readme

mocks

cd into reporter/mocks
make mock-gen

Transactions:

MsgCreateReporter
MsgDelegateReporter
UndelegateReporter
WithdrawReporterCommission
WithdrawDelegatorReward

Getters:
Params
Reporters - get all staked reporters
DelegatorReporter - get reporter a delegator is staked with.
ReporterStake - get a reporter's total tokens.
DelegationRewards - get rewards of a delegator
ReporterOutstandingRewards - get all outstanding rewards for a reporter
ReporterCommission - get a reporter's commission reward
