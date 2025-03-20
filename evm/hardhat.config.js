//require("@nomicfoundation/hardhat-toolbox");
//require("hardhat-gas-reporter");
require("dotenv").config();
require("@nomiclabs/hardhat-ethers");
// require("hardhat-gas-reporter");

// require("@nomiclabs/hardhat-web3");

module.exports = {
  solidity: {
    compilers: [
      {
        version: "0.8.19",
        settings: {
          optimizer: {
            enabled: true,
            runs: 300,
          },
        },
      },
      {
        version: "0.8.3",
        settings: {
          optimizer: {
            enabled: true,
            runs: 300,
          },
        },
      },
      {
        version: "0.7.4",
        settings: {
          optimizer: {
            enabled: true,
            runs: 300,
          },
        },
      },
      {
        version: "0.4.24",
        settings: {
          optimizer: {
            enabled: true,
            runs: 200,
          },
        },
      },
      {
        version: "0.6.11",
        settings: {
          optimizer: {
            enabled: true,
            runs: 100,
          },
        },
      },
    ],
  },
  networks: {
    hardhat: {
      accounts: {
        mnemonic:
          "nick lucian brenda kevin sam fiscal patch fly damp ocean produce wish",
        count: 40,
      },
      forking: {
        url: process.env.NODE_URL,
        blockNumber: 22075120
      },
      allowUnlimitedContractSize: true
    } ,
    sepolia: {
      url: `${process.env.NODE_URL_SEPOLIA}`,
      seeds: [process.env.TESTNET_PK],
      gas: 9000000 ,
      gasPrice: 5000000000
    } ,
    mainnet_testnet: {
      url: `${process.env.NODE_URL_MAINNET_TESTNET}`,
      seeds: [process.env.TESTNET_PK],
      gas: 8000000 ,
      gasPrice: 1000000000
    },
  },

  etherscan: {
    apiKey: process.env.ETHERSCAN
  },

  //etherscan: {
  //  apiKey: {
    // Your API key for Etherscan
    // Obtain one at https://etherscan.io/
    //sepolia: process.env.ETHERSCAN
    //mainnet: process.env.ETHERSCAN
 //}
//},



};


extendEnvironment((hre) => {
  const Web3 = require("web3");
  hre.Web3 = Web3;

  // hre.network.provider is an EIP1193-compatible provider.
  hre.web3 = new Web3(hre.network.provider);
});




