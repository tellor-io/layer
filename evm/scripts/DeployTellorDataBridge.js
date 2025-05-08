require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();

// npx hardhat run scripts/DeployTellorDataBridge.js --network sepolia

// update these variables
var guardianaddress = " "
var PK = process.env.TESTNET_PK
var NODE_URL = process.env.NODE_URL_SEPOLIA_TESTNET

async function deployTellorDataBridge(_pk, _nodeURL) {
    // var net = hre.network.name
    await run("compile")

    //Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider);
    
    ////////  Deploy TellorDataBridge contract  ////////////////////////
    console.log("deploy TellorDataBridge")
    const TellorDataBridge = await ethers.getContractFactory("contracts/TellorDataBridge.sol:TellorDataBridge", wallet);
    const tellorDataBridge= await TellorDataBridge.deploy(guardianaddress);
    await tellorDataBridge.deployed();
    console.log("TellorDataBridge deployed to:", tellorDataBridge.address);
  };

  deployTellorDataBridge(PK, NODE_URL)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
