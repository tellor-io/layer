require("hardhat-gas-reporter");
// require('hardhat-contract-sizer');
require("@nomiclabs/hardhat-ethers");
//require("@nomiclabs/hardhat-etherscan");
require("@nomicfoundation/hardhat-verify");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();
const hre = require("hardhat"); 
const web3 = require('web3');
const h = require("../test/helpers/evmHelpers");
const readline = require('readline').createInterface({
    input: process.stdin,
    output: process.stdout
});

//npx hardhat run scripts/deploy.js --network sepolia

var _tellorMaster = "0x80fc34a2f9FfE86F41580F47368289C402DEc660"
var _tellorFlex = "0xB19584Be015c04cf6CFBF6370Fe94a58b7A38830"
var _tokenBridge = "0x5acb5977f35b1A91C4fE0F4386eB669E046776F2"

const abiCoder = ethers.utils.defaultAbiCoder;
const oracleAddrQueryDataArgs = abiCoder.encode(["bytes"], ["0x"])
const oracleAddrQueryData = abiCoder.encode(["string", "bytes"], ["TellorOracleAddress", oracleAddrQueryDataArgs])
const oracleAddrQueryId = ethers.utils.keccak256(oracleAddrQueryData)

const _ORACLE_CONTRACT_HASH = ethers.utils.solidityKeccak256(["string"], ["_ORACLE_CONTRACT"])
const _PROPOSED_ORACLE_HASH = ethers.utils.solidityKeccak256(["string"], ["_PROPOSED_ORACLE"])
const _TIME_PROPOSED_UPDATED_HASH = ethers.utils.solidityKeccak256(["string"], ["_TIME_PROPOSED_UPDATED"])

const askQuestion = (query) => new Promise((resolve) => {
    readline.question(query, (answer) => {
        resolve(answer);
        readline.close();
    });
});

async function updateOracleMaster(_pk, _nodeURL) {
  
    console.log("\nUPDATING ORACLE MASTER...")
    console.log("Network: ", hre.network.name)
    console.log("TellorMaster: ", _tellorMaster)
    console.log("TellorFlex: ", _tellorFlex)
    console.log("TokenBridge: ", _tokenBridge)

    var net = hre.network.name

    await run("compile")

    //Connect to the network
    let privateKey = _pk;
    var provider = new ethers.providers.JsonRpcProvider(_nodeURL)
    let wallet = new ethers.Wallet(privateKey, provider);
    
    ////////  Connect to tellormaster and tellorflex  ////////////////////////
    const tellorMaster = await ethers.getContractAt("contracts/tellor360/Tellor360.sol:Tellor360", _tellorMaster, wallet);
    const tellorFlex= await ethers.getContractAt("contracts/interfaces/ITellorFlex.sol:ITellorFlex", _tellorFlex, wallet);


    // checklist for updating oracle address
    // 1. submit new oracle address (token bridge) to tellor flex
    // 2. wait 12 hours
    // 3. call updateOracleAddress() on tellor master
    // 4. wait 7 days
    // 5. call updateOracleAddress() on tellor master again

    const timestamp = Math.floor(Date.now() / 1000);
    const lastReportedOracleAddrReport = await tellorFlex.getDataBefore(oracleAddrQueryId, timestamp)
    const lastReportedOracleAddrDecoded = abiCoder.decode(["address"], lastReportedOracleAddrReport._value)
    const latestOracleAddrReportTimestamp = lastReportedOracleAddrReport._timestampRetrieved
    const latestOracleAddrReportDate = new Date(latestOracleAddrReportTimestamp * 1000)

    console.log("\nLAST REPORTED ORACLE ADDRESS")
    console.log("Last reported oracle address: ", lastReportedOracleAddrDecoded[0])
    console.log("Last reported oracle address timestamp: ", latestOracleAddrReportTimestamp.toString())
    console.log("Last reported oracle address date: ", latestOracleAddrReportDate)

    const currentOracleContractAddress = await tellorMaster.addresses(_ORACLE_CONTRACT_HASH)
    const proposedOracleContractAddress = await tellorMaster.addresses(_PROPOSED_ORACLE_HASH)
    const timeProposedUpdated = await tellorMaster.uints(_TIME_PROPOSED_UPDATED_HASH)
    const timeProposedUpdatedDate = new Date(timeProposedUpdated.toString() * 1000)

    console.log("\nCURRENT TELLOR MASTER VARS")
    console.log("Current _ORACLE_CONTRACT address: ", currentOracleContractAddress)
    console.log("Current _PROPOSED_ORACLE address: ", proposedOracleContractAddress)
    console.log("Current _TIME_PROPOSED_UPDATED: ", timeProposedUpdated.toString())
    console.log("Current _TIME_PROPOSED_UPDATED date: ", timeProposedUpdatedDate)

    // checklist
    check1 = lastReportedOracleAddrDecoded[0] == _tokenBridge
    check2 = check1 && timestamp - latestOracleAddrReportTimestamp > 43200
    check3 = proposedOracleContractAddress == _tokenBridge
    check4 = check3 && timestamp - timeProposedUpdated > 86400 * 7
    check5 = check4 && currentOracleContractAddress == _tokenBridge

    console.log("\nCHECKLIST")
    console.log("%s 1. Reported token bridge address to tellor flex", check1 ? "✅" : "❌")
    console.log("%s 2. Wait 12 hours", check2 ? "✅" : "❌")
    console.log("%s 3. Proposed oracle address is token bridge", check3 ? "✅" : "❌")
    console.log("%s 4. Wait 7 days", check4 ? "✅" : "❌")
    console.log("%s 5. Current oracle address is token bridge", check5 ? "✅" : "❌")
    
    if (!check1) {
        const answer = await askQuestion("Submit new oracle address to tellor flex? (Y/n) ");
        if (answer.toLowerCase() === "Y") {
            console.log("Submitting new oracle address to tellor flex...");
            tokenBridgeAddrEncoded = abiCoder.encode(["address"], [_tokenBridge]);
            const tx = await tellorFlex.submitValue(oracleAddrQueryId, tokenBridgeAddrEncoded, 0, oracleAddrQueryData);
            console.log("Transaction hash: ", tx.hash);
        }
    }
  };

  updateOracleMaster(process.env.TESTNET_PK, process.env.NODE_URL_SEPOLIA_TESTNET)
    .then(() => process.exit(0))
    .catch(error => {
	  console.error(error);
	  process.exit(1);
  });
