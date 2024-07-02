//require("@nomicfoundation/hardhat-toolbox");
//require("hardhat-gas-reporter");
require("dotenv").config();
require("@nomiclabs/hardhat-ethers");
// require("@nomiclabs/hardhat-web3");

module.exports = {
  solidity: {
    compilers: [
      {
        version: "0.8.22",
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
      // accounts: {
      //   mnemonic:
      //     "nick lucian brenda kevin sam fiscal patch fly damp ocean produce wish",
      //   count: 40,
      // },
      forking: {
        url: process.env.NODE_URL,
        blockNumber: 19891853
      },
      // allowUnlimitedContractSize: true
    } //,
  },
};

extendEnvironment((hre) => {
  const Web3 = require("web3");
  hre.Web3 = Web3;

  // hre.network.provider is an EIP1193-compatible provider.
  hre.web3 = new Web3(hre.network.provider);
});