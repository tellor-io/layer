require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();

// npx hardhat run scripts/DeployUpdateOracleTestnet.sol --network sepolia

// update these variables
var tokenBridgeAddress = ""
var PK = process.env.TESTNET_PK
var NODE_URL = process.env.NODE_URL_SEPOLIA_TESTNET

async function deployUpdateOracleTestnet(_pk, _nodeURL) {
    // var net = hre.network.name
    await run("compile")

    //Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider);
    
    ////////  Deploy UpdateOracleTestnet contract  ////////////////////////
    console.log("deploying UpdateOracleTestnet")
    const UpdateOracleTestnet = await ethers.getContractFactory("contracts/testing/UpdateOracleTestnet.sol:UpdateOracleTestnet", wallet);
    const updateOracleTestnet= await UpdateOracleTestnet.deploy(tokenBridgeAddress);
    await updateOracleTestnet.deployed();
    console.log("UpdateOracleTestnet deployed to:", updateOracleTestnet.address);
  };

  deployUpdateOracleTestnet(PK, NODE_URL)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
