package reader

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	log "github.com/sirupsen/logrus"
	"github.com/tellor-io/layer/daemons/custom_query/contracts/metrics"
	"golang.org/x/crypto/sha3"
)

type Reader struct {
	clients    []*ethClient
	mu         sync.RWMutex
	timeout    time.Duration
	maxRetries int
	retryDelay time.Duration
}

type ethClient struct {
	client  *ethclient.Client
	rpc     *rpc.Client
	url     string
	healthy bool
	mu      sync.RWMutex
}

func NewReader(urls []string, timeout int) (*Reader, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("no RPC endpoints provided")
	}

	clients := make([]*ethClient, 0, len(urls))

	for _, url := range urls {
		rpcClient, err := rpc.Dial(url)
		if err != nil {
			log.Warnf("Failed to connect to RPC endpoint %s: %v", url, err)
			continue
		}

		ethClient := &ethClient{
			client:  ethclient.NewClient(rpcClient),
			rpc:     rpcClient,
			url:     url,
			healthy: true,
		}
		clients = append(clients, ethClient)
	}

	if len(clients) == 0 {
		return nil, fmt.Errorf("failed to connect to any RPC endpoint")
	}

	reader := &Reader{
		clients:    clients,
		timeout:    time.Duration(timeout) * time.Second,
		maxRetries: 3,
		retryDelay: 100 * time.Millisecond,
	}

	return reader, nil
}

func (r *Reader) ReadContract(ctx context.Context, address string, functionSig string, args []string) ([]byte, error) {
	startTime := time.Now()
	defer func() {
		metrics.ContractCallDuration.Observe(time.Since(startTime).Seconds())
	}()

	// Parse function signature
	methodID, inputData, err := r.encodeFunctionCall(functionSig, args)
	if err != nil {
		metrics.ContractCallErrors.Inc()
		return nil, fmt.Errorf("failed to encode function call: %w", err)
	}

	addr := common.HexToAddress(address)
	callMsg := ethereum.CallMsg{
		To:   &addr,
		Data: append(methodID, inputData...),
	}

	// Try each client with retries
	var lastErr error
	for _, client := range r.getHealthyClients() {
		for retry := 0; retry <= r.maxRetries; retry++ {
			ctx, cancel := context.WithTimeout(ctx, r.timeout)
			result, err := client.client.CallContract(ctx, callMsg, nil)
			cancel()

			if err == nil {
				metrics.ContractCallSuccess.Inc()
				// Decode the result as uint256
				if len(result) == 0 {
					return nil, nil
				}
				log.Debugf("Contract call successful: address=%s, function=%s, value=%s",
					address, functionSig, result)
				return result, nil
			}

			lastErr = err
			log.Warnf("Contract call failed (attempt %d/%d): %v", retry+1, r.maxRetries+1, err)

			if retry < r.maxRetries {
				time.Sleep(r.retryDelay * time.Duration(retry+1))
			}
		}

		// Mark client as unhealthy if all retries failed
		r.markClientUnhealthy(client)
	}

	metrics.ContractCallErrors.Inc()
	return nil, fmt.Errorf("all RPC endpoints failed: %w", lastErr)
}

func (r *Reader) encodeFunctionCall(functionSig string, args []string) ([]byte, []byte, error) {
	// e.g., "getExchangeRate() returns (uint256)" -> "getExchangeRate()"
	parenIndex := strings.Index(functionSig, "(")
	if parenIndex == -1 {
		return nil, nil, fmt.Errorf("invalid function signature: %s", functionSig)
	}

	// Get the part before "returns" if it exists
	funcPart := functionSig
	if idx := strings.Index(functionSig, " returns"); idx != -1 {
		funcPart = functionSig[:idx]
	}

	// Calculate method ID (first 4 bytes of the hash)
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(funcPart))
	methodID := hash.Sum(nil)[:4]

	if len(args) == 0 {
		return methodID, nil, nil
	}

	// get parameter types
	closeParenIndex := strings.Index(funcPart, ")")
	if closeParenIndex == -1 || closeParenIndex <= parenIndex+1 {
		return methodID, nil, nil
	}

	paramString := funcPart[parenIndex+1 : closeParenIndex]
	paramTypes := strings.Split(paramString, ",")

	for i := range paramTypes {
		paramTypes[i] = strings.TrimSpace(paramTypes[i])
	}

	abiArgs := make(abi.Arguments, len(paramTypes))
	for i, paramType := range paramTypes {
		typ, err := abi.NewType(paramType, "", nil)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid parameter type %s: %w", paramType, err)
		}
		abiArgs[i] = abi.Argument{Type: typ}
	}

	values := make([]interface{}, len(args))
	for i, arg := range args {
		val, err := r.parseArgument(arg, paramTypes[i])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse argument %d: %w", i, err)
		}
		values[i] = val
	}

	encodedArgs, err := abiArgs.Pack(values...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode arguments: %w", err)
	}

	return methodID, encodedArgs, nil
}

func (r *Reader) parseArgument(arg string, paramType string) (interface{}, error) {
	switch {
	case strings.HasPrefix(paramType, "uint"):
		value := new(big.Int)
		value, ok := value.SetString(arg, 10)
		if !ok {
			return nil, fmt.Errorf("invalid uint value: %s", arg)
		}
		return value, nil
	case strings.HasPrefix(paramType, "int"):
		value := new(big.Int)
		value, ok := value.SetString(arg, 10)
		if !ok {
			return nil, fmt.Errorf("invalid int value: %s", arg)
		}
		return value, nil
	case paramType == "address":
		if !common.IsHexAddress(arg) {
			return nil, fmt.Errorf("invalid address: %s", arg)
		}
		return common.HexToAddress(arg), nil
	case paramType == "bool":
		return arg == "true" || arg == "1", nil
	case paramType == "bytes32":
		bytes, err := hex.DecodeString(strings.TrimPrefix(arg, "0x"))
		if err != nil {
			return nil, fmt.Errorf("invalid bytes32 value: %s", arg)
		}
		var bytes32 [32]byte
		copy(bytes32[:], bytes)
		return bytes32, nil
	case paramType == "string":
		return arg, nil
	case strings.HasPrefix(paramType, "bytes"):
		return hex.DecodeString(strings.TrimPrefix(arg, "0x"))
	default:
		return nil, fmt.Errorf("unsupported parameter type: %s", paramType)
	}
}

func (r *Reader) getHealthyClients() []*ethClient {
	r.mu.RLock()
	defer r.mu.RUnlock()

	healthy := make([]*ethClient, 0)
	for _, client := range r.clients {
		client.mu.RLock()
		if client.healthy {
			healthy = append(healthy, client)
		}
		client.mu.RUnlock()
	}

	if len(healthy) == 0 {
		return r.clients
	}

	return healthy
}

func (r *Reader) markClientUnhealthy(client *ethClient) {
	client.mu.Lock()
	client.healthy = false
	client.mu.Unlock()
	log.Warnf("Marked RPC client %s as unhealthy", client.url)
}

func (r *Reader) markClientHealthy(client *ethClient) {
	client.mu.Lock()
	wasUnhealthy := !client.healthy
	client.healthy = true
	client.mu.Unlock()

	if wasUnhealthy {
		log.Infof("RPC client %s is now healthy", client.url)
	}
}


func (r *Reader) Close() {
	for _, client := range r.clients {
		client.rpc.Close()
	}
}
