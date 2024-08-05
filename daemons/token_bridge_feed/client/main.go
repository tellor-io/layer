package main

import (
	"fmt"

	"github.com/spf13/viper"
)

type Client struct{}

func (c *Client) getEthRpcUrl() (string, error) {
	viper.SetConfigName("secrets")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	ethRpcUrl := viper.GetString("eth_api_key")
	if ethRpcUrl == "" {
		return "", fmt.Errorf("eth_api_key not set")
	}
	return ethRpcUrl, nil
}

func main() {
	client := &Client{}
	RpcUrl, err := client.getEthRpcUrl()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(RpcUrl)
	}
}