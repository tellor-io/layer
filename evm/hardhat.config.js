require("dotenv").config();
require("@nomiclabs/hardhat-ethers");

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
        url: process.env.NODE_URL_MAINNET,
        blockNumber: 22247348
      },
      allowUnlimitedContractSize: true
    } ,
    sepolia: {
      url: `${process.env.NODE_URL_SEPOLIA_TESTNET}`,
      accounts: [process.env.TESTNET_PK],
      gas: 9000000,
      gasPrice: 5000000000
    } ,
    mainnet: {
      url: `${process.env.NODE_URL_MAINNET}`,
      seeds: [process.env.MAINNET_PK],
      gas: 8000000 ,
      gasPrice: 1000000000
    },
  },
};


extendEnvironment((hre) => {
  const Web3 = require("web3");
  hre.Web3 = Web3;

  // hre.network.provider is an EIP1193-compatible provider.
  hre.web3 = new Web3(hre.network.provider);
});




