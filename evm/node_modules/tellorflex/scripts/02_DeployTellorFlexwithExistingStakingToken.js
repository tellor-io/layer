require("hardhat-gas-reporter");
require('hardhat-contract-sizer');
require("solidity-coverage");
require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-etherscan");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();
const web3 = require('web3');

//const dotenv = require('dotenv').config()
//npx hardhat run scripts/02_DeployTellorFlexwithExistingStakingToken.js --network rinkeby
// npx hardhat run scripts/02_DeployTellorFlexwithExistingStakingToken.js --network harmony_mainnet

var stake_amt = web3.utils.toWei("10");
var rep_lock = 43200; // 12 hours
// var stakerTokenAdd= '0x002e861910d7f87baa832a22ac436f25fb66fa24'
//var stakerTokenAdd= '0x9ff6799d07cbc824ec16cf5fe7bdc6798849e9cc' // harmony testnet TRB address
//var governanceAddress = '0x0Fe623d889Ad1c599E5fF3076A57D1D4F2448CDe'

//harmony mainnet      
//var stakerTokenAdd = '0xd4b28ecb7b765c89f1e67de3359d09a3520f794e'
//var governanceAddress = '0x0d7effefdb084dfeb1621348c8c70cc4e871eba4'

//arbitrum testnet     
var stakerTokenAdd = '0xa800d52F5E72b918f8E09D5439beBEBf09ea2f74'
var governanceAddress = '0x0d7effefdb084dfeb1621348c8c70cc4e871eba4'


async function deployTellorFlex(_network, _pk, _nodeURL, stakerToken, govAdd, stakeAmount, reporterLock) {
    console.log("deploy tellorFlex")
    await run("compile")

    var net = _network

    ///////////////Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider)

    /////////// Deploy Tellor flex
    console.log("deploy tellor flex")

    /////////////TellorFlex
    console.log("Starting deployment for TellorFlex contract...")
    const tellorF = await ethers.getContractFactory("contracts/TellorFlex.sol:TellorFlex", wallet)
    const tellorFwithsigner = await tellorF.connect(wallet)
    const tellor = await tellorFwithsigner.deploy(stakerToken, govAdd, stakeAmount, reporterLock)
    await tellor.deployed();

    if (net == "mainnet"){
        console.log("Tellor contract deployed to:", "https://etherscan.io/address/" + tellor.address);
        console.log("    transaction hash:", "https://etherscan.io/tx/" + tellor.deployTransaction.hash);
    } else if (net == "rinkeby") {
        console.log("Tellor contract deployed to:", "https://rinkeby.etherscan.io/address/" + tellor.address);
        console.log("    transaction hash:", "https://rinkeby.etherscan.io/tx/" + tellor.deployTransaction.hash);
    } else if (net == "harmony_testnet") {
        console.log("Tellor contract deployed to:", "https://explorer.pops.one/address/" + tellor.address);
        console.log("    transaction hash:", "https://explorer.pops.one/tx/" + tellor.deployTransaction.hash);
    } else if (net == "harmony_mainnet") {
        console.log("Tellor contract deployed to:", "https://explorer.harmony.one/address/" + tellor.address);
        console.log("    transaction hash:", "https://explorer.harmony.one/tx/" + tellor.deployTransaction.hash);
    } else if (net == "bsc_testnet") {
        console.log("Tellor contract deployed to:", "https://testnet.bscscan.com/address/" + tellor.address);
        console.log("    transaction hash:", "https://testnet.bscscan.com/tx/" + tellor.deployTransaction.hash);
    } else if (net == "bsc") {
        console.log("Tellor contract deployed to:", "https://bscscan.com/address/" + tellor.address);
        console.log("    transaction hash:", "https://bscscan.com/tx/" + tellor.deployTransaction.hash);
    } else if (net == "polygon") {
        console.log("Tellor contract deployed to:", "https://explorer-mainnet.maticvigil.com/" + tellor.address);
        console.log("    transaction hash:", "https://explorer-mainnet.maticvigil.com/tx/" + tellor.deployTransaction.hash);
    } else if (net == "polygon_testnet") {
        console.log("Tellor contract deployed to:", "https://explorer-mumbai.maticvigil.com/" + tellor.address);
        console.log("    transaction hash:", "https://explorer-mumbai.maticvigil.com/tx/" + tellor.deployTransaction.hash);
    } else if (net == "arbitrum_testnet"){
        console.log("tellor contract deployed to:","https://rinkeby-explorer.arbitrum.io/#/"+ tellor.address)
        console.log("    transaction hash:", "https://rinkeby-explorer.arbitrum.io/#/tx/" + tellor.deployTransaction.hash);
    }  else if (net == "xdaiSokol"){ //https://blockscout.com/poa/xdai/address/
      console.log("tellor contract deployed to:","https://blockscout.com/poa/sokol/address/"+ tellor.address)
      console.log("    transaction hash:", "https://blockscout.com/poa/sokol/tx/" + tellor.deployTransaction.hash);
    } else if (net == "xdai"){ //https://blockscout.com/poa/xdai/address/
      console.log("tellor contract deployed to:","https://blockscout.com/xdai/mainnet/address/"+ tellor.address)
      console.log("    transaction hash:", "https://blockscout.com/xdai/mainnet/tx/" + tellor.deployTransaction.hash);
    } else {
        console.log("Please add network explorer details")
    }


    // Wait for few confirmed transactions.
    // Otherwise the etherscan api doesn't find the deployed contract.
    console.log('waiting for TellorFlex tx confirmation...');
    await tellor.deployTransaction.wait(7)

    console.log('submitting TellorFlex contract for verification...');

    await run("verify:verify",
        {
            address: tellor.address,
            constructorArguments: [stakerToken, govAdd, stakeAmount, reporterLock]
        },
    )

    console.log("TellorFlex contract verified")

}


// deployTellorFlex("harmony_testnet", process.env.TESTNET_PK, process.env.NODE_URL_HARMONY_TESTNET,stakerTokenAdd,governanceAddress,stake_amt,rep_lock)
//     .then(() => process.exit(0))
//     .catch(error => {
//         console.error(error);
//         process.exit(1);
//     });
// deployTellorFlex("harmony_mainnet", process.env.PRIVATE_KEY, process.env.NODE_URL_HARMONY_MAINNET,stakerTokenAdd,governanceAddress,stake_amt,rep_lock)
//     .then(() => process.exit(0))
//     .catch(error => {
//         console.error(error);
//         process.exit(1);
//     });

    deployTellorFlex("arbitrum_testnet", process.env.PRIVATE_KEY, process.env.NODE_URL_ARBITRUM_TESTNET,stakerTokenAdd,governanceAddress,stake_amt,rep_lock)
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error);
        process.exit(1);
    });
