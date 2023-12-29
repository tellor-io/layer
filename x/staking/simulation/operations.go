package simulation

// simulation operation weight constants
const (
	defaultWeightMsgCreateValidator           int = 100
	defaultWeightMsgEditValidator             int = 5
	defaultWeightMsgDelegate                  int = 100
	defaultWeightMsgUndelegate                int = 100
	defaultWeightMsgBeginRedelegate           int = 100
	defaultWeightMsgCancelUnbondingDelegation int = 100

	OpWeightMsgCreateValidator           = "op_weight_msg_create_validator"
	OpWeightMsgEditValidator             = "op_weight_msg_edit_validator"
	OpWeightMsgDelegate                  = "op_weight_msg_delegate"
	OpWeightMsgUndelegate                = "op_weight_msg_undelegate"
	OpWeightMsgBeginRedelegate           = "op_weight_msg_begin_redelegate"
	OpWeightMsgCancelUnbondingDelegation = "op_weight_msg_cancel_unbonding_delegation"
)


