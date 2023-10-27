// SPDX-License-Identifier: UNLICENSED
pragma solidity 0.8.3;

import "forge-std/Test.sol";
import "../src/bridge/LayerLightClientBridge.sol";

contract CounterTest is Test {
    LayerLightClientBridge public bridge;

    // address[] validatorAddr = [0x35C132b955bF9e284005b81ACDb3D2FE5096c714];
    address[] validatorAddr = [0x0E3FbA8eAcE8fE7393D20597c3bB3e9A03d68900];
    uint256[] validatorPower = [100];
    string chainIdString = "luqchain";

    LayerLightClientBridge.MultistoreData multistore = LayerLightClientBridge.MultistoreData({
        oracleIAVLStateHash: 0xc75128a5c12b64d52a625cb6e99e6dd850e738b87bbff5f0c5aa5836b2b44a9b,
        paramsStoreMerkleHash: 0x7d23c97e19775b2bcac154d730753f96bced201a6175a3df258750c6aee591d4,
        slashingToStakingStoresMerkleHash: 0xf03e9a3a8125b3030d3da809a5065fb5f4fb91ae04b45c455218f4844614fc48,
        govToMintStoresMerkleHash: 0x7dcbc829edb61ac95ec85b22e5ac94e5fbcf67b85db27c4e58b441e811e32b25,
        authToFeegrantStoresMerkleHash: 0xf5a5cbb7216c810fa4914be38e7737aea2aebf6d5c4fc3a346887269a195ac4e,
        transferToUpgradeStoresMerkleHash: 0x0d1926797c305464a4fa3bccf75049b85606d5f3651f6e6b93d179d222cee3c9
    });

    LayerLightClientBridge.BlockHeaderMerkleParts merkleParts = LayerLightClientBridge.BlockHeaderMerkleParts({
        versionAndChainIdHash: 0x30a56e8396af0bf8abfe2b36fbc3b9cce5d179f94bd63ec6c906ed9eed360676,
        height: 26756,
        timeSecond: 1695074461,
        timeNanoSecondFraction: 173012000,
        lastBlockIdAndOther: 0x86b84be5c4f0633b64c063fe3387def7aefaab4c40389e3c324f90c598c2d43c,
        nextValidatorHashAndConsensusHash: 0xf152e5b7306c900ee5af119ba737d7f9cbd354e3cbd232447d22bb091b8ae7d6,
        lastResultsHash: 0x9fb9c7533caf1d218da3af6d277f6b101c42e3c3b75d784242da663604dd53c2,
        evidenceAndProposerHash: 0x2be393c8955c7553052452cebe741356dc47ddefdf699ddf01b121859796b787
    });

    LayerLightClientBridge.CommonEncodedVotePartData votePart = LayerLightClientBridge.CommonEncodedVotePartData({
        signedDataPrefix: hex"080211846800000000000022480A20",
        signedDataSuffix: hex"122408011220418E093FE4C97A9B3F207FECF974AB732B8FB7B037317C28DCFE2C339BFD8BA6"
    });

    LayerLightClientBridge.TMSignatureData signatureData = LayerLightClientBridge.TMSignatureData({
        r: 0xf3226fb35be59364391f539eb1ffadc9ae8aee3ad5d98557030a1d3de3cf22e4,
        s: 0x180b53c7729c3d6caae03bb1a387bbb8f6b6258231452cbf43a4ba7efbc72427,
        v: 28,
        encodedTimestamp: hex"089E91A3A80610B0B69062"
    });

    LayerLightClientBridge.TMSignatureData[] sigDataArray;

    function setUp() public {
        bridge = new LayerLightClientBridge();
        bridge.testSetNumber(0);

        bridge.init(validatorAddr, validatorPower, chainIdString);
        sigDataArray.push(signatureData);
    }

    function testSetNumber(uint256 x) public {
        bridge.testSetNumber(x);
        assertEq(bridge.testNumber(), x);
    }

    function testVerifyBlockHeader() public {
        bool result = bridge.verifyBlockHeader(multistore, merkleParts, votePart, sigDataArray);
        assertEq(result, true);
    }

    // function testVerifyBlockHeader() public {
    //     address result = bridge.verifyBlockHeader(multistore, merkleParts, votePart, sigDataArray);
    //     assertEq(result, validatorAddr[0]);
    // }

    function testGetAppHash() public {
        bytes32 _appHash = bridge.getAppHash(multistore);
        assertEq(_appHash, 0xfadf5693808d1fd6f1c7acb3c4ebeeaac51e17c5a76edae581e63377efde6f1a);
    }

    function testGetBlockHeader() public {
        bytes32 _appHash = bridge.getAppHash(multistore);
        bytes32 _blockHeader = bridge.getBlockHeader(merkleParts, _appHash);
        assertEq(_blockHeader, 0x81a518dd8aa830de59dc35357638ebdc21c455230da624476709cf52410a36cc);
    }

   
}
