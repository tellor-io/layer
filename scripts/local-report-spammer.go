package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

var COMMAND_PATH = "/Users/caleb/layer/layerd"
var LAYER_PATH = "/Users/caleb/.layer"
var FAUCET_ADDRESS = "tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp"
var VALIDATOR_ADDRESS = "tellorvaloper1dct4uwgcfjxqaphjmfzjv2yz733n9fycxdz2m6"

var NUM_OF_REPORTERS = 50

func main() {
	reportersMap, err := CreateNewAccountsAndFundReporters(NUM_OF_REPORTERS)
	if err != nil {
		fmt.Println("Error creating accounts: ", err)
		return
	}

	prevQueryData := ""

	for {

		querydata, err := GetCurrentQueryInCyclelist()
		if err != nil {
			// log error
			fmt.Println("error getting current query: ", err)
			return
		}
		if strings.EqualFold(querydata, prevQueryData) {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		// call spam function
		fmt.Println("Calling spam reports")
		go SpamReportsWithReportersMap(reportersMap, querydata)
		prevQueryData = querydata
	}

}

type CyclelistQueryResponse struct {
	Query_data []byte                `json:"query_data"`
	Query_meta oracletypes.QueryMeta `json:"query_meta"`
}

func GetCurrentQueryInCyclelist() (string, error) {
	cmd := exec.Command(COMMAND_PATH, "query", "oracle", "current-cyclelist-query", "--node", "http://54.209.172.1:26657")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("ERROR getting current cyclelist query: ", err)
		return "", nil
	}

	queryDataField := strings.Split(string(output), "query_meta")[0]
	query_data := strings.TrimPrefix(queryDataField, "query_data: ")
	qdBytes := []byte(query_data)
	qd := string(qdBytes[:len(qdBytes)-1])

	return qd, nil
}

func SpamReportsWithReportersMap(reportersMap map[string]ReporterInfo, qd string) {
	maxGoroutines := 40
	ticket := make(chan struct{}, maxGoroutines)

	value := "0000000000000000000000000000000000000000000000000000000004a5ba50"

	var wg sync.WaitGroup

	for reporter_name, info := range reportersMap {
		wg.Add(1)
		ticket <- struct{}{} // would block if guard channel is already filled
		key_path := fmt.Sprintf("%s/%s", LAYER_PATH, reporter_name)
		go func(reporter_info *ReporterInfo, path string) {
			defer wg.Done()
			cmd := exec.Command(COMMAND_PATH, "tx", "oracle", "submit-value", reporter_info.Address, qd, value, "--from", reporter_info.Address, "--chain-id", "layertest-2", "--fees", "15loya", "--keyring-backend", "test", "--keyring-dir", path, "--sequence", strconv.Itoa(reporter_info.SequenceNum), "--home", path, "--node", "http://54.209.172.1:26657", "--yes")
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Println("ERROR submitting value: ", err)
			}
			reporter_info.SequenceNum++
			fmt.Println(string(output))
			<-ticket
		}(&info, key_path)
	}

	wg.Wait()
}

type ReporterInfo struct {
	Address     string
	SequenceNum int
}

func CreateNewAccountsAndFundReporters(numOfReporters int) (map[string]ReporterInfo, error) {
	reporterMap := make(map[string]ReporterInfo, numOfReporters)
	for i := 1; i <= numOfReporters; i++ {
		key_name := fmt.Sprintf("test_reporter%d", i)
		key_path := fmt.Sprintf("%s/%s", LAYER_PATH, key_name)

		// Create account for reporter
		fmt.Println("Create keys")
		cmd := exec.Command(COMMAND_PATH, "keys", "add", key_name, "--keyring-backend", "test", "--home", key_path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(output))
			log.Fatalf("creating key failed for %s: %v\r", key_name, err)
		}
		fmt.Println(string(output))

		// send tokens to reporter from faucet
		fmt.Println("fund account from faucet")
		key_address := GetAddressFromKeyName(key_name)
		cmd = exec.Command(COMMAND_PATH, "tx", "bank", "send", "tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp", key_address, "200000000loya", "--from", "tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp", "--chain-id", "layertest-2", "--keyring-dir", "/Users/caleb/.layer", "--keyring-backend", "test", "--home", "/Users/caleb/.layer", "--fees", "15loya", "--node", "http://54.209.172.1:26657", "--yes")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(output))
			log.Fatalf("sending loya to %s failed: %v\r", key_name, err)
		}
		fmt.Println(string(output))
		fmt.Printf("Val address: %s, Key address: %s, Key path: %s\r", VALIDATOR_ADDRESS, key_address, key_path)
		time.Sleep(5 * time.Second)

		// delegate to validator
		//_ = GetSequenceNumberForAccount(key_address)

		fmt.Println("delegate to validator")
		cmd = exec.Command(COMMAND_PATH, "tx", "staking", "delegate", VALIDATOR_ADDRESS, "150000000loya", "--from", key_address, "--chain-id", "layertest-2", "--keyring-dir", fmt.Sprintf("%s/%s", LAYER_PATH, key_name), "--keyring-backend", "test", "--home", fmt.Sprintf("%s/%s", LAYER_PATH, key_name), "--fees", "15loya", "--node", "http://54.209.172.1:26657", "--yes")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(output))
			log.Fatalf("error delegating to validator with %s: %v\r", key_name, err)
		}
		fmt.Println(string(output))
		time.Sleep(5 * time.Second)

		// create reporter
		fmt.Println("create reporter ")
		cmd = exec.Command(COMMAND_PATH, "tx", "reporter", "create-reporter", "20000", "1000000", "--from", key_address, "--chain-id", "layertest-2", "--keyring-dir", key_path, "--keyring-backend", "test", "--sequence", "1", "--home", key_path, "--fees", "15loya", "--sequence", "1", "--node", "http://54.209.172.1:26657", "--yes")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(output))
			log.Fatalf("error creating reporter for %s: %v\r", key_name, err)
		}
		fmt.Println(string(output))
		time.Sleep(2 * time.Second)

		// add to map
		reporterMap[key_name] = ReporterInfo{Address: key_address, SequenceNum: 2}

	}
	return reporterMap, nil
}

func GetAddressFromKeyName(key_name string) string {
	cmd := exec.Command(COMMAND_PATH, "keys", "show", key_name, "-a", "--keyring-backend", "test", "--home", fmt.Sprintf("%s/%s", LAYER_PATH, key_name))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed: %v\n", err)
	}

	return string(output[:len(output)-1])
}

// func GetSequenceNumberForAccount(address string) string {
// 	cmd := exec.Command(COMMAND_PATH, "query", "auth", "account-info", address, "--node", "http://54.209.172.1:26657")
// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		fmt.Println("Error getting account info: ", err)
// 		panic(err)
// 	}
// 	response := string(output)
// 	fmt.Println(response)
// 	resArr := strings.Split(response, "sequence: ")
// 	fmt.Println("Account number: ", resArr[1])
// 	return resArr[1]
// }