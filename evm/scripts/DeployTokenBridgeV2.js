
require("hardhat-gas-reporter");
require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();
const hre = require("hardhat"); 
const h = require("../test/helpers/evmHelpers");

//npx hardhat run scripts/DeployTokenBridgeV2.js --network sepolia

var _token = " "
var _tellor_flex = " "
var _data_bridge = " "

async function deployForMainnet(_pk, _nodeURL) {
    console.log("Deploying TokenBridgeV2")
    console.log("Token address:", _token)
    console.log("Tellor Flex address:", _tellor_flex)
    console.log("Data bridge address:", _data_bridge)

    await run("compile")

    //Connect to the network
    var net = hre.network.name
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider);
  
    ////////  Deploy token bridge contract  ////////////////////////
    console.log("Deploying TokenBridgeV2")
    const TokenBridge = await ethers.getContractFactory("contracts/token-bridge/TokenBridgeV2.sol:TokenBridgeV2", wallet);
    var tbWithSigner = await TokenBridge.connect(wallet);
    /// @param _token address of tellor token for bridging
    /// @param _dataBridge address of tellor data bridge
    /// @param _tellorFlex address of oracle(tellorFlex) on chain
    const tokenBridge= await tbWithSigner.deploy(_token,_data_bridge,_tellor_flex);
    await tokenBridge.deployed();
    
    /////////  Print addresses   ///////////////////////////

    if (net == "mainnet"){
            console.log("Tellor token bridge deployed to:", tokenBridge.address);
            console.log("https://etherscan.io/address/" + tokenBridge.address);
        
        }  else if (net == "sepolia"){ 
            console.log("Tellor token bridge deployed to:", tokenBridge.address);
            console.log("https://sepolia.etherscan.io/address/" + tokenBridge.address);

        }  else {
            console.log("TokenBridgeV2 deployed to:", tokenBridge.address);
            console.log("Please add network explorer details")
    }
  };

  deployForMainnet(process.env.TESTNET_PK, process.env.NODE_URL_SEPOLIA_TESTNET)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
