package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

// - get latest block height B1
// - get validator set at height and respective powers
// - write validator set and powers to file
// - get Multistore, merkle vals, for block B1, write to file
// - submitVal1, get block height B2, get Multistore, merkle vals, for block B2, write to file
// - get proof for submitVal1
// - submitVal2, get block height B3, get Multistore, merkle vals, for block B3, write to file
// - get proof for submitVal2
// - foundry test
// - load validator set and powers from file
// - load Multistore, merkle vals, for block B1, from file
// - relay block B1
// - relay block B2
// - run proof for submitVal1, save value in TestUserContract
// - relay block B3
// - run proof for submitVal2, save value in TestUserContract

// endpoints
// latest block: http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest
// validators: http://localhost:1317/layer/bridge/blockvalidators?height=555
// header: http://localhost:1317/layer/bridge/blockheadermerkleevm?height=1763

func main() {
	// *** get latest block number ***
	url := "http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to send request to Cosmos API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed to read response body: %v", err)
	}

	var result map[string]interface{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	block := result["block"].(map[string]interface{})
	header := block["header"].(map[string]interface{})
	height := header["height"].(string)

	log.Printf("Height: %s", height)

	// *** query block validators ***
	url = "http://localhost:1317/layer/bridge/blockvalidators?height=" + height

	resp, err = http.Get(url)
	if err != nil {
		log.Fatalf("Failed to send request to Cosmos API: %v", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	var result2 map[string]interface{}

	err = json.Unmarshal(body, &result2)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	validators := result2["Validators"].([]interface{})

	for _, validator := range validators {
		val := validator.(map[string]interface{})
		cosmosAddress := val["cosmosAddress"].(string)
		ethAddress := val["ethAddress"].(string)
		votingPower := val["votingPower"].(string)

		log.Printf("Cosmos Address: %s, Eth Address: %s, Voting Power: %s", cosmosAddress, ethAddress, votingPower)
	}

	// Replace with your desired file path
	filePath := "setup/data/validators.json"

	file, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}

	log.Printf("Response data written to %s", filePath)

	// *** query block header merkle parts ***
	// api call response body:
	// {
	// 	"blockheaderMerkleEvm": {
	// 	  "versionChainidHash": "0xeeeeae3f4b3ae79bfe8b9f2b447ee988dd3029cf8bd46d22f38f85cba12aad93",
	// 	  "height": "1763",
	// 	  "timeSecond": "1698931648",
	// 	  "timeNanosecond": 357908000,
	// 	  "lastblockidCommitHash": "0x775afc32c1c13b96261482ed3f0bfc6b489f6b9240f403e52565173266af97e8",
	// 	  "nextvalidatorConsensusHash": "0x37e845d194475c603d471d3923fd0d8e7bce5c095e60293d975cd8685e9c0018",
	// 	  "lastresultsHash": "0x9fb9c7533caf1d218da3af6d277f6b101c42e3c3b75d784242da663604dd53c2",
	// 	  "evidenceProposerHash": "0xeb571fa2353b95b6bc64d23af6dc1e47b46208274115d3b58ca7a9bcc2ddab3a"
	// 	}
	//   }
	// when saved to file, needs to conform to this solidity struct definition:
	// struct BlockHeaderMerkleParts {
	//     bytes32 versionAndChainIdHash; // [1A]
	//     uint64 height; // [2]
	//     uint64 timeSecond; // [3]
	//     uint32 timeNanoSecondFraction; // between 0 to 10^9 [3]
	//     bytes32 lastBlockIdAndOther; // [2B]
	//     bytes32 nextValidatorHashAndConsensusHash; // [1E]
	//     bytes32 lastResultsHash; // [B]
	//     bytes32 evidenceAndProposerHash; // [2D]
	// }

	url = "http://localhost:1317/layer/bridge/blockheadermerkleevm?height=" + height

	resp, err = http.Get(url)
	if err != nil {
		log.Fatalf("Failed to send request to Cosmos API: %v", err)
	}

	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	var result3 map[string]interface{}

	err = json.Unmarshal(body, &result3)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	blockheaderMerkleEvm := result3["blockheaderMerkleEvm"].(map[string]interface{})
	versionChainidHash := blockheaderMerkleEvm["versionChainidHash"].(string)
	height2 := blockheaderMerkleEvm["height"].(string)
	timeSecond := blockheaderMerkleEvm["timeSecond"].(string)
	timeNanosecondFloat := blockheaderMerkleEvm["timeNanosecond"].(float64)
	timeNanosecond := strconv.FormatFloat(timeNanosecondFloat, 'f', -1, 64)
	lastblockidCommitHash := blockheaderMerkleEvm["lastblockidCommitHash"].(string)
	nextvalidatorConsensusHash := blockheaderMerkleEvm["nextvalidatorConsensusHash"].(string)
	lastresultsHash := blockheaderMerkleEvm["lastresultsHash"].(string)
	evidenceProposerHash := blockheaderMerkleEvm["evidenceProposerHash"].(string)

	log.Printf("Version Chainid Hash: %s", versionChainidHash)
	log.Printf("Height: %s", height2)
	log.Printf("Time Second: %s", timeSecond)
	log.Printf("Time Nanosecond: %s", timeNanosecond)
	log.Printf("Lastblockid Commit Hash: %s", lastblockidCommitHash)
	log.Printf("Nextvalidator Consensus Hash: %s", nextvalidatorConsensusHash)
	log.Printf("Lastresults Hash: %s", lastresultsHash)
	log.Printf("Evidence Proposer Hash: %s", evidenceProposerHash)

	// Replace with your desired file path
	filePath = "setup/data/blockheadermerkleparts.json"

	file, err = os.Create(filePath)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}

	log.Printf("Response data written to %s", filePath)
}
