package bridge

import (
	"context"
	"encoding/hex"
	fmt "fmt"
	"strings"

	"github.com/cometbft/cometbft/libs/protoio"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cometbft "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func (s bridgeServer) BlockValidatorInfo(aContext context.Context, req *QueryBlockValidatorInfoRequest) (*QueryBlockValidatorInfoResponse, error) {
	commit, err := s.getCommit(req.Height)
	if err != nil {
		return nil, err
	}

	ethAddrs, cosmosAddrs, err := getAddresses(&commit.SignedHeader)
	if err != nil {
		return nil, err
	}

	logCtx := sdk.UnwrapSDKContext(aContext)
	s.Logger(logCtx).Error(fmt.Sprintf("ethAddrs %v", ethAddrs))
	s.Logger(logCtx).Error(fmt.Sprintf("cosmosAddrs %v", cosmosAddrs))

	votingPowers, err := getValidatorVotingPowers(aContext, &s, req.Height)
	if err != nil {
		return nil, err
	}
	s.Logger(logCtx).Error(fmt.Sprintf("votingPowers %v", votingPowers))

	validators := make([]ValidatorInfo, len(cosmosAddrs))
	for i, cosmosAddr := range cosmosAddrs {
		validators[i] = ValidatorInfo{
			CosmosAddress: cosmosAddr,
			EthAddress:    ethAddrs[i],
			VotingPower:   votingPowers[cosmosAddr],
		}
	}

	return &QueryBlockValidatorInfoResponse{
		Validators: validators,
	}, nil
}

func getAddresses(info *cometbft.SignedHeader) ([]string, []string, error) {
	ethAddresses := []string{}
	cosmosAddresses := []string{}

	prefix, err := GetPrefix(tmproto.SignedMsgType(info.Commit.Type()), info.Commit.Height, int64(info.Commit.Round))
	if err != nil {
		return nil, nil, err
	}

	prefix = append(prefix, []byte{34, 72, 10, 32}...)

	suffix, err := protoio.MarshalDelimited(
		&tmproto.CanonicalPartSetHeader{
			Total: info.Commit.BlockID.PartSetHeader.Total,
			Hash:  info.Commit.BlockID.PartSetHeader.Hash,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	suffix = append([]byte{18}, suffix...)

	commonVote := CommonEncodedVotePart{SignedDataPrefix: prefix, SignedDataSuffix: suffix}

	commonPart := append(commonVote.SignedDataPrefix, info.Commit.BlockID.Hash...)
	commonPart = append(commonPart, commonVote.SignedDataSuffix...)

	chainIDBytes := []byte(info.ChainID)
	encodedChainIDConstant := append([]byte{50, uint8(len(chainIDBytes))}, chainIDBytes...)

	for _, vote := range info.Commit.Signatures {
		if !vote.ForBlock() {
			continue
		}

		encodedTimestamp := encodeTime(vote.Timestamp)

		msg := append(commonPart, []byte{42, uint8(len(encodedTimestamp))}...)
		msg = append(msg, encodedTimestamp...)
		msg = append(msg, encodedChainIDConstant...)
		msg = append([]byte{uint8(len(msg))}, msg...)

		addr, _, err := recoverETHAddress(msg, vote.Signature, vote.ValidatorAddress)

		if err != nil {
			return nil, nil, err
		}
		formattedEthAddr := toChecksumAddress(hex.EncodeToString([]byte(addr)))
		ethAddresses = append(ethAddresses, formattedEthAddr)
		cosmosAddresses = append(cosmosAddresses, vote.ValidatorAddress.String())
	}
	if len(ethAddresses) == 0 {
		return nil, nil, fmt.Errorf("no valid precommit")
	}

	return ethAddresses, cosmosAddresses, nil
}

func getValidatorVotingPowers(goContext context.Context, b *bridgeServer, height int64) (map[string]int64, error) {
	ctx := sdk.UnwrapSDKContext(goContext)

	result, err := b.clientCtx.Client.Validators(ctx, &height, nil, nil)
	if err != nil {
		return nil, err
	}

	votingPowers := make(map[string]int64)
	for _, validator := range result.Validators {
		votingPowers[validator.Address.String()] = validator.VotingPower
	}

	return votingPowers, nil
}

func toChecksumAddress(address string) string {
	address = strings.TrimPrefix(address, "0x")
	addressHash := common.Bytes2Hex(crypto.Keccak256([]byte(strings.ToLower(address))))
	checksumAddress := "0x"
	for i, c := range address {
		if '0' <= addressHash[i] && addressHash[i] <= '7' {
			checksumAddress += strings.ToLower(string(c))
		} else {
			checksumAddress += strings.ToUpper(string(c))
		}
	}
	return checksumAddress
}
