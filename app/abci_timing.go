package app

import (
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
)

// FinalizeBlock wraps the base app's FinalizeBlock to add timing instrumentation.
// This method ONLY adds logging for timing analysis and does NOT modify any business logic.
// The app hash remains identical to the version without this wrapper.
func (app *App) FinalizeBlock(req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	startTime := time.Now()
	
	// Call the base app's FinalizeBlock - all business logic happens here
	resp, err := app.BaseApp.FinalizeBlock(req)
	
	// Log timing information for per-block analysis
	// Format: [ABCI_TIMING] height=X finalize_block_ms=Y num_txs=Z
	app.Logger().Info("[ABCI_TIMING]",
		"height", req.Height,
		"finalize_block_ms", time.Since(startTime).Milliseconds(),
		"num_txs", len(req.Txs),
	)
	
	return resp, err
}

