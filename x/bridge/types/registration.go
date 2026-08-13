package types

import "fmt"

// InitialRegistrationMessages returns the two fixed messages a validator signs
// once to register an EVM key.
func InitialRegistrationMessages(operatorAddress string) (msgA, msgB string) {
	msgA = fmt.Sprintf("TellorLayer: Initial bridge signature A for operator %s", operatorAddress)
	msgB = fmt.Sprintf("TellorLayer: Initial bridge signature B for operator %s", operatorAddress)
	return msgA, msgB
}
