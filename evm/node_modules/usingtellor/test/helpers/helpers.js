const web3 = require('web3');
const { ethers } = require("hardhat");
const BN = web3.utils.BN;

const hash = web3.utils.keccak256;
var assert = require('assert');

advanceTimeAndBlock = async (time) => {
    await advanceTime(time);
    await advanceBlock();
    console.log("Time Travelling...");
    return Promise.resolve(web3.eth.getBlock("latest"));
  };

const takeFifteen = async () => {
await advanceTime(60 * 18);
};


advanceTime = async (time) =>{
  await network.provider.send("evm_increaseTime", [time])
  await network.provider.send("evm_mine")
}
  advanceBlock = () => {
    return new Promise((resolve, reject) => {
      web3.currentProvider.send(
        {
          jsonrpc: "2.0",
          method: "evm_mine",
          id: new Date().getTime(),
        },
        (err, result) => {
          if (err) {
            return reject(err);
          }
          const newBlockHash = web3.eth.getBlock("latest").hash;

          return resolve(newBlockHash);
        }
      );
    });
  };

  async function expectThrow(promise) {
    try {
      await promise;
    } catch (error) {
      const invalidOpcode = error.message.search("invalid opcode") >= 0;
      const outOfGas = error.message.search("out of gas") >= 0;
      const revert = error.message.search("revert") >= 0;
      const overflow = error.message.search("overflow") >= 0;
      assert(
        invalidOpcode || outOfGas || revert || overflow,
        "Expected throw, got '" + error + "' instead"
      );
      return;
    }
    assert.fail("Expected throw not received");
  }

  function to18(n) {
    return ethers.BigNumber.from(n).mul(ethers.BigNumber.from(10).pow(18))
  }

  function tob32(n){
    return ethers.utils.formatBytes32String(n)
  }

  function uintTob32(n){
    let vars = web3.utils.toHex(n)
    vars = vars.slice(2)
    while(vars.length < 64){
      vars = "0" + vars
    }
    vars = "0x" + vars
    return vars
  }

  function bytes(n){
    return web3.utils.toHex(n)
  }

  function getBlock(){
    return ethers.provider.getBlock()
  }

module.exports = {
  stakeAmount: new BN(web3.utils.toWei("500", "ether")),
  timeTarget: 240,
  hash,
  zeroAddress:"0x0000000000000000000000000000000000000000",
  to18,
  uintTob32,
  tob32,
  bytes,
  getBlock,
  maxUint256: new BN("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
  advanceTime,
  advanceBlock,
  advanceTimeAndBlock,
  takeFifteen,
  expectThrow,
};
