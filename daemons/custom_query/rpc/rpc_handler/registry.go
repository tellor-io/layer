package rpchandler

import "fmt"

var HandlerRegistry = map[string]RpcHandler{
	"generic":                    &GenericHandler{},
	"osmosis_pool_price_handler": &OsmosisPoolPriceHandler{},
}

func GetHandler(name string) (RpcHandler, error) {
	handler, exists := HandlerRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown RPC handler: %s", name)
	}
	return handler, nil
}
