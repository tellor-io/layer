package contract_handlers

import "fmt"

var HandlerRegistry = map[string]ContractHandler{
	"wsteth_handler":       &WSTETHHandler{},
	"susds_handler":        &SUSDSHandler{},
	"reth_handler":         &RocketPoolETHHandler{},
	"king_handler":         &KingHandler{},
	"yieldfi_yeth_handler": &YieldFiYeth{},
	"yieldfi_yusd_handler": &YieldFiYusd{},
}

func GetHandler(name string) (ContractHandler, error) {
	handler, exists := HandlerRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown contract handler: %s", name)
	}
	return handler, nil
}
