package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

var COMMAND_PATH = "/Users/caleb/layer/layerd"
var LAYER_PATH = "/Users/caleb/.layer"
var FAUCET_ADDRESS = "tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp"
var VALIDATOR_ADDRESS = "tellorvaloper1q8rzwrj56ak0z6le84r9m79zn2jj3qdll0lpxz"

var NUM_OF_REPORTERS = 50

func main() {
	reportersMap, err := CreateNewAccountsAndFundReporters(NUM_OF_REPORTERS)
	if err != nil {
		fmt.Println("Error creating accounts: ", err)
		return
	}

	prevQueryData := []byte{}

	for {
		querydata, querymeta, err := c.CurrentQuery(ctx)
		if err != nil {
			// log error
			c.logger.Error("getting current query", "error", err)
		}
		if bytes.Equal(querydata, prevQueryData) || commitedIds[querymeta.Id] {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		go func(ctx context.Context, qd []byte, qm *oracletypes.QueryMeta) {
			err := c.GenerateAndBroadcastSpotPriceReport(ctx, querydata, qm)
			if err != nil {
				c.logger.Error("Generating CycleList message", "error", err)
			}
		}(ctx, querydata, querymeta)

		err = c.WaitForBlockHeight(ctx, int64(querymeta.Expiration))
		if err != nil {
			c.logger.Error("Error waiting for block height", "error", err)
		}
	}

	qd := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004757364630000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "0000000000000000000000000000000000000000000000000000000004a5ba50"

}

func SpamReportsWithReportersMap(reportersMap map[string]string, qd string) {
	maxGoroutines := 40
	ticket := make(chan struct{}, maxGoroutines)

	value := "0000000000000000000000000000000000000000000000000000000004a5ba50"

	var wg sync.WaitGroup

	for reporter_name, addr := range reportersMap {
		wg.Add(1)
		ticket <- struct{}{} // would block if guard channel is already filled
		key_path := fmt.Sprintf("%s/%s", LAYER_PATH, reporter_name)
		go func(addr, path string) {
			defer wg.Done()
			cmd := exec.Command(COMMAND_PATH, "tx", "oracle", "submit-value", addr, qd, value, "--from", addr, "--chain-id", "layertest-2", "--fees", "15loya", "--keyring-backend", "test", "--keyring-dir", path, "--home", path, "--node", "http://54.209.172.1:26657", "--yes")
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Println("ERROR submitting value: ", err)
			}
			fmt.Println(string(output))
			<-ticket
		}(addr, key_path)
	}

	wg.Wait()
}

func CreateNewAccountsAndFundReporters(numOfReporters int) (map[string]string, error) {
	reporterMap := make(map[string]string, numOfReporters)
	for i := 1; i <= numOfReporters; i++ {
		key_name := fmt.Sprintf("test_reporter%d", i)
		key_path := fmt.Sprintf("%s/%s", LAYER_PATH, key_name)

		// Create account for reporter
		cmd := exec.Command(COMMAND_PATH, "add", key_name, "--keyring-backend", "test", "--home", key_path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("creating key failed for %s: %v\r", key_name, err)
		}
		fmt.Println(string(output))

		// send tokens to reporter from faucet
		key_address := GetAddressFromKeyName(key_name)
		cmd = exec.Command(COMMAND_PATH, "tx", "bank", "send", FAUCET_ADDRESS, key_address, "200000000loya", "--from", FAUCET_ADDRESS, "--chain-id", "layertest-2", "--keyring-dir", key_path, "--keyring-backend", "test", "--home", key_path, "--fees", "15loya", "--node", "http://54.209.172.1:26657", "--yes")
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("sending loya to %s failed: %v\r", key_name, err)
		}
		fmt.Println(string(output))

		// delegate to validator
		cmd = exec.Command(COMMAND_PATH, "tx", "staking", "delegate", VALIDATOR_ADDRESS, "150000000loya", "--from", key_address, "--chain-id", "layertest-2", "--keyring-dir", key_path, "--keyring-backend", "test", "--home", key_path, "--fees", "15loya", "--node", "http://54.209.172.1:26657", "--yes")
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("error delegating to validator with %s: %v\r", key_name, err)
		}
		fmt.Println(string(output))

		// create reporter
		cmd = exec.Command(COMMAND_PATH, "tx", "reporter", "create-reporter", "20000", "1000000", "--from", key_address, "--chain-id", "layertest-2", "--keyring-dir", key_path, "--keyring-backend", "test", "--home", key_path, "--fees", "15loya", "--node", "http://54.209.172.1:26657", "--yes")
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("error creating reporter for %s: %v\r", key_name, err)
		}
		fmt.Println(string(output))

		// add to map
		reporterMap[key_name] = key_address

	}
	return reporterMap, nil
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
