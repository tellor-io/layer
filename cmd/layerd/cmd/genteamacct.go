package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"cosmossdk.io/errors"
	"github.com/spf13/cobra"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdkruntime "github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankexported "github.com/cosmos/cosmos-sdk/x/bank/exported"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/genutil/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AddGenesisAccountCmd returns add-genesis-account cobra Command.
func AddTeamAccountCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-team-account [address_or_key_name]",
		Short: "Add a team account to genesis.json,",
		Long: `Add a team account to genesis.json. The provided account must specify
		the account address or key name. If a key name is given,
		the address will be looked up in the local Keybase.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			config.SetRoot(clientCtx.HomeDir)

			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				inBuf := bufio.NewReader(cmd.InOrStdin())
				keyringBackend, err := cmd.Flags().GetString(flags.FlagKeyringBackend)
				if err != nil {
					return err
				}

				// attempt to lookup address from Keybase if no address was provided
				kb, err := keyring.New(sdk.KeyringServiceName(), keyringBackend, clientCtx.HomeDir, inBuf, cdc)
				if err != nil {
					return err
				}

				info, err := kb.Key(args[0])
				if err != nil {
					return fmt.Errorf("failed to get address from Keybase: %w", err)
				}

				addr, err = info.GetAddress()
				if err != nil {
					return fmt.Errorf("failed to get address from Keybase: %w", err)
				}
			}

			// create concrete account type based on input parameters
			var genAccount authtypes.GenesisAccount

			baseAccount := authtypes.NewBaseAccount(addr, nil, 0, 0)

			genAccount = baseAccount

			if err := genAccount.Validate(); err != nil {
				return fmt.Errorf("failed to validate new genesis account: %w", err)
			}

			genFile := config.GenesisFile()
			appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
			if err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			authGenState := authtypes.GetGenesisStateFromAppState(cdc, appState)

			accs, err := authtypes.UnpackAccounts(authGenState.Accounts)
			if err != nil {
				return fmt.Errorf("failed to get accounts from any: %w", err)
			}

			if accs.Contains(addr) {
				return fmt.Errorf("cannot add account at existing address %s", addr)
			}

			// Add the new account to the set of genesis accounts and sanitize the
			// accounts afterwards.
			accs = append(accs, genAccount)
			accs = authtypes.SanitizeGenesisAccounts(accs)

			genAccs, err := authtypes.PackAccounts(accs)
			if err != nil {
				return fmt.Errorf("failed to convert accounts into any's: %w", err)
			}

			authGenState.Accounts = genAccs

			authGenStateBz, err := cdc.MarshalJSON(&authGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal auth genesis state: %w", err)
			}

			appState[authtypes.ModuleName] = authGenStateBz

			disputeGenState := disputetypes.GetGenesisStateFromAppState(cdc, appState)
			disputeGenState.Params.TeamAddress = genAccount.GetAddress()

			disputeGenStateBz, err := cdc.MarshalJSON(disputeGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal bank genesis state: %w", err)
			}

			appState[disputetypes.ModuleName] = disputeGenStateBz

			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return fmt.Errorf("failed to marshal application genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON
			return genutil.ExportGenesisFile(genDoc, genFile)
		},
	}

	cmd.Flags().String(flags.FlagKeyringBackend, flags.DefaultKeyringBackend, "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

const flagGenTxDir string = "gentx-dir"

type printInfo struct {
	Moniker    string          `json:"moniker" yaml:"moniker"`
	ChainID    string          `json:"chain_id" yaml:"chain_id"`
	NodeID     string          `json:"node_id" yaml:"node_id"`
	GenTxsDir  string          `json:"gentxs_dir" yaml:"gentxs_dir"`
	AppMessage json.RawMessage `json:"app_message" yaml:"app_message"`
}

func newPrintInfo(moniker, chainID, nodeID, genTxsDir string, appMessage json.RawMessage) printInfo {
	return printInfo{
		Moniker:    moniker,
		ChainID:    chainID,
		NodeID:     nodeID,
		GenTxsDir:  genTxsDir,
		AppMessage: appMessage,
	}
}

func displayInfo(info printInfo) error {
	out, err := json.MarshalIndent(info, "", " ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(os.Stderr, "%s\n", out)

	return err
}

func CollectGenTxsWithDelegateCmd(genBalIterator genutiltypes.GenesisBalancesIterator, defaultNodeHome string, validator genutiltypes.MessageValidator, valAddrCodec sdkruntime.ValidatorAddressCodec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect-gentxs-delegate",
		Short: "Collect genesis txs and output a genesis file that allows for Delegate gentxs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec

			config.SetRoot(clientCtx.HomeDir)

			nodeID, valPubKey, err := genutil.InitializeNodeValidatorFiles(config)
			if err != nil {
				return errors.Wrap(err, "failed to initialize node validator files")
			}

			appGenesis, err := genutiltypes.AppGenesisFromFile(config.GenesisFile())
			if err != nil {
				return errors.Wrap(err, "failed to read genesis doc from file")
			}

			genTxDir, _ := cmd.Flags().GetString(flagGenTxDir)
			genTxsDir := genTxDir
			if genTxsDir == "" {
				genTxsDir = filepath.Join(config.RootDir, "config", "gentx")
			}

			toPrint := newPrintInfo(config.Moniker, appGenesis.ChainID, nodeID, genTxsDir, json.RawMessage(""))
			initCfg := genutiltypes.NewInitConfig(appGenesis.ChainID, genTxsDir, nodeID, valPubKey)

			appMessage, err := genutil.GenAppStateFromConfig(cdc, clientCtx.TxConfig, config, initCfg, appGenesis, genBalIterator, validator, valAddrCodec)
			if err != nil {
				return errors.Wrap(err, "failed to get genesis app state from config")
			}

			toPrint.AppMessage = appMessage

			return displayInfo(toPrint)
		},
	}

	return cmd
}

func GenAppStateFromConfig(cdc codec.JSONCodec, txEncodingConfig client.TxEncodingConfig,
	config *cfg.Config, initCfg genutiltypes.InitConfig, genesis *genutiltypes.AppGenesis, genBalIterator genutiltypes.GenesisBalancesIterator,
	validator genutiltypes.MessageValidator, valAddrCodec sdkruntime.ValidatorAddressCodec,
) (appState json.RawMessage, err error) {
	// process genesis transactions, else create default genesis.json
	appGenTxs, persistentPeers, err := CollectTxs(
		cdc, txEncodingConfig.TxJSONDecoder(), config.Moniker, initCfg.GenTxsDir, genesis, genBalIterator, validator, valAddrCodec)
	if err != nil {
		return appState, err
	}

	config.P2P.PersistentPeers = persistentPeers
	cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)

	// if there are no gen txs to be processed, return the default empty state
	if len(appGenTxs) == 0 {
		return appState, fmt.Errorf("there must be at least one genesis tx")
	}

	// create the app state
	appGenesisState, err := genutiltypes.GenesisStateFromAppGenesis(genesis)
	if err != nil {
		return appState, err
	}

	appGenesisState, err = genutil.SetGenTxsInAppGenesisState(cdc, txEncodingConfig.TxJSONEncoder(), appGenesisState, appGenTxs)
	if err != nil {
		return appState, err
	}

	appState, err = json.MarshalIndent(appGenesisState, "", "  ")
	if err != nil {
		return appState, err
	}

	genesis.AppState = appState
	err = genutil.ExportGenesisFile(genesis, config.GenesisFile())

	return appState, err
}

func CollectTxs(cdc codec.JSONCodec, txJSONDecoder sdk.TxDecoder, moniker, genTxsDir string,
	genesis *types.AppGenesis, genBalIterator types.GenesisBalancesIterator,
	validator types.MessageValidator, valAddrCodec sdkruntime.ValidatorAddressCodec,
) (appGenTxs []sdk.Tx, persistentPeers string, err error) {
	// prepare a map of all balances in genesis state to then validate
	// against the validators addresses
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(genesis.AppState, &appState); err != nil {
		return appGenTxs, persistentPeers, err
	}

	var fos []os.DirEntry
	fos, err = os.ReadDir(genTxsDir)
	if err != nil {
		return appGenTxs, persistentPeers, err
	}

	balancesMap := make(map[string]bankexported.GenesisBalance)

	genBalIterator.IterateGenesisBalances(
		cdc, appState,
		func(balance bankexported.GenesisBalance) (stop bool) {
			addr := balance.GetAddress()
			balancesMap[addr] = balance
			return false
		},
	)

	// addresses and IPs (and port) validator server info
	var addressesIPs []string

	for _, fo := range fos {
		if fo.IsDir() {
			continue
		}
		if !strings.HasSuffix(fo.Name(), ".json") {
			continue
		}

		// get the genTx
		jsonRawTx, err := os.ReadFile(filepath.Join(genTxsDir, fo.Name()))
		if err != nil {
			return appGenTxs, persistentPeers, err
		}

		genTx, err := types.ValidateAndGetGenTx(jsonRawTx, txJSONDecoder, validator)
		if err != nil {
			return appGenTxs, persistentPeers, err
		}

		appGenTxs = append(appGenTxs, genTx)

		// the memo flag is used to store
		// the ip and node-id, for example this may be:
		// "528fd3df22b31f4969b05652bfe8f0fe921321d5@192.168.2.37:26656"

		memoTx, ok := genTx.(sdk.TxWithMemo)
		if !ok {
			return appGenTxs, persistentPeers, fmt.Errorf("expected TxWithMemo, got %T", genTx)
		}
		nodeAddrIP := memoTx.GetMemo()

		// genesis transactions must be single-message
		msgs := genTx.GetMsgs()

		// TODO abstract out staking message validation back to staking
		switch msgs[0].(type) {
		case (*stakingtypes.MsgCreateValidator):
			msg := msgs[0].(*stakingtypes.MsgCreateValidator)
			// validate validator addresses and funds against the accounts in the state
			valAddr, err := valAddrCodec.StringToBytes(msg.ValidatorAddress)
			if err != nil {
				return appGenTxs, persistentPeers, err
			}

			valAccAddr := sdk.AccAddress(valAddr).String()

			delBal, delOk := balancesMap[valAccAddr]
			if !delOk {
				_, file, no, ok := runtime.Caller(1)
				if ok {
					fmt.Printf("CollectTxs-1, called from %s#%d\n", file, no)
				}

				return appGenTxs, persistentPeers, fmt.Errorf("account %s balance not in genesis state: %+v", valAccAddr, balancesMap)
			}

			_, valOk := balancesMap[sdk.AccAddress(valAddr).String()]
			if !valOk {
				_, file, no, ok := runtime.Caller(1)
				if ok {
					fmt.Printf("CollectTxs-2, called from %s#%d - %s\n", file, no, sdk.AccAddress(msg.ValidatorAddress).String())
				}
				return appGenTxs, persistentPeers, fmt.Errorf("account %s balance not in genesis state: %+v", valAddr, balancesMap)
			}

			if delBal.GetCoins().AmountOf(msg.Value.Denom).LT(msg.Value.Amount) {
				return appGenTxs, persistentPeers, fmt.Errorf(
					"insufficient fund for delegation %v: %v < %v",
					delBal.GetAddress(), delBal.GetCoins().AmountOf(msg.Value.Denom), msg.Value.Amount,
				)
			}

			// exclude itself from persistent peers
			if msg.Description.Moniker != moniker {
				addressesIPs = append(addressesIPs, nodeAddrIP)
			}
		case (*stakingtypes.MsgDelegate):
			msg := msgs[0].(*stakingtypes.MsgDelegate)
			// validate validator addresses and funds against the accounts in the state
			valAddr, err := valAddrCodec.StringToBytes(msg.ValidatorAddress)
			if err != nil {
				return appGenTxs, persistentPeers, err
			}

			valAccAddr := sdk.AccAddress(valAddr).String()

			delBal, delOk := balancesMap[valAccAddr]
			if !delOk {
				_, file, no, ok := runtime.Caller(1)
				if ok {
					fmt.Printf("CollectTxs-1, called from %s#%d\n", file, no)
				}

				return appGenTxs, persistentPeers, fmt.Errorf("account %s balance not in genesis state: %+v", valAccAddr, balancesMap)
			}

			_, valOk := balancesMap[sdk.AccAddress(valAddr).String()]
			if !valOk {
				_, file, no, ok := runtime.Caller(1)
				if ok {
					fmt.Printf("CollectTxs-2, called from %s#%d - %s\n", file, no, sdk.AccAddress(msg.ValidatorAddress).String())
				}
				return appGenTxs, persistentPeers, fmt.Errorf("account %s balance not in genesis state: %+v", valAddr, balancesMap)
			}

			if delBal.GetCoins().AmountOf(msg.Amount.Denom).LT(msg.Amount.Amount) {
				return appGenTxs, persistentPeers, fmt.Errorf(
					"insufficient fund for delegation %v: %v < %v",
					delBal.GetAddress(), delBal.GetCoins().AmountOf(msg.Amount.Denom), msg.Amount.Amount,
				)
			}
		}
	}

	sort.Strings(addressesIPs)
	persistentPeers = strings.Join(addressesIPs, ",")

	return appGenTxs, persistentPeers, nil
}
