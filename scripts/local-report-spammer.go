package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("/Users/caleb/layer/layerd", "keys", "show", "faucet", "-a", "--keyring-backend", "test", "--home", "/Users/caleb/.layer")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed: %v\n", err)
	}

	convertedAddr := string(output[:])

	addr := "tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp"
	if strings.EqualFold(addr, convertedAddr) {
		fmt.Println("these are literally the exact same and this makes no sense")
	}
	fmt.Printf("Addr: %s, Converted Addr: %s\r", addr, convertedAddr)
	// qd := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004757364630000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	// value := "0000000000000000000000000000000000000000000000000000000004a5ba50"

	// cmd = exec.Command("/Users/caleb/layer/layerd", "tx", "oracle", "submit-value", addr, qd, value, "--from", addr, "--chain-id", "layertest-2", "--fees", "15loya", "--keyring-backend", "test", "--keyring-dir", "/Users/caleb/.layer", "--home", "/Users/caleb/.layer", "--node", "http://54.209.172.1:26657", "--yes")
	// output, _ = cmd.CombinedOutput()
	// fmt.Println(string(output))
}
