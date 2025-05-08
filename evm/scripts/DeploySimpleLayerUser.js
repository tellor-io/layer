require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();

// npx hardhat run scripts/DeploySimpleLayerUser.sol --network sepolia

// update these variables
var dataBridgeAddress = " "
var queryId = " "
var PK = process.env.TESTNET_PK
var NODE_URL = process.env.NODE_URL_SEPOLIA_TESTNET

async function deploySimpleLayerUser(_pk, _nodeURL) {
    // var net = hre.network.name
    await run("compile")

    //Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider);
    
    ////////  Deploy SimpleLayerUser contract  ////////////////////////
    console.log("deploying SimpleLayerUser")
    const SimpleLayerUser = await ethers.getContractFactory("contracts/testing/SimpleLayerUser.sol:SimpleLayerUser", wallet);
    const simpleLayerUser= await SimpleLayerUser.deploy(dataBridgeAddress, queryId);
    await simpleLayerUser.deployed();
    console.log("SimpleLayerUser deployed to:", simpleLayerUser.address);
  };

  deploySimpleLayerUser(PK, NODE_URL)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
