package contract_handlers

import "fmt"

var HandlerRegistry = map[string]ContractHandler{
	"reth_handler": &RocketPoolETHHandler{},
	"king_handler": &KingHandler{},
}

func GetHandler(name string) (ContractHandler, error) {
	handler, exists := HandlerRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown contract handler: %s", name)
	}
	return handler, nil
}
