"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BlobEIP4844Transaction = void 0;
const ssz_1 = require("@chainsafe/ssz");
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const keccak_1 = require("ethereum-cryptography/keccak");
const baseTransaction_1 = require("./baseTransaction");
const constants_1 = require("./constants");
const kzg_1 = require("./kzg/kzg");
const types_1 = require("./types");
const util_1 = require("./util");
const blobHelpers_1 = require("./utils/blobHelpers");
const TRANSACTION_TYPE = 0x05;
const TRANSACTION_TYPE_BUFFER = Buffer.from(TRANSACTION_TYPE.toString(16).padStart(2, '0'), 'hex');
const validateBlobTransactionNetworkWrapper = (versionedHashes, blobs, commitments, kzgProof, version) => {
    if (!(versionedHashes.length === blobs.length && blobs.length === commitments.length)) {
        throw new Error('Number of versionedHashes, blobs, and commitments not all equal');
    }
    try {
        kzg_1.kzg.verifyAggregateKzgProof(blobs, commitments, kzgProof);
    }
    catch (e) {
        throw new Error('KZG proof cannot be verified from blobs/commitments');
    }
    for (let x = 0; x < versionedHashes.length; x++) {
        const computedVersionedHash = (0, blobHelpers_1.computeVersionedHash)(commitments[x], version);
        if (!(0, ssz_1.byteArrayEquals)(computedVersionedHash, versionedHashes[x])) {
            throw new Error(`commitment for blob at index ${x} does not match versionedHash`);
        }
    }
};
/**
 * Typed transaction with a new gas fee market mechanism for transactions that include "blobs" of data
 *
 * - TransactionType: 5
 * - EIP: [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844)
 */
