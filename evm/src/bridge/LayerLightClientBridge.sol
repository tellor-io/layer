// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./ProtobufLib.sol";

contract LayerLightClientBridge {
    bytes public encodedChainId;
    uint256 public totalValidatorPower;
    uint256 public unbondingPeriod;
    uint256 public latestBlockHeight;

    mapping (address => uint256) public validatorPowers;
    mapping (uint256 => BlockDetail) public blockDetails; // mapping block height to block details

    struct BlockDetail {
        bytes32 oracleHash;
        uint64 timeSecond;
        uint32 timeNanoSecondFraction;
    }

    enum ResolveStatus {
        RESOLVE_STATUS_OPEN_UNSPECIFIED,
        RESOLVE_STATUS_SUCCESS,
        RESOLVE_STATUS_FAILURE,
        RESOLVE_STATUS_EXPIRED
    }

    struct Value {
        string clientID;
        uint64 oracleScriptID;
        bytes params;
        uint64 askCount;
        uint64 minCount;
        uint64 requestID;
        uint64 ansCount;
        uint64 requestTime;
        uint64 resolveTime;
        ResolveStatus resolveStatus;
        bytes result;
    }

    struct MultistoreData {
        bytes32 oracleIAVLStateHash; // [C]
        bytes32 paramsStoreMerkleHash; // [D]
        bytes32 slashingToStakingStoresMerkleHash; // [I7]
        bytes32 govToMintStoresMerkleHash; // [I10]
        bytes32 authToFeegrantStoresMerkleHash; // [I12]
        bytes32 transferToUpgradeStoresMerkleHash; // [I15]
    }

    struct BlockHeaderMerkleParts {
        bytes32 versionAndChainIdHash; // [1A]
        uint64 height; // [2]
        uint64 timeSecond; // [3]
        uint32 timeNanoSecondFraction; // between 0 to 10^9 [3]
        bytes32 lastBlockIdAndOther; // [2B]
        bytes32 nextValidatorHashAndConsensusHash; // [1E]
        bytes32 lastResultsHash; // [B]
        bytes32 evidenceAndProposerHash; // [2D]
    }

    struct CommonEncodedVotePartData {
        bytes signedDataPrefix;
        bytes signedDataSuffix;
    }

    struct TMSignatureData {
        bytes32 r;
        bytes32 s;
        uint8 v;
        bytes encodedTimestamp;
    }

    struct IAVLMerklePath {
        bool isDataOnRight;
        uint8 subtreeHeight;
        uint256 subtreeSize;
        uint256 subtreeVersion;
        bytes32 siblingHash;
    }

    constructor(uint256 _unbondingPeriod) {
        unbondingPeriod = _unbondingPeriod;
    }

    function init(
        address[] calldata _validators, 
        uint256[] calldata _powers, 
        string calldata _encodedChainID
    ) external {
        require(encodedChainId.length == 0, "Already initialized");
        // set initial state, validator addresses and weights
        for (uint256 _i=0; _i<_validators.length; _i++) {
            validatorPowers[_validators[_i]] = _powers[_i];
            totalValidatorPower += _powers[_i];
        }
        encodedChainId = abi.encode(_encodedChainID);
    }

    function relayBlock(
        MultistoreData calldata _multiStore,
        BlockHeaderMerkleParts calldata _merkleParts,
        CommonEncodedVotePartData calldata _commonEncodedVotePart,
        TMSignatureData[] calldata _signatures
    ) public {
        // if same data for block height already exists, return
        // require last block header less than unbonding period old
        require(_merkleParts.height > latestBlockHeight, "Block already relayed");
        require(verifyBlockHeader(_multiStore, _merkleParts, _commonEncodedVotePart, _signatures), "Invalid block header");
        blockDetails[_merkleParts.height] = BlockDetail({
            oracleHash: _multiStore.oracleIAVLStateHash,
            timeSecond: _merkleParts.timeSecond,
            timeNanoSecondFraction: _merkleParts.timeNanoSecondFraction
        });
        latestBlockHeight = _merkleParts.height;
    }

    
    

    function verifyBlockHeader(
        MultistoreData memory _multiStore,
        BlockHeaderMerkleParts memory _merkleParts,
        CommonEncodedVotePartData memory _commonEncodedVotePart,
        TMSignatureData[] memory _signatures
    ) public view returns(bool) {
        // input block header parts and signatures
        // validate block header
        bytes32 _appHash = getAppHash(_multiStore);
        bytes32 _blockHeader = getBlockHeader(_merkleParts, _appHash);
        bytes memory _commonEncodedPart = checkPartsAndEncodedCommonParts(_commonEncodedVotePart, _blockHeader);

        // load encoded chain id
        bytes memory _encodedChainId = encodedChainId;

        // Tally the total voting power behind the signatures
        address _lastSigner = address(0);
        uint256 _sumVotingPower = 0;
        for (uint256 _i = 0; _i < _signatures.length; _i++) {
            address _signer = checkTimeAndRecoverSigner(_signatures[_i], _commonEncodedPart, _encodedChainId);
            require(_signer > _lastSigner, "Invalid signer order");
            if (validatorPowers[_signer] > 0) {
                _sumVotingPower += validatorPowers[_signer];
            }
            _lastSigner = _signer;
        }
        // Verify that sufficient voting power signed the block
        return _sumVotingPower * 3 >= totalValidatorPower * 2;
    }

    function verifyProof(
        bytes32 _rootHash,
        uint256 _version,
        bytes memory _key,
        bytes32 _dataHash,
        IAVLMerklePath[] memory _merklePaths
    ) internal view returns(bool) {
        bytes memory _encodedVersion = _encodeVarintUnsigned(_version);

        bytes32 _currentMerkleHash = sha256(
            abi.encodePacked(
                uint8(0), // Height of tree (only leaf node) is 0 (signed-varint encode)
                uint8(2), // Size of subtree is 1 (signed-varint encode)
                _encodedVersion,
                uint8(_key.length), // Size of data key
                _key,
                uint8(32), // Size of data hash
                _dataHash
            )
        );

        for (uint256 _i = 0; _i < _merklePaths.length; _i++) {
            _currentMerkleHash = getParentHash(_merklePaths[_i], _currentMerkleHash);
        }

        // verify that calculated merkle root hash equals expected root hash
        return _currentMerkleHash == _rootHash;
    }

    function verifyOracleData(
        uint256 _blockHeight,
        Value calldata _value,
        uint256 _oracleHeight,
        IAVLMerklePath[] calldata _merklePaths
    ) public view returns(Value memory) {
        bytes32 _oracleStateRoot = blockDetails[_blockHeight].oracleHash;
        require(
            _oracleStateRoot != bytes32(0), "No root at this height"
        );

        bytes32 _dataHash = sha256(protoEncode(_value));

        require(
            verifyProof(
                _oracleStateRoot,
                _oracleHeight,
                abi.encodePacked(uint8(255), _value.requestID),
                _dataHash,
                _merklePaths
            ),
            "Invalid proof"
        );

        return _value;
    }

    // ************************************************************ 
    // * internal pure helper functions
    // ************************************************************

    function getBlockHeader(BlockHeaderMerkleParts memory _merkleParts, bytes32 _appHash) public pure returns(bytes32) {
        bytes32 _blockHeader = _merkleInnerHash( // [BlockHeader]
                _merkleInnerHash( // [3A]
                    _merkleInnerHash( // [2A]
                        _merkleParts.versionAndChainIdHash, // [1A]
                        _merkleInnerHash( // [1B]
                            _merkleLeafHash( // [2]
                                abi.encodePacked(
                                    uint8(8),
                                    _encodeVarintUnsigned(_merkleParts.height)
                                )
                            ),
                            _merkleLeafHash( // [3]
                                _encodeTime(
                                    _merkleParts.timeSecond,
                                    _merkleParts.timeNanoSecondFraction
                                )
                            )
                        )
                    ),
                    _merkleParts.lastBlockIdAndOther // [2B]
                ),
                _merkleInnerHash( // [3B]
                    _merkleInnerHash( // [2C]
                        _merkleParts.nextValidatorHashAndConsensusHash, // [1E]
                        _merkleInnerHash( // [1F]
                            _merkleLeafHash( // [A]
                                abi.encodePacked(uint8(10), uint8(32), _appHash)
                            ),
                            _merkleParts.lastResultsHash // [B]
                        )
                    ),
                    _merkleParts.evidenceAndProposerHash // [2D]
                )
            );
        return _blockHeader;
    } 

/*
                                       _______________________[AppHash]______________
                                      /                                              \
                   _________________[I19]_________________                         ____[I20*]____
                  /                                       \	                      /              \
         _______[I15*]______                       _______[I16]_________        [GHIJ]           [KLMN]
        /                   \                     /                     \
    __[I8]__              __[I9]__            __[I10*]__             __[I11]__
   /        \            /         \         /          \           /         \
 [I0]      [I1]       [I2]        [I3]     [I4]         [I5]      [I6*]       [I7]
 /  \     /   \      /   \      /    \    /    \        /  \      /   \      /   \
[0] [1]  [2]   [3]  [4]  [5]  [6]    [7] [8]   [9]     [A] [B]   [C]  [D]  [E*]  [F*]
Right[F], Left[I6], Left[I10], Left[I15], Right[I20]
[0] - acc (auth)     [1] - authz     [2] - bank      [3] - capability [4] - consensus [5] - crisis
[6] - distr          [7] - evidence  [8] - feegrant  [9] - gov        [A] - group     [B] - ibc
[C] - icacontroller  [D] - icahost   [E] - luqchain  [F] - mint       [G] - params    [H] - slashing
[I] - staking        [J] - transfer  [K] - upgrade   [L] - vesting
*/
// struct MultistoreData {
//         bytes32 luqchainIavlStateHash; // [E]
//         bytes32 mintStoreMerkleHash; // [F]
//         bytes32 icacontrollerToIcahostMerkleHash; // [I6]
//         bytes32 feegrantToIbcMerkleHash; // [I10]
//         bytes32 accToEvidenceMerkleHash; // [I15]
//         bytes32 paramsToVestingMerkleHash; // [I20]
// }
    function getAppHash2(MultistoreData memory _store) public pure returns(bytes32) {
        bytes32 _appHash = _merkleInnerHash(
            _merkleInnerHash(
                _store.authToFeegrantStoresMerkleHash,
                _merkleInnerHash( 
                    _store.govToMintStoresMerkleHash,
                    _merkleInnerHash(
                        _store.slashingToStakingStoresMerkleHash, // [I6]
                        _merkleInnerHash(
                            _merkleLeafHash( // [E]
                                abi.encodePacked(
                                        hex"066c7571636861696e20", // oracle prefix (uint8(6) + "oracle" + uint8(32)) NOTE: Switch to Tellor Layer oracle prefix
                                        sha256(                    // using (uint8(6) + "luqchain" + uint8(32)) /// oracle: 0x6f7261636c65 ; luqchain: 0x6c7571636861696e
                                            abi.encodePacked(
                                                _store.oracleIAVLStateHash
                                            )
                                        )
                                    )
                            ),
                            _store.paramsStoreMerkleHash // [F]
                        )
                    )
                )
            ),
            _store.transferToUpgradeStoresMerkleHash // [I20]
        );
        return _appHash;
    }

    function getAppHash(MultistoreData memory _store) public pure returns(bytes32) {
        // oracle prefix (uint8(6) + "oracle" + uint8(32)) NOTE: Switch to Tellor Layer oracle prefix
        // using (uint8(8) + "luqchain" + uint8(32)) /// oracle: 0x6f7261636c65 ; luqchain: 0x6c7571636861696e
        bytes32 _ELeaf = _merkleLeafHash(abi.encodePacked(hex"086c7571636861696e20", sha256(abi.encodePacked(_store.oracleIAVLStateHash))));
        bytes32 _I7 = _merkleInnerHash(_ELeaf, _store.paramsStoreMerkleHash);
        bytes32 _I11 = _merkleInnerHash(_store.slashingToStakingStoresMerkleHash, _I7);
        bytes32 _I16 = _merkleInnerHash(_store.govToMintStoresMerkleHash, _I11); 
        bytes32 _I19 = _merkleInnerHash(_store.authToFeegrantStoresMerkleHash, _I16);
        bytes32 _appHash = _merkleInnerHash(_I19, _store.transferToUpgradeStoresMerkleHash);
        return _appHash;
    }

    function _merkleInnerHash(bytes32 _left, bytes32 _right) internal pure returns(bytes32) {
        return sha256(abi.encodePacked(uint8(1), _left, _right));
    }

    function _merkleLeafHash(bytes memory _value) internal pure returns(bytes32) {
        return sha256(abi.encodePacked(uint8(0), _value));
    }

    /// @dev Returns the upper Merkle hash given a proof component and hash of data subtree.
    /// @param dataSubtreeHash The hash of data subtree up until this point.
    function getParentHash(IAVLMerklePath memory _merklePath, bytes32 dataSubtreeHash)
        internal
        pure
        returns (bytes32)
    {
        (bytes32 leftSubtree, bytes32 rightSubtree) =
            _merklePath.isDataOnRight ? (_merklePath.siblingHash, dataSubtreeHash) : (dataSubtreeHash, _merklePath.siblingHash);
        return
            sha256(
                abi.encodePacked(
                    _merklePath.subtreeHeight << 1, // Tendermint signed-int8 encoding requires multiplying by 2
                    _encodeVarintSigned(_merklePath.subtreeSize),
                    _encodeVarintSigned(_merklePath.subtreeVersion),
                    uint8(32), // Size of left subtree hash
                    leftSubtree,
                    uint8(32), // Size of right subtree hash
                    rightSubtree
                )
            );
    }

    /// @dev Returns the encoded bytes using unsigned varint encoding of the given input.
    function _encodeVarintUnsigned(uint256 _value)
        internal
        pure
        returns (bytes memory)
    {
        // Computes the size of the encoded value.
        uint256 tempValue = _value;
        uint256 size = 0;
        while (tempValue > 0) {
            ++size;
            tempValue >>= 7;
        }
        // Allocates the memory buffer and fills in the encoded value.
        bytes memory result = new bytes(size);
        tempValue = _value;
        for (uint256 idx = 0; idx < size; ++idx) {
            result[idx] = bytes1(uint8(128) | uint8(tempValue & 127));
            tempValue >>= 7;
        }
        result[size - 1] &= bytes1(uint8(127)); // Drop the first bit of the last byte.
        return result;
    }

    /// @dev Returns the encoded bytes using signed varint encoding of the given input.
    function _encodeVarintSigned(uint256 value)
        internal
        pure
        returns (bytes memory)
    {
        return _encodeVarintUnsigned(value * 2);
    }

    /// @dev Returns the encoded bytes following how tendermint encodes time.
    function _encodeTime(uint64 second, uint32 nanoSecond)
        internal
        pure
        returns (bytes memory)
    {
        bytes memory result =
            abi.encodePacked(hex"08", _encodeVarintUnsigned(uint256(second)));
        if (nanoSecond > 0) {
            result = abi.encodePacked(
                result,
                hex"10",
                _encodeVarintUnsigned(uint256(nanoSecond))
            );
        }
        return result;
    }

    /// @dev Returns the address that signed on the given block hash.
    /// @param _blockHash The block hash that the validator signed data on.
    function checkPartsAndEncodedCommonParts(CommonEncodedVotePartData memory _votePart, bytes32 _blockHash)
        public
        pure
        returns (bytes memory)
    {
        // We need to limit the possible size of the prefix and suffix to ensure only one possible block hash.

        // There are only two possible prefix sizes.
        // 1. If Round == 0, the prefix size should be 15 because the encoded Round was cut off.
        // 2. If not then the prefix size should be 24 (15 + 9).
        require(
            _votePart.signedDataPrefix.length == 15 || _votePart.signedDataPrefix.length == 24,
            "CommonEncodedVotePart: Invalid prefix's size"
        );

        // The suffix is encoded of a CanonicalPartSetHeader, which has a fixed size in practical.
        // There are two reasons why.
        // 1. The maximum value of CanonicalPartSetHeader.Total is 48 (3145728 / 65536) because Band's MaxBlockSizeBytes
        // is 3145728 bytes, and the max BlockPartSizeBytes's size is 65536 bytes.
        // 2. The CanonicalPartSetHeader.Hash's size is fixed (32 bytes) because it is a product of SHA256.
        // Therefore, the overall size is fixed.
        require(_votePart.signedDataSuffix.length == 38, "CommonEncodedVotePart: Invalid suffix's size");

        return abi.encodePacked(
            _votePart.signedDataPrefix,
            _blockHash,
            _votePart.signedDataSuffix
        );
    }

    /// @dev Returns the address that signed on the given encoded canonical vote message on Cosmos.
    /// @param _commonEncodedPart The first common part of the encoded canonical vote.
    /// @param _encodedChainID The last part of the encoded canonical vote.
    function checkTimeAndRecoverSigner(TMSignatureData memory _sigData, bytes memory _commonEncodedPart, bytes memory _encodedChainID)
        public
        pure
        returns (address)
    {
        // We need to limit the possible size of the encodedCanonicalVote to ensure only one possible block hash.
        // The size of the encodedTimestamp will be between 6 and 12 according to the following two constraints.
        // 1. The size of an encoded Unix's second is 6 bytes until over a thousand years in the future.
        // 2. The NanoSecond size can vary from 0 to 6 bytes.
        // Therefore, 6 + 0 <= the size <= 6 + 6.
        require(
            6 <= _sigData.encodedTimestamp.length && _sigData.encodedTimestamp.length <= 12,
            "TMSignature: Invalid timestamp's size"
        );
        bytes memory encodedCanonicalVote = abi.encodePacked(
            _commonEncodedPart,
            uint8(42),
            uint8(_sigData.encodedTimestamp.length),
            _sigData.encodedTimestamp,
            _encodedChainID
        );
        return
            ecrecover(
                sha256(abi.encodePacked(uint8(encodedCanonicalVote.length), encodedCanonicalVote)),
                _sigData.v,
                _sigData.r,   
                _sigData.s
            );
    }

    function protoEncode(Value memory instance)
        internal
        pure
        returns (bytes memory)
    {
        bytes memory finalEncoded;

        // Omit encoding clientID if default value
        if (bytes(instance.clientID).length > 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(
                    1,
                    uint64(ProtobufLib.WireType.LengthDelimited)
                ),
                ProtobufLib.encode_uint64(
                    uint64(bytes(instance.clientID).length)
                ),
                bytes(instance.clientID)
            );
        }

        // Omit encoding oracleScriptID if default value
        if (uint64(instance.oracleScriptID) != 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(2, uint64(ProtobufLib.WireType.Varint)),
                ProtobufLib.encode_uint64(instance.oracleScriptID)
            );
        }

        // Omit encoding params if default value
        if (bytes(instance.params).length > 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(
                    3,
                    uint64(ProtobufLib.WireType.LengthDelimited)
                ),
                ProtobufLib.encode_uint64(
                    uint64(bytes(instance.params).length)
                ),
                bytes(instance.params)
            );
        }

        // Omit encoding askCount if default value
        if (uint64(instance.askCount) != 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(4, uint64(ProtobufLib.WireType.Varint)),
                ProtobufLib.encode_uint64(instance.askCount)
            );
        }

        // Omit encoding minCount if default value
        if (uint64(instance.minCount) != 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(5, uint64(ProtobufLib.WireType.Varint)),
                ProtobufLib.encode_uint64(instance.minCount)
            );
        }

        // Omit encoding requestID if default value
        if (uint64(instance.requestID) != 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(6, uint64(ProtobufLib.WireType.Varint)),
                ProtobufLib.encode_uint64(instance.requestID)
            );
        }

        // Omit encoding ansCount if default value
        if (uint64(instance.ansCount) != 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(7, uint64(ProtobufLib.WireType.Varint)),
                ProtobufLib.encode_uint64(instance.ansCount)
            );
        }

        // Omit encoding requestTime if default value
        if (uint64(instance.requestTime) != 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(8, uint64(ProtobufLib.WireType.Varint)),
                ProtobufLib.encode_uint64(instance.requestTime)
            );
        }

        // Omit encoding resolveTime if default value
        if (uint64(instance.resolveTime) != 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(9, uint64(ProtobufLib.WireType.Varint)),
                ProtobufLib.encode_uint64(instance.resolveTime)
            );
        }

        // Omit encoding resolveStatus if default value
        if (uint64(instance.resolveStatus) != 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(10, uint64(ProtobufLib.WireType.Varint)),
                ProtobufLib.encode_int32(int32(uint32(instance.resolveStatus)))
            );
        }

        // Omit encoding result if default value
        if (bytes(instance.result).length > 0) {
            finalEncoded = abi.encodePacked(
                finalEncoded,
                ProtobufLib.encode_key(
                    11,
                    uint64(ProtobufLib.WireType.LengthDelimited)
                ),
                ProtobufLib.encode_uint64(
                    uint64(bytes(instance.result).length)
                ),
                bytes(instance.result)
            );
        }

        return finalEncoded;
    }
}