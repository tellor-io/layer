package keeper

// set commit report by reporter and query id
// func (k Keeper) SetCommitReport(ctx sdk.Context, reporter sdk.AccAddress, commit *types.CommitReport) {
// store := k.CommitStore(ctx)
// store.Set(append(reporter, commit.Report.QueryId...), k.cdc.MustMarshal(commit))
// Append commit report to reports for current block
// blockKey := types.NumKey(commit.Block)
// bz := store.Get(blockKey)
// var blockReports types.CommitsByHeight
// k.cdc.MustUnmarshal(bz, &blockReports)
// blockReports.Commits = append(blockReports.Commits, commit.Report)
// TODO: figure out if we need an index here
// store.Set(blockKey, k.cdc.MustMarshal(&blockReports))

// Delete last blocks reports
// store.Delete(types.NumKey(commit.Block - 1))
// }
