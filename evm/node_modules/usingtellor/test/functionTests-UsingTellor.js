const { expect, assert } = require("chai");
const { ethers } = require("hardhat");
const h = require("./helpers/helpers");
const web3 = require('web3');
let abiCoder = new ethers.utils.AbiCoder

const precision = BigInt(1e18);
const FAUCET_AMOUNT = BigInt(1000) * precision;
const ZERO_ADDRESS = "0x0000000000000000000000000000000000000000";

describe("UsingTellor Function Tests", function() {

	let bench
  let playground
  let mappingContract;
	let owner, addr0, addr1, addr2;

	beforeEach(async function () {
    
		const TellorPlayground = await ethers.getContractFactory("TellorPlayground");
		playground = await TellorPlayground.deploy();
    await playground.deployed();

    const BenchUsingTellor = await ethers.getContractFactory("BenchUsingTellor");
    bench = await BenchUsingTellor.deploy(playground.address);
    await bench.deployed();

    const MappingContract = await ethers.getContractFactory("MappingContractExample");
    mappingContract = await MappingContract.deploy();
    await mappingContract.deployed();

		[owner, addr1, addr2] = await ethers.getSigners();
	});

  it("retrieveData()", async function() {
    await playground.connect(addr1).submitValue(h.uintTob32(1),150,0,'0x')
    blocky1 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),160,1,'0x')
    blocky2 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),170,2,'0x')
    blocky3 = await h.getBlock()
    expect(await bench.retrieveData(h.uintTob32(1), blocky1.timestamp)).to.equal(h.bytes(150))
    expect(await bench.retrieveData(h.uintTob32(1), blocky2.timestamp)).to.equal(h.bytes(160))
    expect(await bench.retrieveData(h.uintTob32(1), blocky3.timestamp)).to.equal(h.bytes(170))
  })

  it("getNewValueCountbyQueryId", async function() {
    await playground.connect(addr1).submitValue(h.uintTob32(1),150,0,'0x')
    expect(await bench.getNewValueCountbyQueryId(h.uintTob32(1))).to.equal(1)
    await playground.connect(addr1).submitValue(h.uintTob32(1),160,1,'0x')
    expect(await bench.getNewValueCountbyQueryId(h.uintTob32(1))).to.equal(2)
    await playground.connect(addr1).submitValue(h.uintTob32(1),170,2,'0x')
    expect(await bench.getNewValueCountbyQueryId(h.uintTob32(1))).to.equal(3)
  })

  it("getTimestampbyQueryIdandIndex()", async function() {
    await playground.connect(addr1).submitValue(h.uintTob32(1),150,0,'0x')
    blocky1 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),160,1,'0x')
    blocky2 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),170,2,'0x')
    blocky3 = await h.getBlock()
    expect(await bench.getTimestampbyQueryIdandIndex(h.uintTob32(1), 0)).to.equal(blocky1.timestamp)
    expect(await bench.getTimestampbyQueryIdandIndex(h.uintTob32(1), 1)).to.equal(blocky2.timestamp)
    expect(await bench.getTimestampbyQueryIdandIndex(h.uintTob32(1), 2)).to.equal(blocky3.timestamp)
  })

  it("getIndexForDataBefore()", async function() {
    await playground.connect(addr1).submitValue(h.uintTob32(1),150,0,'0x')
    blocky1 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),160,1,'0x')
    blocky2 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),170,2,'0x')
    blocky3 = await h.getBlock()
    index = await bench.getIndexForDataBefore(h.uintTob32(1), blocky3.timestamp)
    expect(index[0])
    expect(index[1]).to.equal(1)
  })

  it("getDataBefore()", async function() {
    await playground.connect(addr1).submitValue(h.uintTob32(1),150,0,'0x')
    blocky1 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),160,1,'0x')
    blocky2 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),170,2,'0x')
    blocky3 = await h.getBlock()
    dataBefore = await bench.getDataBefore(h.uintTob32(1), blocky2.timestamp)
    expect(dataBefore[0]).to.equal(h.bytes(150))
    expect(dataBefore[1]).to.equal(blocky1.timestamp)
  })

	it("isInDispute()", async function() {
		await playground.connect(addr1).submitValue(h.uintTob32(1),150,0,'0x')
    blocky1 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),160,1,'0x')
    blocky2 = await h.getBlock()
		expect(await bench.isInDispute(h.uintTob32(1), blocky1.timestamp)).to.be.false;
		await playground.beginDispute(h.uintTob32(1), blocky1.timestamp)
		expect(await bench.isInDispute(h.uintTob32(1), blocky1.timestamp))
		await playground.beginDispute(h.uintTob32(1), blocky1.timestamp)
		expect(await bench.isInDispute(h.uintTob32(1), blocky1.timestamp))
		expect(await bench.isInDispute(h.uintTob32(1), blocky2.timestamp)).to.be.false;
		await playground.beginDispute(h.uintTob32(1), blocky2.timestamp)
		expect(await bench.isInDispute(h.uintTob32(1), blocky2.timestamp))
	})

  it("valueFor()", async function() {
    await bench.setIdMappingContract(mappingContract.address);
    oracleQueryDataPartial = abiCoder.encode(['string','string'], ['eth','usd'])
    oracleQueryData = abiCoder.encode(['string', 'bytes'], ['SpotPrice', oracleQueryDataPartial])
    oracleQueryId = ethers.utils.keccak256(oracleQueryData)
    let eipId = "0xdfaa6f747f0f012e8f2069d6ecacff25f5cdf0258702051747439949737fc0b5"
    assert(await bench.idMappingContract() == mappingContract.address, "mapping contract should be set correctly")
    await playground.connect(addr1).submitValue(oracleQueryId,1500,0,oracleQueryData)
    blocky1 = await h.getBlock()
    let gvfData = await bench.valueFor(eipId);
    assert(gvfData[0] == 1500, "value should be correct")
    assert(gvfData[1] == blocky1.timestamp, "timestamp should be correct")
    assert(gvfData[2] == 200, "status code should be correct");
	})

  it("setIdMappingContract()", async function() {
    await bench.setIdMappingContract(mappingContract.address);
    assert(await bench.idMappingContract() == mappingContract.address, "mapping contract should be set correctly")
    await h.expectThrow(bench.setIdMappingContract(addr1.address), "mapping contract should not be able to be called twice")
  })

	it("tellor()", async function() {
		expect(await bench.tellor()).to.equal(playground.address)
	})

  it("getIndexForDataAfter()", async function() {
    blocky0 = await h.getBlock()
    result = await bench.getIndexForDataAfter(h.uintTob32(1), blocky0.timestamp)
    expect(result[0]).to.be.false
    expect(result[1]).to.equal(0)

    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(150),0,'0x')
    blocky1 = await h.getBlock()

    result = await bench.getIndexForDataAfter(h.uintTob32(1), blocky0.timestamp)
    expect(result[0]).to.be.true
    expect(result[1]).to.equal(0)
    result = await bench.getIndexForDataAfter(h.uintTob32(1), blocky1.timestamp)
    expect(result[0]).to.be.false
    expect(result[1]).to.equal(0)

    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(160),1,'0x')
    blocky2 = await h.getBlock()

    result = await bench.getIndexForDataAfter(h.uintTob32(1), blocky0.timestamp)
    expect(result[0]).to.be.true
    expect(result[1]).to.equal(0)
    result = await bench.getIndexForDataAfter(h.uintTob32(1), blocky1.timestamp)
    expect(result[0]).to.be.true
    expect(result[1]).to.equal(1)
    result = await bench.getIndexForDataAfter(h.uintTob32(1), blocky2.timestamp)
    expect(result[0]).to.be.false
    expect(result[1]).to.equal(0)
  })

  it("getDataAfter()", async function() {
    blocky0 = await h.getBlock()
    // result = await bench.getDataAfter(h.uintTob32(1), blocky0.timestamp)
    // expect(result[0]).to.equal('0x')
    // expect(result[1]).to.equal(0)

    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(150),0,'0x')
    blocky1 = await h.getBlock()

    result = await bench.getDataAfter(h.uintTob32(1), blocky0.timestamp)
    expect(result[0]).to.equal(h.uintTob32(150))
    expect(result[1]).to.equal(blocky1.timestamp)
    result = await bench.getDataAfter(h.uintTob32(1), blocky1.timestamp)
    expect(result[0]).to.equal('0x')
    expect(result[1]).to.equal(0)

    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(160),1,'0x')
    blocky2 = await h.getBlock()

    result = await bench.getDataAfter(h.uintTob32(1), blocky0.timestamp)
    expect(result[0]).to.equal(h.uintTob32(150))
    expect(result[1]).to.equal(blocky1.timestamp)
    result = await bench.getDataAfter(h.uintTob32(1), blocky1.timestamp)
    expect(result[0]).to.equal(h.uintTob32(160))
    expect(result[1]).to.equal(blocky2.timestamp)
    result = await bench.getDataAfter(h.uintTob32(1), blocky2.timestamp)
    expect(result[0]).to.equal('0x')
    expect(result[1]).to.equal(0)
  })

  it("getMultipleValuesBefore", async function() {
    // submit 2 values
    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(150),0,'0x')
    blocky1 = await h.getBlock()
    timestamp = await playground.getTimestampbyQueryIdandIndex(h.uintTob32(1), 0)
    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(160),0,'0x')
    blocky2 = await h.getBlock()
    await h.advanceTime(10)
    blockyNow0 = await h.getBlock()

    // 1 hour before 1st submission
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky1.timestamp - 3600, 3600, 4)
    expect(result[0].length).to.equal(0)
    expect(result[1].length).to.equal(0)

    // maxCount = 4
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow0.timestamp, 3600, 4)
    expect(result[0].length).to.equal(2)
    expect(result[1].length).to.equal(2)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[0][1]).to.equal(h.uintTob32(160))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[1][1]).to.equal(blocky2.timestamp)

    // maxCount = 3
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow0.timestamp, 3600, 3)
    expect(result[0].length).to.equal(2)
    expect(result[1].length).to.equal(2)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[0][1]).to.equal(h.uintTob32(160))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[1][1]).to.equal(blocky2.timestamp)

    // maxCount = 2
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow0.timestamp, 3600, 2)
    expect(result[0].length).to.equal(2)
    expect(result[1].length).to.equal(2)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[0][1]).to.equal(h.uintTob32(160))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[1][1]).to.equal(blocky2.timestamp)

    // maxCount = 1
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow0.timestamp, 3600, 1)
    expect(result[0].length).to.equal(1)
    expect(result[1].length).to.equal(1)
    expect(result[0][0]).to.equal(h.uintTob32(160))
    expect(result[1][0]).to.equal(blocky2.timestamp)

    // maxAge = 5
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow0.timestamp, 5, 4)
    expect(result[0].length).to.equal(0)
    expect(result[1].length).to.equal(0)

    // submit another 2 values
    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(170),0,'0x')
    blocky3 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(180),0,'0x')
    blocky4 = await h.getBlock()
    await h.advanceTime(10)
    blockyNow1 = await h.getBlock()

    // maxCount = 6, don't update blocky
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow0.timestamp, 3600, 6)
    expect(result[0].length).to.equal(2)
    expect(result[1].length).to.equal(2)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[0][1]).to.equal(h.uintTob32(160))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[1][1]).to.equal(blocky2.timestamp)

    // maxCount = 6, update blocky
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow1.timestamp, 3600, 6)
    expect(result[0].length).to.equal(4)
    expect(result[1].length).to.equal(4)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[0][1]).to.equal(h.uintTob32(160))
    expect(result[0][2]).to.equal(h.uintTob32(170))
    expect(result[0][3]).to.equal(h.uintTob32(180))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[1][1]).to.equal(blocky2.timestamp)
    expect(result[1][2]).to.equal(blocky3.timestamp)
    expect(result[1][3]).to.equal(blocky4.timestamp)

    // maxCount = 5
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow1.timestamp, 3600, 5)
    expect(result[0].length).to.equal(4)
    expect(result[1].length).to.equal(4)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[0][1]).to.equal(h.uintTob32(160))
    expect(result[0][2]).to.equal(h.uintTob32(170))
    expect(result[0][3]).to.equal(h.uintTob32(180))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[1][1]).to.equal(blocky2.timestamp)
    expect(result[1][2]).to.equal(blocky3.timestamp)
    expect(result[1][3]).to.equal(blocky4.timestamp)

    // maxCount = 4
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow1.timestamp, 3600, 4)
    expect(result[0].length).to.equal(4)
    expect(result[1].length).to.equal(4)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[0][1]).to.equal(h.uintTob32(160))
    expect(result[0][2]).to.equal(h.uintTob32(170))
    expect(result[0][3]).to.equal(h.uintTob32(180))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[1][1]).to.equal(blocky2.timestamp)
    expect(result[1][2]).to.equal(blocky3.timestamp)
    expect(result[1][3]).to.equal(blocky4.timestamp)

    // maxCount = 3
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow1.timestamp, 3600, 3)
    expect(result[0].length).to.equal(3)
    expect(result[1].length).to.equal(3)
    expect(result[0][0]).to.equal(h.uintTob32(160))
    expect(result[0][1]).to.equal(h.uintTob32(170))
    expect(result[0][2]).to.equal(h.uintTob32(180))
    expect(result[1][0]).to.equal(blocky2.timestamp)
    expect(result[1][1]).to.equal(blocky3.timestamp)
    expect(result[1][2]).to.equal(blocky4.timestamp)

    // maxCount = 2
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow1.timestamp, 3600, 2)
    expect(result[0].length).to.equal(2)
    expect(result[1].length).to.equal(2)
    expect(result[0][0]).to.equal(h.uintTob32(170))
    expect(result[0][1]).to.equal(h.uintTob32(180))
    expect(result[1][0]).to.equal(blocky3.timestamp)
    expect(result[1][1]).to.equal(blocky4.timestamp)

    // maxCount = 1
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow1.timestamp, 3600, 1)
    expect(result[0].length).to.equal(1)
    expect(result[1].length).to.equal(1)
    expect(result[0][0]).to.equal(h.uintTob32(180))
    expect(result[1][0]).to.equal(blocky4.timestamp)

    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky4.timestamp, 100, 3)
  })

  it("sliceUint", async() => {
    let val1 = h.uintTob32(1)
    let val2 = h.uintTob32(2)
    let val3 = h.uintTob32(300000000000000)
    let val4 = h.uintTob32(300000000000001)
    let val5 = abiCoder.encode(["uint256"], [1])
    let val6 = abiCoder.encode(["uint256"], ["21010191828172717718232237237237128"])
    let val7 = '0x01'
    let val8 = '0x10'

    expect(await bench.sliceUint(val1)).to.equal(1)
    expect(await bench.sliceUint(val2)).to.equal(2)
    expect(await bench.sliceUint(val3)).to.equal(300000000000000)
    expect(await bench.sliceUint(val4)).to.equal(300000000000001)
    expect(await bench.sliceUint(val5)).to.equal(1)
    expect(await bench.sliceUint(val6)).to.equal("21010191828172717718232237237237128")
    expect(await bench.sliceUint(val7)).to.equal(1)
    expect(await bench.sliceUint(val8)).to.equal(16)
  })

  it("getReporterByTimestamp()", async function() {
    await playground.connect(addr1).submitValue(h.uintTob32(1),150,0,'0x')
    blocky1 = await h.getBlock()
    expect(await bench.getReporterByTimestamp(h.uintTob32(1), blocky1.timestamp)).to.equal(addr1.address)
    await playground.connect(addr2).submitValue(h.uintTob32(1),160,1,'0x')
    blocky2 = await h.getBlock()
    expect(await bench.getReporterByTimestamp(h.uintTob32(1), blocky2.timestamp)).to.equal(addr2.address)
  })
});
