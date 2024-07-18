
require("hardhat-gas-reporter");
require('hardhat-contract-sizer');
require("@nomiclabs/hardhat-ethers");
//require("@nomiclabs/hardhat-etherscan");
require("@nomicfoundation/hardhat-verify");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();
const hre = require("hardhat"); 
const web3 = require('web3');

//npx hardhat run scripts/deploy.js --network sepolia

var _thegardianaddress = " "
var _token = " "
var _tellorFlex = " "


async function deployForMainnet(_pk, _nodeURL) {
  
    var net = hre.network.name

    await run("compile")

    //Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider);
    
    ////////  Deploy Blobstream contract  ////////////////////////
    console.log("deploy BlobstreamO bridge")
    const BlobstreamO = await ethers.getContractFactory("contracts/BlobstreamO.sol:BlobstreamO", wallet);
    var BlobWithSigner = await BlobstreamO.connect(wallet);
    const blobstreamO= await BlobWithSigner.deploy(_thegardianaddress);
    await blobstreamO.deployed();


        ////////  Deploy token bridge contract  ////////////////////////
        console.log("deploy token bridge")
        const TokenBridge = await ethers.getContractFactory("contracts/TokenBridge.sol:TokenBridge", wallet);
        var tbWithSigner = await TokenBridge.connect(wallet);
        /// @param _token address of tellor token for bridging
        /// @param _blobstream address of BlobstreamO data bridge
        /// @param _tellorFlex address of oracle(tellorFlex) on chain
        const tokenBridge= await tbWithSigner.deploy(_token,blobstreamO.address,_tellorFlex);
        await tokenBridge.deployed();

    /////////  Print addresses   ///////////////////////////

    if (net == "mainnet"){
            console.log("Tellor token bridge deployed to:", tokenBridge.address);
            console.log("Tellor token bridge deployed to:", "https://etherscan.io/address/" + tokenBridge.address);
            console.log("Tellor BlobstreamO bridge deployed to:", blobstreamO.address);
            console.log("Tellor blobstreamO bridge deployed to:", "https://etherscan.io/address/" + blobstreamO.address);
        
        }  else if (net == "sepolia"){ 
            console.log("Tellor token bridge deployed to:", tokenBridge.address);
            console.log("Tellor token bridge deployed to:", "https://sepolia.etherscan.io/address/" + tokenBridge.address);
            console.log("Tellor BlobstreamO bridge deployed to:", blobstreamO.address);
            console.log("Tellor blobstreamO bridge deployed to:", "https://sepolia.etherscan.io/address/" + blobstreamO.address);

        }  else {
        console.log("Please add network explorer details")
    }


    // Wait for few confirmed transactions.
    // Otherwise the etherscan api doesn't find the deployed contract.
    console.log('waiting for tx confirmation...');
    await tokenBridge.deployTransaction.wait(3)

    console.log('submitting contract for verification...');
    try {
    await run("verify:verify", {
      address: tokenBridge,
      constructor: [_token,blobstreamO.address,_tellorFlex]
    },
    )
    await run("verify:verify",
    {
        address: blobstreamO.address,
        constructorArguments: [flex.address, teamMultisigAddress]
    },
)

    console.log("Contract verified")
     } catch (e) {
    console.log(e)
    }

  };

  deployForMainnet(process.env.TESTNET_PK, process.env.NODE_URL_SEPOLIA_TESTNET)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
