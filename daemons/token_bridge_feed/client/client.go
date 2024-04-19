package client

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	tokenbridge "github.com/tellor-io/layer/daemons/token_bridge_feed/abi"
)

type Client struct {
	// Add necessary fields
	lastReportedDepositId *big.Int
	pendingReports        []DepositReport
}

type DepositReceipt struct {
	DepositId *big.Int
	Sender    common.Address
	Recipient string
	Amount    *big.Int
}

type DepositReport struct {
	QueryData []byte
	Value     string
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

func (c *Client) QueryBridgeDeposits() {
	client, err := ethclient.Dial("ws://127.0.0.1:7545")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	contractAddress := common.HexToAddress("0x47F2853f1f85E2c3E3194cC3152769E3c9900a4e")

	tbContract, err := tokenbridge.NewTokenBridge(contractAddress, client)
	if err != nil {
		log.Fatalf("Failed to instantiate a TokenBridge contract: %v", err)
	}

	latestDepositId, err := c.QueryCurrentDepositId(tbContract)
	if err != nil {
		log.Fatalf("Failed to query the latest deposit ID: %v", err)
	}

	if c.lastReportedDepositId == nil {
		c.lastReportedDepositId = big.NewInt(0)
	}

	if latestDepositId.Uint64() > c.lastReportedDepositId.Uint64() {
		c.lastReportedDepositId = big.NewInt(int64(c.lastReportedDepositId.Uint64() + 1))

		depositTicket, err := c.QueryDepositDetails(tbContract, c.lastReportedDepositId)
		if err != nil {
			log.Fatalf("Failed to query deposit details: %v", err)
		}

		// assemble and add to pending reports
		queryData, err := c.EncodeQueryData(depositTicket)
		if err != nil {
			log.Fatalf("Failed to encode query data: %v", err)
		}
		reportValue, err := c.EncodeReportValue(depositTicket)
		if err != nil {
			log.Fatalf("Failed to encode report value: %v", err)
		}
		c.pendingReports = append(c.pendingReports, DepositReport{queryData, reportValue})
	}

}

func (c *Client) QueryCurrentDepositId(contract *tokenbridge.TokenBridge) (*big.Int, error) {
	// Query the latest deposit ID from the bridge contract
	latestDepositId, err := contract.DepositId(nil)
	if err != nil {
		return latestDepositId, fmt.Errorf("failed to query latest deposit ID: %v", err)
	}

	return latestDepositId, nil
}

func (c *Client) QueryDepositDetails(contract *tokenbridge.TokenBridge, depositId *big.Int) (DepositReceipt, error) {
	// Query depositDetails details for a specific depositDetails ID
	depositDetails, err := contract.Deposits(nil, depositId)
	if err != nil {
		return DepositReceipt{}, fmt.Errorf("failed to query deposit details for ID %d: %v", depositId, err)
	}

	depositReceipt := DepositReceipt{
		DepositId: depositId,
		Sender:    depositDetails.Sender,
		Recipient: depositDetails.Recipient,
		Amount:    depositDetails.Amount,
	}

	return depositReceipt, nil
}

func (c *Client) EncodeQueryData(depositReceipt DepositReceipt) ([]byte, error) {
	// encode query data
	queryTypeString := "TRBBridge"
	toLayerBool := true
	// prepare encoding
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return nil, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	// encode query data arguments first
	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}
	queryDataArgsEncoded, err := queryDataArgs.Pack(toLayerBool, depositReceipt.DepositId)
	if err != nil {
		return nil, err
	}

	// encode query data
	finalArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryDataEncoded, err := finalArgs.Pack(queryTypeString, queryDataArgsEncoded)
	if err != nil {
		return nil, err
	}
	return queryDataEncoded, nil
}

func (c *Client) EncodeReportValue(depositReceipt DepositReceipt) (string, error) {
	// replicate solidity encoding, abi.encode(address ethSender, string layerRecipient, uint256 amount)

	// prepare encoding
	AddressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return "", err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return "", err
	}
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return "", err
	}

	reportValueArgs := abi.Arguments{
		{Type: AddressType},
		{Type: StringType},
		{Type: Uint256Type},
	}

	// encode report value arguments
	reportValueArgsEncoded, err := reportValueArgs.Pack(depositReceipt.Sender, depositReceipt.Recipient, depositReceipt.Amount)
	if err != nil {
		return "", err
	}

	reportValString := hex.EncodeToString(reportValueArgsEncoded)

	return reportValString, nil
}

func (c *Client) GetPendingBridgeDeposit() (DepositReport, error) {
	if len(c.pendingReports) == 0 {
		return DepositReport{}, fmt.Errorf("no pending bridge deposits")
	}

	report := c.pendingReports[0]
	c.pendingReports = c.pendingReports[1:]
	return report, nil
}
