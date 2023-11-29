
require("@nomiclabs/hardhat-waffle");
require("hardhat-gas-reporter");
require("solidity-coverage");
require("hardhat-gas-reporter");
require('hardhat-contract-sizer');
require("@nomiclabs/hardhat-ethers");
require("@nomiclabs/hardhat-etherscan");
require("dotenv").config();
/**
 * @type import('hardhat/config').HardhatUserConfig
 */

 module.exports = {
  solidity: {
    compilers: [
      {
        version: "0.8.3",
        settings: {
          optimizer: {
            enabled: true,
            runs: 300
          }
        }
      },
      {
        version: "0.7.4",
        settings: {
          optimizer: {
            enabled: true,
            runs: 300
          }
        }
      }
    ]
  },
  networks: {
    hardhat: {
      // hardfork: process.env.CODE_COVERAGE ? "berlin" : "london",
      initialBaseFeePerGas: 0,
      accounts: {
        mnemonic:
          "nick lucian brenda kevin sam fiscal patch fly damp ocean produce wish",
        count: 40,
      },
      allowUnlimitedContractSize: true
    },
    // rinkeby: {
    //      url: `${process.env.NODE_URL_RINKEBY}`,
    //      seeds: [process.env.PRIVATE_KEY],
    //      gas: 10000000 ,
    //      gasPrice: 40000000000
    // } ,
    //    mainnet: {
    //      url: `${process.env.NODE_URL_MAINNET}`,
    //      seeds: [process.env.PRIVATE_KEY],
    //      gas: 3000000 ,
    //      gasPrice: 300000000000
    //    }
    // polygon_testnet: {
    //     url: `${process.env.NODE_URL_MUMBAI}`,
    //     accounts: [process.env.TESTNET_PK],
    //     gas: 10000000 ,
    //     gasPrice: 50000000000
    //   }
      // harmony_testnet: {
      //   url: `${process.env.NODE_URL_HARMONY_TESTNET}`,
      //   accounts: [process.env.TESTNET_PK],
      //   gas: 10000000 ,
      //   gasPrice: 50000000000
      // }
      // harmony_mainnet: {
      //   url: `${process.env.NODE_URL_HARMONY_MAINNET}`,
      //   accounts: [process.env.MAINNET_PK],
      //   gas: 10000000 ,
      //   gasPrice: 50000000000
      // }
     harmony_mainnet: {
        url: "https://api.s0.t.hmny.io/",
        seeds: [process.env.PRIVATE_KEY],
        gas: 10000000 ,
        gasPrice: 50000000000
    }
    , arbitrum_testnet: {
      url: `${process.env.NODE_URL_ARBITRUM_TESTNET}`,
      seeds: [process.env.PRIVATE_KEY],
      gas: 10000000 ,
      gasPrice: 50000000000
    }
  },
  etherscan: {
    // Your API key for Etherscan
    // Obtain one at https://etherscan.io/
    apiKey: process.env.POLYGONSCAN
  },

  contractSizer: {
    alphaSort: true,
    runOnCompile: true,
    disambiguatePaths: false,
  },

  mocha: {
    grep: "@skip-on-coverage", // Find everything with this tag
    invert: true               // Run the grep's inverse set.
  }

}
