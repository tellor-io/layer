package client

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	tokenbridge "github.com/tellor-io/layer/daemons/token_bridge_feed/abi"
)

type Client struct {
	// Add necessary fields
	latestDepositId uint64
}

func (c *Client) QueryAPI(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %v", err)
	}

	return body, nil
}

func (c *Client) QueryDepositEvents() {
	client, err := ethclient.Dial("ws://127.0.0.1:7545")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	contractAddress := common.HexToAddress("0x459250387162D9Fc0990058f9D8856497B1CF286")

	tbContract, err := tokenbridge.NewClient(contractAddress, client)
	if err != nil {
		log.Fatalf("Failed to instantiate a TokenBridge contract: %v", err)
	}

	logsCh := make(chan *tokenbridge.ClientDeposit)
	sub, err := tbContract.WatchDeposit(nil, logsCh)
	if err != nil {
		log.Fatalf("Failed to subscribe to Deposit events: %v", err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatalf("Error in Deposit event subscription: %v", err)
		case log := <-logsCh:
			fmt.Println("Deposit event: ", log)
		}
	}
}

func (c *Client) TestQuery() {
	url := "http://localhost:1317/oracle/params"
	body, err := c.QueryAPI(url)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}
