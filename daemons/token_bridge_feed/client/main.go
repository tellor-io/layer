package main

import (
	"fmt"

	"github.com/spf13/viper"
)

type Client struct{}

func (c *Client)  getEthRpcUrl() (string, error) {
	viper.SetConfigName("secrets")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	ethRpcUrl := viper.GetString("eth_rpc_url")
	if ethRpcUrl == "" {
		return "", fmt.Errorf("eth_rpc_url not set")
	}
	return ethRpcUrl, nil
}

func main() {
	client := &Client{}
	ethRpcUrl, err := client.getEthRpcUrl()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(ethRpcUrl)
	}
}