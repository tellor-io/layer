
require("hardhat-gas-reporter");
require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();
const hre = require("hardhat"); 
const h = require("../test/helpers/evmHelpers");

//npx hardhat run scripts/deploy.js --network sepolia

var _guardian_address = ""
var _token = ""
var _tellor_flex = ""
var _layer_chain_id = "layertest-4"


_valset_domain_separator = h.getDomainSeparator(_layer_chain_id)

async function deployForMainnet(_pk, _nodeURL) {
    console.log("deploying bridge with token bridge")
    console.log("guardian address", _guardian_address)
    console.log("token address", _token)
    console.log("tellor flex address", _tellor_flex)
    console.log("layer chain id", _layer_chain_id)
    console.log("valset domain separator", _valset_domain_separator)
  
    var net = hre.network.name

    await run("compile")

    //Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider);
    
    ////////  Deploy TellorDataBridge contract  ////////////////////////
    console.log("deploy TellorDataBridge")
    const TellorDataBridge = await ethers.getContractFactory("contracts/bridge/TellorDataBridge.sol:TellorDataBridge", wallet);
    var TellorWithSigner = await TellorDataBridge.connect(wallet);
    const tellorDataBridge= await TellorWithSigner.deploy(_guardian_address,_valset_domain_separator);
    await tellorDataBridge.deployed();


    ////////  Deploy token bridge contract  ////////////////////////
    console.log("deploy token bridge")
    const TokenBridge = await ethers.getContractFactory("contracts/token-bridge/TokenBridge.sol:TokenBridge", wallet);
    var tbWithSigner = await TokenBridge.connect(wallet);
    /// @param _token address of tellor token for bridging
    /// @param _dataBridge address of tellor data bridge
    /// @param _tellorFlex address of oracle(tellorFlex) on chain
    const tokenBridge= await tbWithSigner.deploy(_token,tellorDataBridge.address,_tellor_flex);
    await tokenBridge.deployed();

    /////////  Print addresses   ///////////////////////////

    if (net == "mainnet"){
            console.log("Tellor token bridge deployed to:", tokenBridge.address);
            console.log("Tellor token bridge deployed to:", "https://etherscan.io/address/" + tokenBridge.address);
            console.log("Tellor data bridge deployed to:", tellorDataBridge.address);
            console.log("Tellor data bridge deployed to:", "https://etherscan.io/address/" + tellorDataBridge.address);
        
        }  else if (net == "sepolia"){ 
            console.log("Tellor token bridge deployed to:", tokenBridge.address);
            console.log("Tellor token bridge deployed to:", "https://sepolia.etherscan.io/address/" + tokenBridge.address);
            console.log("Tellor data bridge deployed to:", tellorDataBridge.address);
            console.log("Tellor data bridge deployed to:", "https://sepolia.etherscan.io/address/" + tellorDataBridge.address);

        }  else {
        console.log("Please add network explorer details")
    }
  };

  deployForMainnet(process.env.TESTNET_PK, process.env.NODE_URL_SEPOLIA_TESTNET)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
