package client

import (
	"encoding/hex"
	"fmt"
)

func (c *Client) deposits() (queryData []byte, value string, err error) {
	oldestDeposit, err := c.TokenDepositsCache.GetOldestReport()
	if err != nil {
		return nil, "", fmt.Errorf("no pending deposits")
	}

	return oldestDeposit.QueryData, hex.EncodeToString(oldestDeposit.Value), nil
}
