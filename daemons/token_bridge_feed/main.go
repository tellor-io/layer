package main

import (
	tokenbridgeclient "github.com/tellor-io/layer/daemons/token_bridge_feed/client"
)

func main() {
	client := &tokenbridgeclient.Client{}
	// url := "http://localhost:1317/layer/bridge/get_validator_checkpoint"

	// data, err := client.QueryAPI(url)
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return
	// }

	// var result map[string]interface{}
	// err = json.Unmarshal(data, &result)
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return
	// }

	client.QueryDepositEvents()

	// fmt.Println(result)
}
