package combined_handler

import (
	"context"
	"fmt"
	"sync"

	contractreader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	rpcreader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

type CombinedHandler interface {
	FetchValue(
		ctx context.Context,
		contractReaders map[string]*contractreader.Reader,
		rpcReaders map[string]*rpcreader.Reader,
		priceCache *pricefeedservertypes.MarketToExchangePrices,
	) (float64, error)
}

type ParallelFetcher struct {
	mu      sync.Mutex
	results map[string]any
	errors  map[string]error
	wg      sync.WaitGroup
}

func NewParallelFetcher() *ParallelFetcher {
	return &ParallelFetcher{
		results: make(map[string]any),
		errors:  make(map[string]error),
	}
}

func (p *ParallelFetcher) FetchContract(
	ctx context.Context,
	key string,
	reader *contractreader.Reader,
	address string,
	functionSig string,
	args []string,
) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		result, err := reader.ReadContract(ctx, address, functionSig, args)

		p.mu.Lock()
		defer p.mu.Unlock()
		if err != nil {
			p.errors[key] = err
		} else {
			p.results[key] = result
		}
	}()
}

func (p *ParallelFetcher) FetchRPC(
	ctx context.Context,
	key string,
	reader *rpcreader.Reader,
) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		result, err := reader.FetchJSON(ctx)

		p.mu.Lock()
		defer p.mu.Unlock()
		if err != nil {
			p.errors[key] = err
		} else {
			p.results[key] = result
		}
	}()
}

func (p *ParallelFetcher) Wait() {
	p.wg.Wait()
}

func (p *ParallelFetcher) GetResult(key string) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err, exists := p.errors[key]; exists {
		return nil, err
	}

	result, exists := p.results[key]
	if !exists {
		return nil, fmt.Errorf("no result found for key: %s", key)
	}

	return result, nil
}

func (p *ParallelFetcher) GetBytes(key string) ([]byte, error) {
	result, err := p.GetResult(key)
	if err != nil {
		return nil, err
	}

	bytes, ok := result.([]byte)
	if !ok {
		return nil, fmt.Errorf("result for key %s is not []byte", key)
	}

	return bytes, nil
}

func (p *ParallelFetcher) GetContractBytes(key string) ([]byte, error) {
	return p.GetBytes(key)
}
