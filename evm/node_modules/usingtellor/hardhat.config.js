/**
 * @type import('hardhat/config').HardhatUserConfig
 */
require("@nomiclabs/hardhat-waffle");
require("@nomiclabs/hardhat-ethers");

task("balance", "Prints an account's balance").setAction(async () => {});

module.exports = {
  solidity: "0.8.3",
  networks: {
    hardhat: {
      // forking: {
      //   url: "https://eth-mainnet.alchemyapi.io/v2/7dW8KCqWwKa1vdaitq-SxmKfxWZ4yPG6",
      //   blockNumber: 14390000
      // },
    },
  },
};
