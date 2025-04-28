package fork_test

import (
	"github.com/cosmos/cosmos-sdk/codec/types"
)

type ModuleStateData struct {
	TipperTotal   []TipperTotalData `json:"tipper_total"`
	TotalTips     TotalTipsData     `json:"total_tips"`
	TippedQueries []types.QueryMeta `json:"tipped_queries"`
}
