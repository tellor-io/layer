
require("hardhat-gas-reporter");
require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();
const hre = require("hardhat"); 
const h = require("../test/helpers/evmHelpers");

//npx hardhat run scripts/DeployFullTestBridgesV1.js --network sepolia

// This script deploys the TellorPlayground, TellorDataBridge, and TokenBridge v1 contracts.
// Then it initializes the token bridge and mints it an initial balance.

var _data_bridge_guardian_address = " "
var _layer_chain_id = " "

// tokenbridge v1 contract init 
var token_bridge_deposit_id = 0;
var token_bridge_withdraw_id = 0;
var token_bridge_initial_balance_trb = 10000

_valset_domain_separator = h.getDomainSeparator(_layer_chain_id)

async function deployForMainnet(_pk, _nodeURL) {
    console.log("deploying bridge with token bridge")
    console.log("guardian address", _data_bridge_guardian_address)
    console.log("layer chain id", _layer_chain_id)
    console.log("valset domain separator", _valset_domain_separator)
  
    var net = hre.network.name

    await run("compile")

    //Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider);

    ////////  Deploy TellorPlayground contract  ////////////////////////
    console.log("deploy TellorPlayground")
    const TellorPlayground = await ethers.getContractFactory("contracts/testing/TellorPlayground.sol:TellorPlayground", wallet);
    const tellorPlayground= await TellorPlayground.deploy();
    await tellorPlayground.deployed();
    
    ////////  Deploy TellorDataBridge contract  ////////////////////////
    console.log("deploy TellorDataBridge")
    const TellorDataBridge = await ethers.getContractFactory("contracts/bridge/TellorDataBridge.sol:TellorDataBridge", wallet);
    var TellorWithSigner = await TellorDataBridge.connect(wallet);
    const tellorDataBridge= await TellorWithSigner.deploy(_data_bridge_guardian_address,_valset_domain_separator);
    await tellorDataBridge.deployed();

    ////////  Deploy token bridge contract  ////////////////////////
    console.log("deploy token bridge")
    const TokenBridge = await ethers.getContractFactory("contracts/token-bridge/TokenBridge.sol:TokenBridge", wallet);
    var tbWithSigner = await TokenBridge.connect(wallet);
    /// @param _token address of tellor token for bridging
    /// @param _dataBridge address of tellor data bridge
    /// @param _tellorFlex address of oracle(tellorFlex) on chain
    const tokenBridge= await tbWithSigner.deploy(tellorPlayground.address,tellorDataBridge.address,tellorPlayground.address);
    await tokenBridge.deployed();

    /////////  Print addresses   ///////////////////////////

    if (net == "mainnet"){
            console.log("Tellor playground deployed to:", tellorPlayground.address);
            console.log("Tellor playground deployed to:", "https://etherscan.io/address/" + tellorPlayground.address);
            console.log("Tellor token bridge deployed to:", tokenBridge.address);
            console.log("Tellor token bridge deployed to:", "https://etherscan.io/address/" + tokenBridge.address);
            console.log("Tellor data bridge deployed to:", tellorDataBridge.address);
            console.log("Tellor data bridge deployed to:", "https://etherscan.io/address/" + tellorDataBridge.address);
        
        }  else if (net == "sepolia"){ 
            console.log("Tellor playground deployed to:", tellorPlayground.address);
            console.log("Tellor playground deployed to:", "https://sepolia.etherscan.io/address/" + tellorPlayground.address);
            console.log("Tellor token bridge deployed to:", tokenBridge.address);
            console.log("Tellor token bridge deployed to:", "https://sepolia.etherscan.io/address/" + tokenBridge.address);
            console.log("Tellor data bridge deployed to:", tellorDataBridge.address);
            console.log("Tellor data bridge deployed to:", "https://sepolia.etherscan.io/address/" + tellorDataBridge.address);

        }  else {
        console.log("Please add network explorer details")
    }

    ////////  Verify the contracts  ////////////////////////

    // wait for transaction receipt (5 confirmations)
    console.log("Waiting for 5 confirmations")
    await tokenBridge.deployTransaction.wait(5);
    console.log("5 confirmations received");

    console.log("Verifying the contracts")
    console.log("Verifying TellorPlayground contract")
    try {
        await hre.run("verify:verify", {
            address: tellorPlayground.address,
            constructorArguments: []
        });
        console.log("Contract verified");
    } catch (error) {
        console.error("Error verifying contract:", error);
    }
    console.log("Verifying TellorDataBridge contract")
    try {
        await hre.run("verify:verify", {
            address: tellorDataBridge.address,
            constructorArguments: [_data_bridge_guardian_address, _valset_domain_separator]
        });
        console.log("Contract verified");
    } catch (error) {
        console.error("Error verifying contract:", error);
    }
    console.log("Verifying TokenBridge contract")
    try {
        await hre.run("verify:verify", {
            address: tokenBridge.address,
            constructorArguments: [tellorPlayground.address, tellorDataBridge.address, tellorPlayground.address]
        });
        console.log("Contract verified");
    } catch (error) {
        console.error("Error verifying contract:", error);
    }

    ////////  Initialize the token bridge  ////////////////////////
    console.log("Initializing the token bridge")
    await tokenBridge.init(token_bridge_deposit_id, token_bridge_withdraw_id);
    console.log("Token bridge initialized");

    ////////  Mint tokens to the token bridge  ////////////////////////
    // faucet mints 1000 tokens at a time. Use faucet to mint to this wallet address until balance is reached, 
    // and then transfer the balance to the token bridge.
    console.log("Minting tokens to the token bridge")
    num_calls = Math.floor(token_bridge_initial_balance_trb / 1000) + 1;
    for (let i = 0; i < num_calls; i++) {
        await tellorPlayground.faucet(wallet.address);
    }
    console.log("Tokens minted to the wallet address");
    console.log("Transferring tokens to the token bridge")
    transfer_tx = await tellorPlayground.transfer(tokenBridge.address, ethers.utils.parseEther(token_bridge_initial_balance_trb.toString()));
    await transfer_tx.wait(2);
    console.log("Tokens transferred to the token bridge");
    console.log("Token bridge balance:", ethers.utils.formatEther(await tellorPlayground.balanceOf(tokenBridge.address)), "TRB");
  };

  deployForMainnet(process.env.TESTNET_PK, process.env.NODE_URL_SEPOLIA_TESTNET)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
