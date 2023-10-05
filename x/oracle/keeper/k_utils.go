package keeper

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"
)

func hashQdata(queryData string) ([]byte, error) {
	// Decode the hex-encoded input string
	qbytes, err := hex.DecodeString(queryData)
	if err != nil {
		return nil, err
	}

	return crypto.Keccak256(qbytes), nil
}

func Uint64ToBytes(num uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, num)
	return bytes
}
