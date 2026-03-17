require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();
const h = require("../test/helpers/evmHelpers");
require("@nomicfoundation/hardhat-verify");

// npx hardhat run scripts/DeployTellorPlayground.js --network sepolia

var PK = process.env.TESTNET_PK
var NODE_URL = process.env.NODE_URL_SEPOLIA_TESTNET

async function deployTellorPlayground(_pk, _nodeURL) {
    console.log("deploy TellorPlayground")
    // var net = hre.network.name
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
    console.log("TellorPlayground deployed to:", tellorPlayground.address);

    ////////  Verify the contract  ////////////////////////

    // wait for transaction receipt (5 confirmations)
    console.log("Waiting for 5 confirmations")
    await tellorPlayground.deployTransaction.wait(5);
    console.log("5 confirmations received");

    console.log("Verifying the contract")
    try {
        await hre.run("verify:verify", {
            address: tellorPlayground.address,
            constructorArguments: []
        });
        console.log("Contract verified");
    } catch (error) {
        console.error("Error verifying contract:", error);
    }
  };

  deployTellorPlayground(PK, NODE_URL)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
