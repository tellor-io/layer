require("hardhat-gas-reporter");
require('hardhat-contract-sizer');
require("solidity-coverage");
require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-etherscan");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();
const web3 = require('web3');

//const dotenv = require('dotenv').config()
//npx hardhat run scripts/01_DeployTellorFlexwithtestStakingToken.js --network rinkeby

var stake_amt = web3.utils.toWei("10");
var rep_lock = 43200; // 12 hours

var governanceAddress = '0x20bEC8F31dea6C13A016DC7fCBdF74f61DC8Ec2c'


async function deployTellorFlex(_network, _pk, _nodeURL,  govAdd, stakeAmount, reporterLock) {
    console.log("deploy tellorFlex")
    await run("compile")

    var net = _network

    ///////////////Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider)

    //////////Deploy test token
    console.log("Starting deployment for staking token contract...")
    const stakingToken = await ethers.getContractFactory("contracts/testing/StakingToken.sol:StakingToken", wallet)
    const sTokenwithsigner = await stakingToken.connect(wallet)
    const st = await sTokenwithsigner.deploy()
    await st.deployed();

    if (net == "mainnet") {
        console.log("stakingToken contract deployed to:", "https://etherscan.io/address/" + st.address);
        console.log("    stakingToken transaction hash:", "https://etherscan.io/tx/" + st.deployTransaction.hash);
    } else if (net == "rinkeby") {
        console.log("stakingToken contract deployed to:", "https://rinkeby.etherscan.io/address/" + st.address);
        console.log("    stakingToken transaction hash:", "https://rinkeby.etherscan.io/tx/" + st.deployTransaction.hash);
    } else if (net == "harmony_testnet") {
        console.log("stakingToken contract deployed to:", "https://explorer.pops.one/address/" + st.address);
        console.log("    stakingToken transaction hash:", "https://explorer.pops.one/tx/" + st.deployTransaction.hash);
    } else if (net == "harmony_mainnet") {
        console.log("stakingToken contract deployed to:", "https://explorer.harmony.one/address/" + st.address);
        console.log("    stakingToken transaction hash:", "https://explorer.harmony.one/tx/" + st.deployTransaction.hash);
    } else {
        console.log("Please add network explorer details")
    }

    // Wait for few confirmed transactions.
    // Otherwise the etherscan api doesn't find the deployed contract.
    console.log('waiting for stakingToken tx confirmation...');
    await st.deployTransaction.wait(7)

    // console.log('submitting stakingToken contract for verification...');

    // await run("verify:verify",
    //     {
    //         address: st.address
             
    //     },
    // )

    // console.log("TellorFlex contract verified")

    /////////// Deploy Tellor flex
    console.log("deploy tellor flex")


    /////////////TellorFlex
    console.log("Starting deployment for TellorFlex contract...")
    const tellorF = await ethers.getContractFactory("contracts/TellorFlex.sol:TellorFlex", wallet)
    const tellorFwithsigner = await tellorF.connect(wallet)
    const tellor = await tellorFwithsigner.deploy(st.address, govAdd, stakeAmount, reporterLock)
    await tellor.deployed();

    if (net == "mainnet") {
        console.log("TellorFlex contract deployed to:", "https://etherscan.io/address/" + tellor.address);
        console.log("    TellorFlex transaction hash:", "https://etherscan.io/tx/" + tellor.deployTransaction.hash);
    } else if (net == "rinkeby") {
        console.log("TellorFlex contract deployed to:", "https://rinkeby.etherscan.io/address/" + tellor.address);
        console.log("    TellorFlex transaction hash:", "https://rinkeby.etherscan.io/tx/" + tellor.deployTransaction.hash);
    } else if (net == "harmony_testnet") {
        console.log("stakingToken contract deployed to:", "https://explorer.pops.one/address/" + st.address);
        console.log("    stakingToken transaction hash:", "https://explorer.pops.one/tx/" + st.deployTransaction.hash);
    } else if (net == "harmony_mainnet") {
        console.log("stakingToken contract deployed to:", "https://explorer.harmony.one/address/" + st.address);
        console.log("    stakingToken transaction hash:", "https://explorer.harmony.one/tx/" + st.deployTransaction.hash);
    } else {
        console.log("Please add network explorer details")
    }

    // Wait for few confirmed transactions.
    // Otherwise the etherscan api doesn't find the deployed contract.
    console.log('waiting for TellorFlex tx confirmation...');
    await tellor.deployTransaction.wait(7)

    // console.log('submitting TellorFlex contract for verification...');

    // await run("verify:verify",
    //     {
    //         address: tellor.address,
    //         constructorArguments: [st.address, govAdd, stakeAmount, reporterLock] 
    //     },
    // )

    // console.log("TellorFlex contract verified")

}


deployTellorFlex("harmony_testnet", process.env.TESTNET_PK, process.env.NODE_URL_HARMONY_TESTNET,governanceAddress,stake_amt,rep_lock)
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error);
        process.exit(1);
    });
// deployTellorFlex("harmony_mainnet", process.env.MAINNET_PK, process.env.NODE_URL_HARMONY_MAINNET,governanceAddress,stake_amt,rep_lock)
//     .then(() => process.exit(0))
//     .catch(error => {
//         console.error(error);
//         process.exit(1);
//     });

