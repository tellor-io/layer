require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();

// npx hardhat run scripts/DeployBlobstreamO.js --network sepolia

// update these variables
var guardianaddress = " "
var PK = process.env.TESTNET_PK
var NODE_URL = process.env.NODE_URL_SEPOLIA_TESTNET

async function deployBlobstreamO(_pk, _nodeURL) {
    // var net = hre.network.name
    await run("compile")

    //Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider);
    
    ////////  Deploy Blobstream contract  ////////////////////////
    console.log("deploy BlobstreamO bridge")
    const BlobstreamO = await ethers.getContractFactory("contracts/bridge/BlobstreamO.sol:BlobstreamO", wallet);
    const blobstreamO= await BlobstreamO.deploy(guardianaddress);
    await blobstreamO.deployed();
    console.log("BlobstreamO deployed to:", blobstreamO.address);
  };

  deployBlobstreamO(PK, NODE_URL)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
