"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BlobNetworkTransactionWrapper = exports.KZGProofType = exports.KZGCommitmentType = exports.SignedBlobTransactionType = exports.ECDSASignatureType = exports.BlobTransactionType = exports.AccessTupleType = exports.AddressType = exports.isAccessList = exports.isAccessListBuffer = exports.Capability = void 0;
const ssz_1 = require("@chainsafe/ssz");
const constants_1 = require("./constants");
const Bytes20 = new ssz_1.ByteVectorType(20);
const Bytes32 = new ssz_1.ByteVectorType(32);
const Bytes48 = new ssz_1.ByteVectorType(48);
const Uint64 = new ssz_1.UintBigintType(8);
const Uint256 = new ssz_1.UintBigintType(32);
/**
 * Can be used in conjunction with {@link Transaction.supports}
 * to query on tx capabilities
 */
var Capability;
(function (Capability) {
    /**
     * Tx supports EIP-155 replay protection
     * See: [155](https://eips.ethereum.org/EIPS/eip-155) Replay Attack Protection EIP
     */
    Capability[Capability["EIP155ReplayProtection"] = 155] = "EIP155ReplayProtection";
    /**
     * Tx supports EIP-1559 gas fee market mechanism
     * See: [1559](https://eips.ethereum.org/EIPS/eip-1559) Fee Market EIP
     */
    Capability[Capability["EIP1559FeeMarket"] = 1559] = "EIP1559FeeMarket";
    /**
     * Tx is a typed transaction as defined in EIP-2718
     * See: [2718](https://eips.ethereum.org/EIPS/eip-2718) Transaction Type EIP
     */
    Capability[Capability["EIP2718TypedTransaction"] = 2718] = "EIP2718TypedTransaction";
    /**
     * Tx supports access list generation as defined in EIP-2930
     * See: [2930](https://eips.ethereum.org/EIPS/eip-2930) Access Lists EIP
     */
    Capability[Capability["EIP2930AccessLists"] = 2930] = "EIP2930AccessLists";
})(Capability = exports.Capability || (exports.Capability = {}));
function isAccessListBuffer(input) {
    if (input.length === 0) {
        return true;
    }
    const firstItem = input[0];
    if (Array.isArray(firstItem)) {
        return true;
    }
    return false;
}
exports.isAccessListBuffer = isAccessListBuffer;
function isAccessList(input) {
    return !isAccessListBuffer(input); // This is exactly the same method, except the output is negated.
}
exports.isAccessList = isAccessList;
/** EIP4844 types */
exports.AddressType = Bytes20; // SSZ encoded address
// SSZ encoded container for address and storage keys
exports.AccessTupleType = new ssz_1.ContainerType({
    address: exports.AddressType,
    storageKeys: new ssz_1.ListCompositeType(Bytes32, constants_1.MAX_VERSIONED_HASHES_LIST_SIZE),
});
// SSZ encoded blob transaction
exports.BlobTransactionType = new ssz_1.ContainerType({
    chainId: Uint256,
    nonce: Uint64,
    maxPriorityFeePerGas: Uint256,
    maxFeePerGas: Uint256,
    gas: Uint64,
    to: new ssz_1.UnionType([new ssz_1.NoneType(), exports.AddressType]),
    value: Uint256,
    data: new ssz_1.ByteListType(constants_1.MAX_CALLDATA_SIZE),
    accessList: new ssz_1.ListCompositeType(exports.AccessTupleType, constants_1.MAX_ACCESS_LIST_SIZE),
    maxFeePerDataGas: Uint256,
    blobVersionedHashes: new ssz_1.ListCompositeType(Bytes32, constants_1.MAX_VERSIONED_HASHES_LIST_SIZE),
});
// SSZ encoded ECDSA Signature
exports.ECDSASignatureType = new ssz_1.ContainerType({
    yParity: new ssz_1.BooleanType(),
    r: Uint256,
    s: Uint256,
});
// SSZ encoded signed blob transaction
exports.SignedBlobTransactionType = new ssz_1.ContainerType({
    message: exports.BlobTransactionType,
    signature: exports.ECDSASignatureType,
});
// SSZ encoded KZG Commitment/Proof (48 bytes)
exports.KZGCommitmentType = Bytes48;
exports.KZGProofType = exports.KZGCommitmentType;
// SSZ encoded blob network transaction wrapper
exports.BlobNetworkTransactionWrapper = new ssz_1.ContainerType({
    tx: exports.SignedBlobTransactionType,
    blobKzgs: new ssz_1.ListCompositeType(exports.KZGCommitmentType, constants_1.MAX_TX_WRAP_KZG_COMMITMENTS),
    blobs: new ssz_1.ListCompositeType(new ssz_1.ByteVectorType(constants_1.FIELD_ELEMENTS_PER_BLOB * constants_1.BYTES_PER_FIELD_ELEMENT), constants_1.LIMIT_BLOBS_PER_TX),
    kzgAggregatedProof: exports.KZGProofType,
});
//# sourceMappingURL=types.js.map