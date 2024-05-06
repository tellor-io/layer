package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/tellor-io/layer/x/oracle/types"
)

type Accounts struct {
	PrivateKey secp256k1.PrivKey
	Account    sdk.AccAddress
}

// func (s *KeeperTestSuite) CreateFiveTestAccounts() []Accounts {
// 	// accounts := make([]Accounts, 0, 5)
// 	// for i := 0; i < 5; i++ {
// 	// 	privKey := secp256k1.GenPrivKey()
// 	// 	accountAddress := sdk.AccAddress(privKey.PubKey().Address())
// 	// 	account := authtypes.BaseAccount{
// 	// 		Address:       accountAddress.String(),
// 	// 		PubKey:        codectypes.UnsafePackAny(privKey.PubKey()),
// 	// 		AccountNumber: uint64(i + 1),
// 	// 	}
// 	// 	existingAccount := s.accountKeeper.GetAccount(s.ctx, accountAddress)
// 	// 	if existingAccount == nil {
// 	// 		s.accountKeeper.SetAccount(s.ctx, &account)
// 	// 		accounts = append(accounts, Accounts{
// 	// 			PrivateKey: *privKey,
// 	// 			Account:    accountAddress,
// 	// 		})
// 	// 	}
// 	// }
// }
// func (s *KeeperTestSuite) TestTransfer(t *testing.T) {
// 	s.SetupTest()
// 	privKey := secp256k1.GenPrivKey()
// 	accountAddress := sdk.AccAddress(privKey.PubKey().Address())
// 	tip := sdk.NewCoin("loya", math.NewInt(1000000))
// 	ctx := context.Background()
// 	res, err := s.oracleKeeper.transfer()
// }

func ReturnTestQueryMeta(tip math.Int) types.QueryMeta {
	return types.QueryMeta{
		Id:                    1,
		Amount:                tip,
		Expiration:            time.Now().Add(1 * time.Minute),
		RegistrySpecTimeframe: 1 * time.Minute,
		HasRevealedReports:    false,
		QueryId:               []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"),
		QueryType:             "SpotPrice",
	}
}

func (s *KeeperTestSuite) TestGetQueryTip() {
	//returns a query metadata with a tip of 1 TRB
	queryMeta := ReturnTestQueryMeta(math.NewInt(1 * 1e6))
	s.oracleKeeper.Query.Set(s.ctx, queryMeta.QueryId, queryMeta)

	// test with a valid queryId
	res, err := s.oracleKeeper.GetQueryTip(s.ctx, queryMeta.QueryId)
	s.NoError(err)
	s.Equal(math.NewInt(1*1e6), res)

	// test with an invalid queryId that should return 0
	res, err = s.oracleKeeper.GetQueryTip(s.ctx, []byte("test"))
	s.NoError(err)
	s.Equal(math.NewInt(0), res)
}

func (s *KeeperTestSuite) TestGetUserTips() {

}
