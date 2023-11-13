/// <reference types="node" />
import { BaseTransaction } from './baseTransaction';
import { SignedBlobTransactionType } from './types';
import type { AccessList, AccessListBuffer, BlobEIP4844TxData, JsonTx, TxOptions, TxValuesArray } from './types';
import type { ValueOf } from '@chainsafe/ssz';
import type { Common } from '@nomicfoundation/ethereumjs-common';
/**
 * Typed transaction with a new gas fee market mechanism for transactions that include "blobs" of data
 *
 * - TransactionType: 5
 * - EIP: [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844)
 */
export declare class BlobEIP4844Transaction extends BaseTransaction<BlobEIP4844Transaction> {
    readonly chainId: bigint;
    readonly accessList: AccessListBuffer;
    readonly AccessListJSON: AccessList;
    readonly maxPriorityFeePerGas: bigint;
    readonly maxFeePerGas: bigint;
    readonly maxFeePerDataGas: bigint;
    readonly common: Common;
    versionedHashes: Buffer[];
    blobs?: Buffer[];
    kzgCommitments?: Buffer[];
    aggregateKzgProof?: Buffer;
    /**
     * This constructor takes the values, validates them, assigns them and freezes the object.
     *
     * It is not recommended to use this constructor directly. Instead use
     * the static constructors or factory methods to assist in creating a Transaction object from
     * varying data types.
     */
    constructor(txData: BlobEIP4844TxData, opts?: TxOptions);
    static fromTxData(txData: BlobEIP4844TxData, opts?: TxOptions): BlobEIP4844Transaction;
    /**
     * Creates the minimal representation of a blob transaction from the network wrapper version.
     * The minimal representation is used when adding transactions to an execution payload/block
     * @param txData a {@link BlobEIP4844Transaction} containing optional blobs/kzg commitments
     * @param opts - dictionary of {@link TxOptions}
     * @returns the "minimal" representation of a BlobEIP4844Transaction (i.e. transaction object minus blobs and kzg commitments)
     */
    static minimalFromNetworkWrapper(txData: BlobEIP4844Transaction, opts?: TxOptions): BlobEIP4844Transaction;
    /**
     * Creates a transaction from the network encoding of a blob transaction (with blobs/commitments/proof)
     * @param serialized a buffer representing a serialized BlobTransactionNetworkWrapper
     * @param opts any TxOptions defined
     * @returns a BlobEIP4844Transaction
     * @throws if no KZG library is loaded -- using the `initKzg` helper method -- or if `opts.common` not provided
     */
    static fromSerializedBlobTxNetworkWrapper(serialized: Buffer, opts?: TxOptions): BlobEIP4844Transaction;
    /**
     * Creates a transaction from the "minimal" encoding of a blob transaction (without blobs/commitments/kzg proof)
     * @param serialized a buffer representing a serialized signed blob transaction
     * @param opts any TxOptions defined
     * @returns a BlobEIP4844Transaction
     */
    static fromSerializedTx(serialized: Buffer, opts?: TxOptions): BlobEIP4844Transaction;
    /**
     * The up front amount that an account must have for this transaction to be valid
     * @param baseFee The base fee of the block (will be set to 0 if not provided)
     */
    getUpfrontCost(baseFee?: bigint): bigint;
    /**
     * This method is not implemented for blob transactions as the `raw` method is used exclusively with
     * rlp encoding and these transactions use SSZ for serialization.
     */
    raw(): TxValuesArray;
    toValue(): ValueOf<typeof SignedBlobTransactionType>;
    /**
     * Serialize a blob transaction to the execution payload variant
     * @returns the minimum (execution payload) serialization of a signed transaction
     */
    serialize(): Buffer;
    /**
     * @returns the serialized form of a blob transaction in the network wrapper format (used for gossipping mempool transactions over devp2p)
     */
    serializeNetworkWrapper(): Buffer;
    getMessageToSign(hashMessage: false): Buffer | Buffer[];
    getMessageToSign(hashMessage?: true | undefined): Buffer;
    /**
     * Returns the hash of a blob transaction
     */
    unsignedHash(): Buffer;
    hash(): Buffer;
    getMessageToVerifySignature(): Buffer;
    /**
     * Returns the public key of the sender
     */
    getSenderPublicKey(): Buffer;
    toJSON(): JsonTx;
    _processSignature(v: bigint, r: Buffer, s: Buffer): BlobEIP4844Transaction;
    /**
     * Return a compact error string representation of the object
     */
    errorStr(): string;
    /**
     * Internal helper function to create an annotated error message
     *
     * @param msg Base error message
     * @hidden
     */
    protected _errorMsg(msg: string): string;
    /**
     * @returns the number of blobs included with this transaction
     */
    numBlobs(): number;
}
//# sourceMappingURL=eip4844Transaction.d.ts.map