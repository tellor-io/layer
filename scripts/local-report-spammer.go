package main

import (
	"fmt"
	"log"
	"os/exec"
)

var COMMAND_PATH = "/Users/caleb/layer/layerd"
var LAYER_PATH = "/Users/caleb/.layer"
var FAUCET_ADDRESS = "tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp"

func main() {
	cmd := exec.Command(COMMAND_PATH, "keys", "show", "faucet", "-a", "--keyring-backend", "test", "--home", LAYER_PATH)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed: %v\n", err)
	}

	fmt.Println("here")
	addr := string(output[:len(output)-1])

	qd := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004757364630000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "0000000000000000000000000000000000000000000000000000000004a5ba50"

	cmd = exec.Command(COMMAND_PATH, "tx", "oracle", "submit-value", addr, qd, value, "--from", addr, "--chain-id", "layertest-2", "--fees", "15loya", "--keyring-backend", "test", "--keyring-dir", LAYER_PATH, "--home", LAYER_PATH, "--node", "http://54.209.172.1:26657", "--yes")
	output, _ = cmd.CombinedOutput()
	fmt.Println(string(output))

}

func CreateNewAccountsAndFundReporters(numOfReporters int) (map[string]string, error) {
	for i := 1; i <= numOfReporters; i++ {
		key_name := fmt.Sprintf("test_reporter%d", i)
		key_path := fmt.Sprintf("%s/%s", LAYER_PATH, key_name)
		cmd := exec.Command(COMMAND_PATH, "add", key_name, "--keyring-backend", "test", "--home", key_path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("creating key failed for %s: %v", key_name, err)
		}
		fmt.Println(string(output))

		FundNewReporterAccount(key_name, key_path)

	}
}

func FundNewReporterAccount(key_name, key_path string) {
	key_address := GetAddressFromKeyName(key_name)
	cmd := exec.Command(COMMAND_PATH, "tx", "bank", "send", FAUCET_ADDRESS, key_address, "10000000loya", "--from", FAUCET_ADDRESS, "--chain-id", "layertest-2", "--keyring-dir", key_path, "--keyring-backend", "test", "--home", key_path, "--fees", "15loya", "--node", "http://54.209.172.1:26657", "--yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("sending loya to %s failed: %v", key_name, err)
	}
	fmt.Println(string(output))
}

func GetAddressFromKeyName(key_name string) string {
	cmd := exec.Command(COMMAND_PATH, "keys", "show", "faucet", "-a", "--keyring-backend", "test", "--home", LAYER_PATH)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed: %v\n", err)
	}

	fmt.Println("here")
	return string(output[:len(output)-1])
}
