package simulation_test

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestSimulateMsgRegisterSpec(t *testing.T) {
	// initialize parameters
	s := rand.NewSource(1)
	r := rand.New(s)
	fmt.Println("r: ", r)

	// execute SimulateMsgRegisterSpec function
	registeredSpec := SimulateMsgRegisterSpec(nil, nil, nil, nil)
	fmt.Println("registeredSpec: ", registeredSpec)

}
