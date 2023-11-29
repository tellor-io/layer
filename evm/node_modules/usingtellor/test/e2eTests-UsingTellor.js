const { expect, assert } = require("chai");
const { ethers } = require("hardhat");
const h = require("./helpers/helpers");
const web3 = require('web3');
let abiCoder = new ethers.utils.AbiCoder

const QUERY_DATA_1 = h.uintTob32(1);
const QUERY_ID_1 = h.hash(QUERY_DATA_1);

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

  it("getDataAfter without disputes", async function() {
    // setup
    let queryData1 = h.uintTob32(1);
    let queryId1 = h.hash(queryData1);

    // no data
    dataRetrieved = await bench.getDataAfter(queryId1,1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)
    

    // one data point
    await playground.connect(addr1).submitValue(queryId1,150,0,queryData1)
    blocky1 = await h.getBlock()

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // two data points
    await h.advanceTime(10)
    await playground.connect(addr1).submitValue(queryId1,160,1,queryData1)
    blocky2 = await h.getBlock()

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // three data points
    await h.advanceTime(10)
    await playground.connect(addr1).submitValue(queryId1,170,2,queryData1)
    blocky3 = await h.getBlock()

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // four data points
    await h.advanceTime(10)
    await playground.connect(addr1).submitValue(queryId1,180,3,queryData1)
    blocky4 = await h.getBlock()

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky4.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky4.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // five data points
    await h.advanceTime(10)
    await playground.connect(addr1).submitValue(queryId1,190,4,queryData1)
    blocky5 = await h.getBlock()

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky4.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(190))
    expect(dataRetrieved[1]).to.equal(blocky5.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky4.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(190))
    expect(dataRetrieved[1]).to.equal(blocky5.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId1,blocky5.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)
  })

  it("getDataAfter, one disputed data point", async function() {
    await playground.submitValue(QUERY_ID_1, 150, 0, QUERY_DATA_1)
    blocky1 = await h.getBlock()
    await playground.beginDispute(QUERY_ID_1, blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)
  })

  it("getDataAfter, 2 values, one dispute", async function() {
    // disputed, non-disputed
    await playground.submitValue(QUERY_ID_1, 150, 0, QUERY_DATA_1)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(QUERY_ID_1, 160, 1, QUERY_DATA_1)
    blocky2 = await h.getBlock()

    await playground.beginDispute(QUERY_ID_1, blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // non-disputed, disputed
    let queryData2 = h.uintTob32(2)
    let queryId2 = h.hash(queryData2)

    await playground.submitValue(queryId2, 150, 0, queryData2)
    blocky1 = await h.getBlock()

    await playground.submitValue(queryId2, 160, 1, queryData2)
    blocky2 = await h.getBlock()

    await playground.beginDispute(queryId2, blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)
  })

  it("getDataAfter, 3 values, 1 dispute", async function() {
    // disputed, non-disputed, non-disputed
    await playground.submitValue(QUERY_ID_1, 150, 0, QUERY_DATA_1)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(QUERY_ID_1, 160, 1, QUERY_DATA_1)
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(QUERY_ID_1, 170, 2, QUERY_DATA_1)
    blocky3 = await h.getBlock()

    await playground.beginDispute(QUERY_ID_1, blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // non-disputed, disputed, non-disputed
    let queryData2 = h.uintTob32(2)
    let queryId2 = h.hash(queryData2)

    await playground.submitValue(queryId2, 150, 0, queryData2)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId2, 160, 1, queryData2)
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId2, 170, 2, queryData2)
    blocky3 = await h.getBlock()

    await playground.beginDispute(queryId2, blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // non-disputed, non-disputed, disputed
    let queryData3 = h.uintTob32(3)
    let queryId3 = h.hash(queryData3)

    await playground.submitValue(queryId3, 150, 0, queryData3)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId3, 160, 1, queryData3)
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId3, 170, 2, queryData3)
    blocky3 = await h.getBlock()

    await playground.beginDispute(queryId3, blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)
    
    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)
  })

  it("getDataAfter, 3 values, 2 to 3 disputes", async function() {
    // disputed, disputed, non-disputed
    await playground.submitValue(QUERY_ID_1, 150, 0, h.uintTob32(1))
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(QUERY_ID_1, 160, 1, h.uintTob32(1))
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(QUERY_ID_1, 170, 2, h.uintTob32(1))
    blocky3 = await h.getBlock()

    await playground.beginDispute(QUERY_ID_1, blocky1.timestamp)
    await playground.beginDispute(QUERY_ID_1, blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // disputed, non-disputed, disputed
    queryData2 = h.uintTob32(2)
    queryId2 = h.hash(queryData2)
    await playground.submitValue(queryId2, 150, 0, queryData2)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId2, 160, 1, queryData2)
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId2, 170, 2, queryData2)
    blocky3 = await h.getBlock()

    await playground.beginDispute(queryId2, blocky1.timestamp)
    await playground.beginDispute(queryId2, blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // non-disputed, disputed, disputed
    queryData3 = h.uintTob32(3)
    queryId3 = h.hash(queryData3)

    await playground.submitValue(queryId3, 150, 0, queryData3)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId3, 160, 1, queryData3)
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId3, 170, 2, queryData3)
    blocky3 = await h.getBlock()

    await playground.beginDispute(queryId3, blocky2.timestamp)
    await playground.beginDispute(queryId3, blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // disputed, disputed, disputed
    await playground.beginDispute(queryId3, blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)
  })

  it("getDataAfter, 4 values, 1 dispute", async function() {
    // disputed, non-disputed, non-disputed, non-disputed
    await playground.submitValue(QUERY_ID_1, 150, 0, QUERY_DATA_1)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(QUERY_ID_1, 160, 1, QUERY_DATA_1)
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(QUERY_ID_1, 170, 2, QUERY_DATA_1)
    blocky3 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(QUERY_ID_1, 180, 3, QUERY_DATA_1)
    blocky4 = await h.getBlock()

    await playground.beginDispute(QUERY_ID_1, blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky4.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(QUERY_ID_1, blocky4.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // non-disputed, disputed, non-disputed, non-disputed
    queryData2 = h.uintTob32(2)
    queryId2 = h.hash(queryData2)

    await playground.submitValue(queryId2, 150, 0, queryData2)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId2, 160, 1, queryData2)
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId2, 170, 2, queryData2)
    blocky3 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId2, 180, 3, queryData2)
    blocky4 = await h.getBlock()

    await playground.beginDispute(queryId2, blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)
    
    dataRetrieved = await bench.getDataAfter(queryId2, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky4.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId2, blocky4.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // non-disputed, non-disputed, disputed, non-disputed
    queryData3 = h.uintTob32(3)
    queryId3 = h.hash(queryData3)

    await playground.submitValue(queryId3, 150, 0, queryData3)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId3, 160, 1, queryData3)
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId3, 170, 2, queryData3)
    blocky3 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId3, 180, 3, queryData3)
    blocky4 = await h.getBlock()

    await playground.beginDispute(queryId3, blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)
    
    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(180))
    expect(dataRetrieved[1]).to.equal(blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky4.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId3, blocky4.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)


    // non-disputed, non-disputed, non-disputed, disputed
    queryData4 = h.uintTob32(4)
    queryId4 = h.hash(queryData4)

    await playground.submitValue(queryId4, 150, 0, queryData4)
    blocky1 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId4, 160, 1, queryData4)
    blocky2 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId4, 170, 2, queryData4)
    blocky3 = await h.getBlock()

    await h.advanceTime(10)
    await playground.submitValue(queryId4, 180, 3, queryData4)
    blocky4 = await h.getBlock()

    await playground.beginDispute(queryId4, blocky4.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId4, blocky1.timestamp - 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(150))
    expect(dataRetrieved[1]).to.equal(blocky1.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId4, blocky1.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId4, blocky1.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(160))
    expect(dataRetrieved[1]).to.equal(blocky2.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId4, blocky2.timestamp)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId4, blocky2.timestamp + 1)
    expect(dataRetrieved[0]).to.equal(h.bytes(170))
    expect(dataRetrieved[1]).to.equal(blocky3.timestamp)

    dataRetrieved = await bench.getDataAfter(queryId4, blocky3.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId4, blocky3.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId4, blocky4.timestamp)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)

    dataRetrieved = await bench.getDataAfter(queryId4, blocky4.timestamp + 1)
    expect(dataRetrieved[0]).to.equal('0x')
    expect(dataRetrieved[1]).to.equal(0)
  })

  it("getMultipleValuesBefore", async function() {
    // submit 4 values
    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(150),0,'0x')
    blocky1 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(160),0,'0x')
    blocky2 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(170),0,'0x')
    blocky3 = await h.getBlock()
    await playground.connect(addr1).submitValue(h.uintTob32(1),h.uintTob32(180),0,'0x')
    blocky4 = await h.getBlock()

    await h.advanceTime(10)
    blockyNow0 = await h.getBlock()

    // dispute 2nd value
    await playground.connect(addr1).beginDispute(h.uintTob32(1), blocky2.timestamp)

    // check from blockyNow
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow0.timestamp, 3600, 4)
    expect(result[0].length).to.equal(3)
    expect(result[1].length).to.equal(3)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[0][1]).to.equal(h.uintTob32(170))
    expect(result[1][1]).to.equal(blocky3.timestamp)
    expect(result[0][2]).to.equal(h.uintTob32(180))
    expect(result[1][2]).to.equal(blocky4.timestamp)

    // check from blocky4
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky4.timestamp, 3600, 4)
    expect(result[0].length).to.equal(2)
    expect(result[1].length).to.equal(2)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[0][1]).to.equal(h.uintTob32(170))
    expect(result[1][1]).to.equal(blocky3.timestamp)

    // check from blocky3
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky3.timestamp, 3600, 4)
    expect(result[0].length).to.equal(1)
    expect(result[1].length).to.equal(1)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[1][0]).to.equal(blocky1.timestamp)

    // check from blocky2
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky2.timestamp, 3600, 4)
    expect(result[0].length).to.equal(1)
    expect(result[1].length).to.equal(1)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[1][0]).to.equal(blocky1.timestamp)

    // check from blocky1
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky1.timestamp, 3600, 4)
    expect(result[0].length).to.equal(0)
    expect(result[1].length).to.equal(0)

    // dispute 3rd value
    await playground.connect(addr1).beginDispute(h.uintTob32(1), blocky3.timestamp)

    // check from blockyNow
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blockyNow0.timestamp, 3600, 4)
    expect(result[0].length).to.equal(2)
    expect(result[1].length).to.equal(2)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[1][0]).to.equal(blocky1.timestamp)
    expect(result[0][1]).to.equal(h.uintTob32(180))
    expect(result[1][1]).to.equal(blocky4.timestamp)

    // check from blocky4
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky4.timestamp, 3600, 4)
    expect(result[0].length).to.equal(1)
    expect(result[1].length).to.equal(1)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[1][0]).to.equal(blocky1.timestamp)

    // check from blocky3
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky3.timestamp, 3600, 4)
    expect(result[0].length).to.equal(1)
    expect(result[1].length).to.equal(1)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[1][0]).to.equal(blocky1.timestamp)

    // check from blocky2
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky2.timestamp, 3600, 4)
    expect(result[0].length).to.equal(1)
    expect(result[1].length).to.equal(1)
    expect(result[0][0]).to.equal(h.uintTob32(150))
    expect(result[1][0]).to.equal(blocky1.timestamp)

    // check from blocky1
    result = await bench.getMultipleValuesBefore(h.uintTob32(1), blocky1.timestamp, 3600, 4)
    expect(result[0].length).to.equal(0)
    expect(result[1].length).to.equal(0)
    

  })
})