class BlobEIP4844Transaction extends baseTransaction_1.BaseTransaction {
    /**
     * This constructor takes the values, validates them, assigns them and freezes the object.
     *
     * It is not recommended to use this constructor directly. Instead use
     * the static constructors or factory methods to assist in creating a Transaction object from
     * varying data types.
     */
    constructor(txData, opts = {}) {
        super({ ...txData, type: TRANSACTION_TYPE }, opts);
        const { chainId, accessList, maxFeePerGas, maxPriorityFeePerGas, maxFeePerDataGas } = txData;
        this.common = this._getCommon(opts.common, chainId);
        this.chainId = this.common.chainId();
        if (this.common.isActivatedEIP(1559) === false) {
            throw new Error('EIP-1559 not enabled on Common');
        }
        if (this.common.isActivatedEIP(4844) === false) {
            throw new Error('EIP-4844 not enabled on Common');
        }
        this.activeCapabilities = this.activeCapabilities.concat([1559, 2718, 2930]);
        // Populate the access list fields
        const accessListData = util_1.AccessLists.getAccessListData(accessList ?? []);
        this.accessList = accessListData.accessList;
        this.AccessListJSON = accessListData.AccessListJSON;
        // Verify the access list format.
        util_1.AccessLists.verifyAccessList(this.accessList);
        this.maxFeePerGas = (0, ethereumjs_util_1.bufferToBigInt)((0, ethereumjs_util_1.toBuffer)(maxFeePerGas === '' ? '0x' : maxFeePerGas));
        this.maxPriorityFeePerGas = (0, ethereumjs_util_1.bufferToBigInt)((0, ethereumjs_util_1.toBuffer)(maxPriorityFeePerGas === '' ? '0x' : maxPriorityFeePerGas));
        this._validateCannotExceedMaxInteger({
            maxFeePerGas: this.maxFeePerGas,
            maxPriorityFeePerGas: this.maxPriorityFeePerGas,
        });
        baseTransaction_1.BaseTransaction._validateNotArray(txData);
        if (this.gasLimit * this.maxFeePerGas > ethereumjs_util_1.MAX_INTEGER) {
            const msg = this._errorMsg('gasLimit * maxFeePerGas cannot exceed MAX_INTEGER (2^256-1)');
            throw new Error(msg);
        }
        if (this.maxFeePerGas < this.maxPriorityFeePerGas) {
            const msg = this._errorMsg('maxFeePerGas cannot be less than maxPriorityFeePerGas (The total must be the larger of the two)');
            throw new Error(msg);
        }
        this.maxFeePerDataGas = (0, ethereumjs_util_1.bufferToBigInt)((0, ethereumjs_util_1.toBuffer)((maxFeePerDataGas ?? '') === '' ? '0x' : maxFeePerDataGas));
        this.versionedHashes = (txData.versionedHashes ?? []).map((vh) => (0, ethereumjs_util_1.toBuffer)(vh));
        this._validateYParity();
        this._validateHighS();
        if (this.common.isActivatedEIP(3860) && this.txOptions.disableMaxInitCodeSizeCheck !== true) {
            (0, util_1.checkMaxInitCodeSize)(this.common, this.data.length);
        }
        for (const hash of this.versionedHashes) {
            if (hash.length !== 32) {
                const msg = this._errorMsg('versioned hash is invalid length');
                throw new Error(msg);
            }
            if (BigInt(hash[0]) !== this.common.paramByEIP('sharding', 'blobCommitmentVersionKzg', 4844)) {
                const msg = this._errorMsg('versioned hash does not start with KZG commitment version');
                throw new Error(msg);
            }
        }
        if (this.versionedHashes.length > constants_1.LIMIT_BLOBS_PER_TX) {
            const msg = this._errorMsg(`tx can contain at most ${constants_1.LIMIT_BLOBS_PER_TX} blobs`);
            throw new Error(msg);
        }
        this.blobs = txData.blobs?.map((blob) => (0, ethereumjs_util_1.toBuffer)(blob));
        this.kzgCommitments = txData.kzgCommitments?.map((commitment) => (0, ethereumjs_util_1.toBuffer)(commitment));
        this.aggregateKzgProof = (0, ethereumjs_util_1.toBuffer)(txData.kzgProof);
        const freeze = opts?.freeze ?? true;
        if (freeze) {
            Object.freeze(this);
        }
    }
    static fromTxData(txData, opts) {
        return new BlobEIP4844Transaction(txData, opts);
    }
    /**
     * Creates the minimal representation of a blob transaction from the network wrapper version.
     * The minimal representation is used when adding transactions to an execution payload/block
     * @param txData a {@link BlobEIP4844Transaction} containing optional blobs/kzg commitments
     * @param opts - dictionary of {@link TxOptions}
     * @returns the "minimal" representation of a BlobEIP4844Transaction (i.e. transaction object minus blobs and kzg commitments)
     */
    static minimalFromNetworkWrapper(txData, opts) {
        const tx = BlobEIP4844Transaction.fromTxData({
            ...txData,
            ...{ blobs: undefined, kzgCommitments: undefined, kzgProof: undefined },
        }, opts);
        return tx;
    }
    /**
     * Creates a transaction from the network encoding of a blob transaction (with blobs/commitments/proof)
     * @param serialized a buffer representing a serialized BlobTransactionNetworkWrapper
     * @param opts any TxOptions defined
     * @returns a BlobEIP4844Transaction
     * @throws if no KZG library is loaded -- using the `initKzg` helper method -- or if `opts.common` not provided
     */
    static fromSerializedBlobTxNetworkWrapper(serialized, opts) {
        if (!opts || !opts.common) {
            throw new Error('common instance required to validate versioned hashes');
        }
        // Validate network wrapper
        const wrapper = types_1.BlobNetworkTransactionWrapper.deserialize(serialized.slice(1));
        const decodedTx = wrapper.tx.message;
        const version = Number(opts.common.paramByEIP('sharding', 'blobCommitmentVersionKzg', 4844));
        validateBlobTransactionNetworkWrapper(decodedTx.blobVersionedHashes, wrapper.blobs, wrapper.blobKzgs, wrapper.kzgAggregatedProof, version);
        const accessList = [];
        for (const listItem of decodedTx.accessList) {
            const address = Buffer.from(listItem.address);
            const storageKeys = listItem.storageKeys.map((key) => Buffer.from(key));
            const accessListItem = [address, storageKeys];
            accessList.push(accessListItem);
        }
        const to = decodedTx.to.value === null
            ? undefined
            : ethereumjs_util_1.Address.fromString((0, ethereumjs_util_1.bufferToHex)(Buffer.from(decodedTx.to.value)));
        const versionedHashes = decodedTx.blobVersionedHashes.map((el) => Buffer.from(el));
        const commitments = wrapper.blobKzgs.map((el) => Buffer.from(el));
        const blobs = wrapper.blobs.map((el) => Buffer.from(el));
        const txData = {
            ...decodedTx,
            ...{
                versionedHashes,
                accessList,
                to,
                blobs,
                kzgCommitments: commitments,
                kzgProof: Buffer.from(wrapper.kzgAggregatedProof),
                r: wrapper.tx.signature.r,
                s: wrapper.tx.signature.s,
                v: BigInt(wrapper.tx.signature.yParity),
                gasLimit: decodedTx.gas,
                maxFeePerGas: decodedTx.maxFeePerGas,
                maxPriorityFeePerGas: decodedTx.maxPriorityFeePerGas,
            },
        };
        return new BlobEIP4844Transaction(txData, opts);
    }
    /**
     * Creates a transaction from the "minimal" encoding of a blob transaction (without blobs/commitments/kzg proof)
     * @param serialized a buffer representing a serialized signed blob transaction
     * @param opts any TxOptions defined
     * @returns a BlobEIP4844Transaction
     */
    static fromSerializedTx(serialized, opts) {
        const decoded = types_1.SignedBlobTransactionType.deserialize(serialized.slice(1));
        const tx = decoded.message;
        const accessList = [];
        for (const listItem of tx.accessList) {
            const address = Buffer.from(listItem.address);
            const storageKeys = listItem.storageKeys.map((key) => Buffer.from(key));
            const accessListItem = [address, storageKeys];
            accessList.push(accessListItem);
        }
        const to = tx.to.value === null ? undefined : ethereumjs_util_1.Address.fromString((0, ethereumjs_util_1.bufferToHex)(Buffer.from(tx.to.value)));
        const versionedHashes = tx.blobVersionedHashes.map((el) => Buffer.from(el));
        const txData = {
            ...tx,
            ...{
                versionedHashes,
                to,
                accessList,
                r: decoded.signature.r,
                s: decoded.signature.s,
                v: BigInt(decoded.signature.yParity),
                gasLimit: decoded.message.gas,
            },
        };
        return new BlobEIP4844Transaction(txData, opts);
    }
    /**
     * The up front amount that an account must have for this transaction to be valid
     * @param baseFee The base fee of the block (will be set to 0 if not provided)
     */
    getUpfrontCost(baseFee = BigInt(0)) {
        const prio = this.maxPriorityFeePerGas;
        const maxBase = this.maxFeePerGas - baseFee;
        const inclusionFeePerGas = prio < maxBase ? prio : maxBase;
        const gasPrice = inclusionFeePerGas + baseFee;
        return this.gasLimit * gasPrice + this.value;
    }
    /**
     * This method is not implemented for blob transactions as the `raw` method is used exclusively with
     * rlp encoding and these transactions use SSZ for serialization.
     */
    raw() {
        throw new Error('Method not implemented.');
    }
    toValue() {
        const to = {
            selector: this.to !== undefined ? 1 : 0,
            value: this.to?.toBuffer() ?? null,
        };
        return {
            message: {
                chainId: this.common.chainId(),
                nonce: this.nonce,
                maxPriorityFeePerGas: this.maxPriorityFeePerGas,
                maxFeePerGas: this.maxFeePerGas,
                gas: this.gasLimit,
                to,
                value: this.value,
                data: this.data,
                accessList: this.accessList.map((listItem) => {
                    return { address: listItem[0], storageKeys: listItem[1] };
                }),
                blobVersionedHashes: this.versionedHashes,
                maxFeePerDataGas: this.maxFeePerDataGas,
            },
            // TODO: Decide how to serialize an unsigned transaction
            signature: {
                r: this.r ?? BigInt(0),
                s: this.s ?? BigInt(0),
                yParity: this.v === BigInt(1) ? true : false,
            },
        };
    }
    /**
     * Serialize a blob transaction to the execution payload variant
     * @returns the minimum (execution payload) serialization of a signed transaction
     */
    serialize() {
        const sszEncodedTx = types_1.SignedBlobTransactionType.serialize(this.toValue());
        return Buffer.concat([TRANSACTION_TYPE_BUFFER, sszEncodedTx]);
    }
    /**
     * @returns the serialized form of a blob transaction in the network wrapper format (used for gossipping mempool transactions over devp2p)
     */
    serializeNetworkWrapper() {
        if (this.blobs === undefined ||
            this.kzgCommitments === undefined ||
            this.aggregateKzgProof === undefined) {
            throw new Error('cannot serialize network wrapper without blobs, KZG commitments and aggregate KZG proof provided');
        }
        const to = {
            selector: this.to !== undefined ? 1 : 0,
            value: this.to?.toBuffer() ?? null,
        };
        const blobArrays = this.blobs?.map((blob) => Uint8Array.from(blob)) ?? [];
        const serializedTxWrapper = types_1.BlobNetworkTransactionWrapper.serialize({
            blobs: blobArrays,
            blobKzgs: this.kzgCommitments?.map((commitment) => Uint8Array.from(commitment)) ?? [],
            tx: { ...(0, util_1.blobTxToNetworkWrapperDataFormat)(this), ...to },
            kzgAggregatedProof: Uint8Array.from(this.aggregateKzgProof ?? []),
        });
        return Buffer.concat([Buffer.from([0x05]), serializedTxWrapper]);
    }
    getMessageToSign(_hashMessage) {
        return this.unsignedHash();
    }
    /**
     * Returns the hash of a blob transaction
     */
    unsignedHash() {
        const serializedTx = types_1.BlobTransactionType.serialize(this.toValue().message);
        return Buffer.from((0, keccak_1.keccak256)(Buffer.concat([TRANSACTION_TYPE_BUFFER, serializedTx])));
    }
    hash() {
        return Buffer.from((0, keccak_1.keccak256)(this.serialize()));
    }
    getMessageToVerifySignature() {
        return this.getMessageToSign();
    }
    /**
     * Returns the public key of the sender
     */
    getSenderPublicKey() {
        if (!this.isSigned()) {
            const msg = this._errorMsg('Cannot call this method if transaction is not signed');
            throw new Error(msg);
        }
        const msgHash = this.getMessageToVerifySignature();
        const { v, r, s } = this;
        this._validateHighS();
        try {
            return (0, ethereumjs_util_1.ecrecover)(msgHash, v + BigInt(27), // Recover the 27 which was stripped from ecsign
            (0, ethereumjs_util_1.bigIntToUnpaddedBuffer)(r), (0, ethereumjs_util_1.bigIntToUnpaddedBuffer)(s));
        }
        catch (e) {
            const msg = this._errorMsg('Invalid Signature');
            throw new Error(msg);
        }
    }
    toJSON() {
        const accessListJSON = util_1.AccessLists.getAccessListJSON(this.accessList);
        return {
            chainId: (0, ethereumjs_util_1.bigIntToHex)(this.chainId),
            nonce: (0, ethereumjs_util_1.bigIntToHex)(this.nonce),
            maxPriorityFeePerGas: (0, ethereumjs_util_1.bigIntToHex)(this.maxPriorityFeePerGas),
            maxFeePerGas: (0, ethereumjs_util_1.bigIntToHex)(this.maxFeePerGas),
            gasLimit: (0, ethereumjs_util_1.bigIntToHex)(this.gasLimit),
            to: this.to !== undefined ? this.to.toString() : undefined,
            value: (0, ethereumjs_util_1.bigIntToHex)(this.value),
            data: '0x' + this.data.toString('hex'),
            accessList: accessListJSON,
            v: this.v !== undefined ? (0, ethereumjs_util_1.bigIntToHex)(this.v) : undefined,
            r: this.r !== undefined ? (0, ethereumjs_util_1.bigIntToHex)(this.r) : undefined,
            s: this.s !== undefined ? (0, ethereumjs_util_1.bigIntToHex)(this.s) : undefined,
            maxFeePerDataGas: (0, ethereumjs_util_1.bigIntToHex)(this.maxFeePerDataGas),
            versionedHashes: this.versionedHashes.map((hash) => (0, ethereumjs_util_1.bufferToHex)(hash)),
        };
    }
    _processSignature(v, r, s) {
        const opts = { ...this.txOptions, common: this.common };
        return BlobEIP4844Transaction.fromTxData({
            chainId: this.chainId,
            nonce: this.nonce,
            maxPriorityFeePerGas: this.maxPriorityFeePerGas,
            maxFeePerGas: this.maxFeePerGas,
            gasLimit: this.gasLimit,
            to: this.to,
            value: this.value,
            data: this.data,
            accessList: this.accessList,
            v: v - BigInt(27),
            r: (0, ethereumjs_util_1.bufferToBigInt)(r),
            s: (0, ethereumjs_util_1.bufferToBigInt)(s),
            maxFeePerDataGas: this.maxFeePerDataGas,
            versionedHashes: this.versionedHashes,
            blobs: this.blobs,
            kzgCommitments: this.kzgCommitments,
            kzgProof: this.aggregateKzgProof,
        }, opts);
    }
    /**
     * Return a compact error string representation of the object
     */
    errorStr() {
        let errorStr = this._getSharedErrorPostfix();
        errorStr += ` maxFeePerGas=${this.maxFeePerGas} maxPriorityFeePerGas=${this.maxPriorityFeePerGas}`;
        return errorStr;
    }
    /**
     * Internal helper function to create an annotated error message
     *
     * @param msg Base error message
     * @hidden
     */
    _errorMsg(msg) {
        return `${msg} (${this.errorStr()})`;
    }
    /**
     * @returns the number of blobs included with this transaction
     */
    numBlobs() {
        return this.versionedHashes.length;
    }
}
exports.BlobEIP4844Transaction = BlobEIP4844Transaction;
//# sourceMappingURL=eip4844Transaction.js.map