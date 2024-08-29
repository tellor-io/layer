const { expect } = require("chai");
const { ethers } = require("hardhat");
const h = require("./helpers/evmHelpers");
var assert = require('assert');
const abiCoder = new ethers.utils.AbiCoder();


describe("TokenBridge - Function Tests", async function () {

    let blobstream, accounts, guardian, tbridge, token, blocky0,
        valTs, valParams, valSet, initialValAddrs, initialPowers, threshold;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks
    const WITHDRAW1_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, 1])
    const WITHDRAW1_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridge", WITHDRAW1_QUERY_DATA_ARGS])
    const WITHDRAW1_QUERY_ID = h.hash(WITHDRAW1_QUERY_DATA)
    const EVM_RECIPIENT = "0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0"
    const LAYER_RECIPIENT = "tellor1zy50vdk8fdae0var2ryjhj2ysxtcm8dp2qtckd"
    const INITIAL_LAYER_TOKEN_SUPPLY = h.toWei("100")


    beforeEach(async function () {
        // init accounts
        accounts = await ethers.getSigners();
        guardian = accounts[10]
        val1 = ethers.Wallet.createRandom()
        val2 = ethers.Wallet.createRandom()
        initialValAddrs = [val1.address,val2.address]
        initialPowers = [1, 2]
        threshold = 2
        blocky = await h.getBlock()
        valTimestamp = (blocky.timestamp - 2) * 1000
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
        // deploy contracts
        blobstream = await ethers.deployContract(
            "BlobstreamO", [
            guardian.address
        ]
        )
        await blobstream.init(threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint)
        token = await ethers.deployContract("TellorPlayground")
        oldOracle = await ethers.deployContract("TellorPlayground")
        tbridge = await ethers.deployContract("TestTokenBridge", [token.address,blobstream.address, oldOracle.address])
        blocky0 = await h.getBlock()
        // fund accounts
        await token.faucet(accounts[0].address)
        await token.faucet(accounts[10].address)
        await token.connect(accounts[10]).transfer(tbridge.address, INITIAL_LAYER_TOKEN_SUPPLY)

        // sleep 1 second for api rate limit
        await new Promise(r => setTimeout(r, 1000));
    });

    it("constructor", async function () {
        assert.equal(await tbridge.token(), await token.address)
        assert.equal(await tbridge.bridge(), await blobstream.address)
    })
    it.only("withdrawFromLayer", async function () {
        depositAmount = h.toWei("20")
        tip = h.toWei("0")
        await h.expectThrow(tbridge.depositToLayer(depositAmount, tip, LAYER_RECIPIENT)) // not approved
        await token.approve(await tbridge.address, h.toWei("100"))
        await tbridge.depositToLayer(depositAmount, tip, LAYER_RECIPIENT)
        await h.advanceTime(43200)
        value = h.getWithdrawValue(EVM_RECIPIENT,LAYER_RECIPIENT,20)
        blocky = await h.getBlock()
        timestamp = (blocky.timestamp - 43200) * 1000
        aggregatePower = 3
        attestTimestamp = blocky.timestamp * 1000
        previousTimestamp = 0
        nextTimestamp = 0
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint = await h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
        dataDigest = await h.getDataDigest(
            WITHDRAW1_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            valCheckpoint,
            attestTimestamp
        )
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await h.layerSign(dataDigest, val1.privateKey)
        sig2 = await h.layerSign(dataDigest, val2.privateKey)
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        oracleDataStruct = await h.getOracleDataStruct(
            WITHDRAW1_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            attestTimestamp
        )
        await tbridge.withdrawFromLayer(
            oracleDataStruct,
            currentValSetArray,
            sigStructArray,
            1,
        )
        recipientBal = await token.balanceOf(EVM_RECIPIENT)
        expectedBal = 20e12 // 20 loya
        assert.equal(recipientBal.toString(), expectedBal)

        // assemble another withdraw, freeze bridge, then unfreeze
        await token.faucet(accounts[0].address)
        await token.transfer(tbridge.address, h.toWei("1000"))
        await h.advanceTime(43200)
        blocky = await h.getBlock()
        timestamp = (blocky.timestamp - 43200) * 1000
        attestTimestamp = blocky.timestamp * 1000
        WITHDRAW2_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, 2])
        WITHDRAW2_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridge", WITHDRAW2_QUERY_DATA_ARGS])
        WITHDRAW2_QUERY_ID = h.hash(WITHDRAW2_QUERY_DATA)
        value = h.getWithdrawValue(EVM_RECIPIENT,LAYER_RECIPIENT,20)
        dataDigest = await h.getDataDigest(
            WITHDRAW2_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            valCheckpoint,
            attestTimestamp
        )
        sig1 = await h.layerSign(dataDigest, val1.privateKey)
        sig2 = await h.layerSign(dataDigest, val2.privateKey)
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        oracleDataStruct = await h.getOracleDataStruct(
            WITHDRAW2_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            attestTimestamp
        )
        for (let i = 0; i < 10; i++) {
            await token.faucet(guardian.address)
        }
        await token.connect(guardian).approve(tbridge.address, h.toWei("10000"))
        await tbridge.connect(guardian).pauseBridge()
        await h.expectThrow(tbridge.withdrawFromLayer(
            oracleDataStruct,
            currentValSetArray,
            sigStructArray,
            2
        ))
        balanceDead = await token.balanceOf("0x000000000000000000000000000000000000dEaD")
        assert.equal(balanceDead.toString(), h.toWei("10000"))

        await h.advanceTime(86400 * 21)
        // update the validator set
        blocky = await h.getBlock()
        valTimestamp = (blocky.timestamp - 2) * 1000
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
        await blobstream.connect(guardian).guardianResetValidatorSet(threshold, valTimestamp, valCheckpoint)

        // withdraw
        timestamp = (blocky.timestamp - 43200) * 1000
        attestTimestamp = blocky.timestamp * 1000
        value = h.getWithdrawValue(EVM_RECIPIENT,LAYER_RECIPIENT,20)
        dataDigest = await h.getDataDigest(
            WITHDRAW2_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            valCheckpoint,
            attestTimestamp
        )
        sig1 = await h.layerSign(dataDigest, val1.privateKey)
        sig2 = await h.layerSign(dataDigest, val2.privateKey)
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        oracleDataStruct = await h.getOracleDataStruct(
            WITHDRAW2_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            attestTimestamp
        )
        await tbridge.withdrawFromLayer(
            oracleDataStruct,
            currentValSetArray,
            sigStructArray,
            2
        )
        recipientBal = await token.balanceOf(EVM_RECIPIENT)
        expectedBal = 40e12 // 40 loya
        assert.equal(recipientBal.toString(), expectedBal)
    })
    it("depositToLayer", async function () {
        depositAmount = h.toWei("1")
        tip = h.toWei(".01")
        assert.equal(await token.balanceOf(await accounts[0].address), h.toWei("1000"))
        assert.equal(await token.balanceOf(await tbridge.address), INITIAL_LAYER_TOKEN_SUPPLY)
        await h.expectThrow(tbridge.depositToLayer(depositAmount, tip, LAYER_RECIPIENT)) // not approved
        await token.approve(await tbridge.address, h.toWei("900"))
        await h.expectThrow(tbridge.depositToLayer(0, tip, LAYER_RECIPIENT)) // zero amount
        await h.expectThrow(tbridge.depositToLayer(h.toWei("21"), tip, LAYER_RECIPIENT)) // over limit
        await h.expectThrow(tbridge.depositToLayer(depositAmount, h.toWei("1.01"), LAYER_RECIPIENT)) // tip over amount
        await tbridge.depositToLayer(depositAmount, tip, LAYER_RECIPIENT)
        blocky1 = await h.getBlock()
        tbridgeBal = await token.balanceOf(await tbridge.address)
        expBalBridge = BigInt(depositAmount) + BigInt(INITIAL_LAYER_TOKEN_SUPPLY)
        assert.equal(tbridgeBal.toString(), expBalBridge.toString())
        userBal = await token.balanceOf(await accounts[0].address)
        assert.equal(userBal.toString(), h.toWei("999"))
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10) - BigInt(depositAmount)
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
        await tbridge.refreshDepositLimit(1)
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
        assert.equal(await tbridge.depositId(), 1)
        depositDetails = await tbridge.deposits(1)
        assert.equal(depositDetails.amount.toString(), depositAmount)
        assert.equal(depositDetails.tip.toString(), tip)
        assert.equal(depositDetails.recipient, LAYER_RECIPIENT)
        assert.equal(depositDetails.sender, await accounts[0].address)
        assert.equal(depositDetails.blockHeight, blocky1.number)
        assert.equal(await tbridge.depositId(), 1)
        await h.advanceTime(43200)
        expectedDepositLimit2 = (BigInt(100e18) + BigInt(depositAmount)) * BigInt(2) / BigInt(10)
        await tbridge.refreshDepositLimit(1)
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit2);
    })

    it("depositLimit", async function () {
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10)
        await tbridge.refreshDepositLimit(1)
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
        await token.approve(await tbridge.address, h.toWei("900"))
        depositAmount = h.toWei("2")
        tip = h.toWei("0")
        await tbridge.depositToLayer(depositAmount, tip, LAYER_RECIPIENT)
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10) - BigInt(depositAmount)
        await tbridge.refreshDepositLimit(1)
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
        await h.advanceTime(43200)
        expectedDepositLimit2 = (BigInt(100e18) + BigInt(depositAmount)) / BigInt(5)
        await tbridge.refreshDepositLimit(1)
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit2);
    })

    it("withdrawLimit", async function () {
        expectedWithdrawLimit = BigInt(100e18) / BigInt(20)
        withdrawLimit = await tbridge.withdrawLimit()
        assert(withdrawLimit == expectedWithdrawLimit, "withdrawLimit should be correct")
    })
    it("claimExtraWithdraw", async function () {
        const WITHDRAW_AMOUNT = h.toWei("10")
        let _addy = await accounts[2].address
        value = h.getWithdrawValue(_addy,LAYER_RECIPIENT,BigInt(WITHDRAW_AMOUNT) / BigInt(1e12))
        blocky = await h.getBlock()
        timestamp = (blocky.timestamp - 2) * 1000
        aggregatePower = 3
        attestTimestamp = (blocky.timestamp + 43200) * 1000
        previousTimestamp = 0
        nextTimestamp = 0
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint = await h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
        dataDigest = await h.getDataDigest(
            WITHDRAW1_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            valCheckpoint,
            attestTimestamp
        )
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await h.layerSign(dataDigest, val1.privateKey)
        sig2 = await h.layerSign(dataDigest, val2.privateKey)
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        oracleDataStruct = await h.getOracleDataStruct(
            WITHDRAW1_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            attestTimestamp
        )
        await h.advanceTime(43200)
        expectedWithdrawLimit = BigInt(INITIAL_LAYER_TOKEN_SUPPLY) / BigInt(20)
        let _limit0 = await tbridge.withdrawLimit.call()
        assert(_limit0 == expectedWithdrawLimit, "withdrawLimit should be correct")
        assert(await token.balanceOf(_addy) == 0)
        await tbridge.withdrawFromLayer(
            oracleDataStruct,
            currentValSetArray,
            sigStructArray,
            1,
        )
        recipientBal0 = await token.balanceOf(_addy)
        assert(recipientBal0 - _limit0 == 0, "token balance should be correct")
        tokensToClaim = await tbridge.tokensToClaim(accounts[2].address)
        assert(tokensToClaim == BigInt(WITHDRAW_AMOUNT) - BigInt(recipientBal0), "tokensToClaim should be correct")
        await h.expectThrow(tbridge.claimExtraWithdraw(await accounts[2].address))
        await h.advanceTime(43200)
        _limit1 = await tbridge.withdrawLimit.call()

        await tbridge.claimExtraWithdraw(await accounts[2].address);
        await h.expectThrow(tbridge.claimExtraWithdraw(await accounts[2].address))
        recipientBal1 = await token.balanceOf(await accounts[2].address)
        assert(recipientBal1 == BigInt(recipientBal0) + BigInt(_limit1), "token balance should be correct")
        tokensToClaim = await tbridge.tokensToClaim(accounts[2].address)
        assert(tokensToClaim == BigInt(WITHDRAW_AMOUNT) - BigInt(recipientBal1), "tokensToClaim should be correct")
        await h.advanceTime(43200)
        
        await tbridge.claimExtraWithdraw(await accounts[2].address);
        await h.expectThrow(tbridge.claimExtraWithdraw(await accounts[2].address))
        recipientBal2 = await token.balanceOf(await accounts[2].address)
        assert(recipientBal2 == WITHDRAW_AMOUNT, "token balance should be correct")
        tokensToClaim = await tbridge.tokensToClaim(accounts[2].address)
        assert(tokensToClaim == BigInt(0), "tokensToClaim should be correct")
    })
    // more complex tests
    it("100 deposits and withdrawals", async function () {
        this.timeout(300000)
        // fund accts
        await token.faucet(accounts[0].address)
        await token.faucet(accounts[1].address)
        await token.faucet(accounts[1].address)
        await token.connect(accounts[0]).approve(tbridge.address, h.toWei("10000"))
        await token.connect(accounts[1]).approve(tbridge.address, h.toWei("10000"))

        initUserBal0 = await token.balanceOf(accounts[0].address)
        initUserBal1 = await token.balanceOf(accounts[1].address)
        niters = 100
        depositAmount0 = h.toWei("5")
        depositAmount1 = h.toWei("10")
        tip = h.toWei("0")
        // deposits
        for (let i = 0; i < niters; i++) {
            await tbridge.connect(accounts[0]).depositToLayer(depositAmount0, tip, LAYER_RECIPIENT)
            await tbridge.connect(accounts[1]).depositToLayer(depositAmount1, tip, LAYER_RECIPIENT)
            await h.advanceTime(43200)
        }
        // checks
        userBal0 = await token.balanceOf(accounts[0].address)
        userBal1 = await token.balanceOf(accounts[1].address)
        bridgeBal = await token.balanceOf(await tbridge.address)
        expectedBal0 = BigInt(initUserBal0) - BigInt(depositAmount0) * BigInt(niters)
        expectedBal1 = BigInt(initUserBal1) - BigInt(depositAmount1) * BigInt(niters)
        expectedBalBridge = BigInt(depositAmount0) * BigInt(niters) + BigInt(depositAmount1) * BigInt(niters) + BigInt(INITIAL_LAYER_TOKEN_SUPPLY)
        assert(BigInt(userBal0) == expectedBal0, "user 0 balance should be correct")
        assert(BigInt(userBal1) == expectedBal1, "user 1 balance should be correct")
        assert(BigInt(bridgeBal) == expectedBalBridge, "bridge balance should be correct")
        assert(await tbridge.depositId() == BigInt(niters * 2), "deposit id should be correct")

        // withdrawals
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint = await h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
        withdrawValue0 = h.getWithdrawValue(accounts[0].address, LAYER_RECIPIENT, BigInt(depositAmount0) / BigInt(1e12))
        withdrawValue1 = h.getWithdrawValue(accounts[1].address, LAYER_RECIPIENT, BigInt(depositAmount1) / BigInt(1e12))
        aggregatePower = 3
        expTokensToClaim0 = BigInt(0)
        expTokensToClaim1 = BigInt(0)
        
        for (let i = 0; i<niters; i++) {
            // guardian reset valset, past unbonding period
            blocky = await h.getBlock()
            validatorTimestamp = await blobstream.validatorTimestamp()
            if (blocky.timestamp - (validatorTimestamp / 1000) > UNBONDING_PERIOD) {
                valTimestamp = (blocky.timestamp - 2) * 1000
                newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
                valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
                await blobstream.connect(guardian).guardianResetValidatorSet(threshold, valTimestamp, valCheckpoint)
            }
            withdrawId0 = i * 2 + 1
            withdrawId1 = i * 2 + 2
            withdrawQueryDataArgs0 = abiCoder.encode(['bool', 'uint256'], [false, withdrawId0])
            withdrawQueryDataArgs1 = abiCoder.encode(['bool', 'uint256'], [false, withdrawId1])
            withdrawQueryData0 = abiCoder.encode(['string', 'bytes'], ['TRBBridge', withdrawQueryDataArgs0])
            withdrawQueryData1 = abiCoder.encode(['string', 'bytes'], ['TRBBridge', withdrawQueryDataArgs1])
            withdrawQueryId0 = h.hash(withdrawQueryData0)
            withdrawQueryId1 = h.hash(withdrawQueryData1)
            blocky = await h.getBlock()
            reportTimestamp = (blocky.timestamp - 84600) * 1000
            attestationTimestamp = blocky.timestamp * 1000
            dataDigest0 = await h.getDataDigest(
                withdrawQueryId0,
                withdrawValue0,
                reportTimestamp,
                aggregatePower,
                reportTimestamp - 1,
                0,
                valCheckpoint,
                attestationTimestamp
            )
            dataDigest1 = await h.getDataDigest(
                withdrawQueryId1,
                withdrawValue1,
                reportTimestamp,
                aggregatePower,
                reportTimestamp - 1,
                0,
                valCheckpoint,
                attestationTimestamp
            )
            currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
            sig0_1 = await h.layerSign(dataDigest0, val1.privateKey)
            sig0_2 = await h.layerSign(dataDigest0, val2.privateKey)
            sig1_1 = await h.layerSign(dataDigest1, val1.privateKey)
            sig1_2 = await h.layerSign(dataDigest1, val2.privateKey)
            sigStructArray0 = await h.getSigStructArray([sig0_1, sig0_2])
            sigStructArray1 = await h.getSigStructArray([sig1_1, sig1_2])
            oracleDataStruct0 = await h.getOracleDataStruct(
                withdrawQueryId0,
                withdrawValue0,
                reportTimestamp,
                aggregatePower,
                reportTimestamp - 1,
                0,
                attestationTimestamp
            )
            oracleDataStruct1 = await h.getOracleDataStruct(
                withdrawQueryId1,
                withdrawValue1,
                reportTimestamp,
                aggregatePower,
                reportTimestamp - 1,
                0,
                attestationTimestamp
            )

            withdrawLimitBefore0 = await tbridge.withdrawLimit()
            await tbridge.withdrawFromLayer(
                oracleDataStruct0,
                currentValSetArray,
                sigStructArray0,
                withdrawId0,
            )
            withdrawLimitBefore1 = await tbridge.withdrawLimit()
            await tbridge.withdrawFromLayer(
                oracleDataStruct1,
                currentValSetArray,
                sigStructArray1,
                withdrawId1,
            )
            
            if (BigInt(depositAmount0) > BigInt(withdrawLimitBefore0)) {
                expectedBal0 += BigInt(withdrawLimitBefore0)
                expTokensToClaim0 += BigInt(depositAmount0) - BigInt(withdrawLimitBefore0)
                expectedBalBridge -= BigInt(withdrawLimitBefore0)
            } else {
                expectedBal0 += BigInt(depositAmount0)
                expectedBalBridge -= BigInt(depositAmount0)
            }
            if (depositAmount1 > withdrawLimitBefore1) {
                expectedBal1 += BigInt(withdrawLimitBefore1)
                expTokensToClaim1 += BigInt(depositAmount1) - BigInt(withdrawLimitBefore1)
                expectedBalBridge -= BigInt(withdrawLimitBefore1)
            } else {
                expectedBal1 += BigInt(depositAmount1)
                expectedBalBridge -= BigInt(depositAmount1)
            }
            userBal0 = await token.balanceOf(accounts[0].address)
            userBal1 = await token.balanceOf(accounts[1].address)
            bridgeBal = await token.balanceOf(await tbridge.address)
            tokensToClaim0 = await tbridge.tokensToClaim(accounts[0].address)
            tokensToClaim1 = await tbridge.tokensToClaim(accounts[1].address)

            assert(BigInt(userBal0) == expectedBal0, "user0 bal should be correct")
            assert(BigInt(userBal1) == expectedBal1, "user1 bal should be correct")
            assert(BigInt(bridgeBal) == expectedBalBridge, "bridge bal should be correct")
            assert(BigInt(tokensToClaim0) == expTokensToClaim0, "tokensToClaim0 should be correct")
            assert(BigInt(tokensToClaim1) == expTokensToClaim1, "tokensToClaim1 should be correct")
        }

        await h.expectThrow(tbridge.claimExtraWithdraw(accounts[0].address))
        await h.expectThrow(tbridge.claimExtraWithdraw(accounts[1].address))
        await h.advanceTime(43200)

        while (tokensToClaim0 > 0) {
            await tbridge.claimExtraWithdraw(accounts[0].address)
            tokensToClaim0 = await tbridge.tokensToClaim(accounts[0].address)
            await h.advanceTime(43200)
        }
        while (tokensToClaim1 > 0) {
            await tbridge.claimExtraWithdraw(accounts[1].address)
            tokensToClaim1 = await tbridge.tokensToClaim(accounts[1].address)
            await h.advanceTime(43200)
        }

        userBal0 = await token.balanceOf(accounts[0].address)
        userBal1 = await token.balanceOf(accounts[1].address)
        bridgeBal = await token.balanceOf(await tbridge.address)
        assert(BigInt(userBal0) == initUserBal0, "user0 bal should be correct")
        assert(BigInt(userBal1) == initUserBal1, "user1 bal should be correct")
        assert(BigInt(bridgeBal) == BigInt(INITIAL_LAYER_TOKEN_SUPPLY), "bridge bal should be correct")
    }, 300000)
    
    it("mint rate rounding error", async function() {
        // test worst case rounding error of mintToOracle every block
        // and no rounding errors from EVM->Layer decimal conversion.
        // layer mintToOracle calculation:
        // uint256 _releasedAmount = (146.94 ether *
        //     (block.timestamp - uints[keccak256("_LAST_RELEASE_TIME_DAO")])) /
        //     86400;

        // minting rate per day on layer
        const amtPerDayLayer = BigInt(h.toWei("146.94"));
        
        // minting rate per block on evm (12 sec blocks)
        const blocksPerDayEVM = BigInt(86400) / BigInt(12);
        const amtPerBlockEVM = amtPerDayLayer / blocksPerDayEVM;

        // total minted amount in a year on evm
        const amtPerYearEVM = amtPerBlockEVM * blocksPerDayEVM * BigInt(365);
        
        // total minted amount in a year on layer
        const amtPerYearLayer = amtPerDayLayer * BigInt(365);

        // difference in minted amounts between evm and layer in a year
        const diffPerYear = amtPerYearLayer - amtPerYearEVM;

        // Assert that difference is less than a small threshold (e.g., 1e12 wei)
        const threshold = BigInt(1e12);
        assert(diffPerYear < threshold, "Difference in minting rates between EVM and Layer should be less than threshold");
    })

    it("mintToOracle on deposit", async function() {
        const bridge2 = await ethers.deployContract("TestTokenBridge", [token.address,blobstream.address, oldOracle.address])
        await token.setOracleMintRecipient(bridge2.address)
        deployTime = await token.lastReleaseTimeDao()
        bridgeBal = await token.balanceOf(await bridge2.address)
        assert(BigInt(bridgeBal) == BigInt(0), "bridge bal should be correct")
        await token.approve(bridge2.address, h.toWei("10000"))
        await h.advanceTime(86400)
        tip = h.toWei("0")
        await bridge2.depositToLayer(h.toWei("20"), tip, LAYER_RECIPIENT)
        blocky1 = await h.getBlock()
        // formula from Tellor360:
        // uint256 _releasedAmount = (146.94 ether *
        // (block.timestamp - uints[keccak256("_LAST_RELEASE_TIME_DAO")])) /
        // 86400;
        expectedBal = (BigInt(h.toWei("146.94")) * (BigInt(blocky1.timestamp) - BigInt(deployTime))) / BigInt(86400) + BigInt(h.toWei("20"))
        bridgeBal = await token.balanceOf(await bridge2.address)
        assert(BigInt(bridgeBal) == BigInt(expectedBal), "bridge bal should be correct")
        await h.advanceTime(86400)
        await bridge2.depositToLayer(h.toWei("20"), tip, LAYER_RECIPIENT)
        expectedBal = expectedBal + BigInt(h.toWei("20"))
        bridgeBal = await token.balanceOf(await bridge2.address)
        assert(BigInt(bridgeBal) == BigInt(expectedBal), "bridge bal should be correct")
    })
})
