require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();

//npx hardhat run scripts/DeploySimpleLayerUser.sol --network sepolia

var blobstreamOaddress = "0xFa600432D5f7C252D399bB1A77546a447eeCaF67"
var queryId = "0x83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"

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
    const simpleLayerUser= await SimpleLayerUser.deploy(blobstreamOaddress, queryId);
    await simpleLayerUser.deployed();
    console.log("SimpleLayerUser deployed to:", simpleLayerUser.address);
  };

  deploySimpleLayerUser(process.env.TESTNET_PK, process.env.NODE_URL_SEPOLIA_TESTNET)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
