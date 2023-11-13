"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DBManager = void 0;
const ethereumjs_block_1 = require("@nomicfoundation/ethereumjs-block");
const ethereumjs_rlp_1 = require("@nomicfoundation/ethereumjs-rlp");
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const cache_1 = require("./cache");
const operation_1 = require("./operation");
class NotFoundError extends Error {
    constructor(blockNumber) {
        super(`Key ${blockNumber.toString()} was not found`);
        this.code = 'LEVEL_NOT_FOUND';
        // `Error.captureStackTrace` is not defined in some browser contexts
        if (typeof Error.captureStackTrace !== 'undefined') {
            Error.captureStackTrace(this, this.constructor);
        }
    }
}
/**
 * Abstraction over a DB to facilitate storing/fetching blockchain-related
 * data, such as blocks and headers, indices, and the head block.
 * @hidden
 */
class DBManager {
    constructor(db, common) {
        this._db = db;
        this._common = common;
        this._cache = {
            td: new cache_1.Cache({ max: 1024 }),
            header: new cache_1.Cache({ max: 512 }),
            body: new cache_1.Cache({ max: 256 }),
            numberToHash: new cache_1.Cache({ max: 2048 }),
            hashToNumber: new cache_1.Cache({ max: 2048 }),
        };
    }
    /**
     * Fetches iterator heads from the db.
     */
    async getHeads() {
        const heads = await this.get(operation_1.DBTarget.Heads);
        for (const key of Object.keys(heads)) {
            heads[key] = Buffer.from(heads[key]);
        }
        return heads;
    }
    /**
     * Fetches header of the head block.
     */
    async getHeadHeader() {
        return this.get(operation_1.DBTarget.HeadHeader);
    }
    /**
     * Fetches head block.
     */
    async getHeadBlock() {
        return this.get(operation_1.DBTarget.HeadBlock);
    }
    /**
     * Fetches a block (header and body) given a block id,
     * which can be either its hash or its number.
     */
    async getBlock(blockId) {
        if (typeof blockId === 'number' && Number.isInteger(blockId)) {
            blockId = BigInt(blockId);
        }
        let number;
        let hash;
        if (Buffer.isBuffer(blockId)) {
            hash = blockId;
            number = await this.hashToNumber(blockId);
        }
        else if (typeof blockId === 'bigint') {
            number = blockId;
            hash = await this.numberToHash(blockId);
        }
        else {
            throw new Error('Unknown blockId type');
        }
        const header = await this.getHeader(hash, number);
        let body;
        try {
            body = await this.getBody(hash, number);
        }
        catch (error) {
            if (error.code !== 'LEVEL_NOT_FOUND') {
                throw error;
            }
            // Do extra validations on the header since we are assuming empty transactions and uncles
            if (!header.transactionsTrie.equals(ethereumjs_util_1.KECCAK256_RLP) ||
                !header.uncleHash.equals(ethereumjs_util_1.KECCAK256_RLP_ARRAY)) {
                throw error;
            }
            body = [[], []];
            // If this block had empty withdrawals push an empty array in body
            if (header.withdrawalsRoot !== undefined) {
                // Do extra validations for withdrawal before assuming empty withdrawals
                if (!header.withdrawalsRoot.equals(ethereumjs_util_1.KECCAK256_RLP)) {
                    throw error;
                }
                body.push([]);
            }
        }
        const blockData = [header.raw(), ...body];
        const opts = { common: this._common };
        if (number === BigInt(0)) {
            opts.hardforkByTTD = await this.getTotalDifficulty(hash, BigInt(0));
        }
        else {
            opts.hardforkByTTD = await this.getTotalDifficulty(header.parentHash, number - BigInt(1));
        }
        return ethereumjs_block_1.Block.fromValuesArray(blockData, opts);
    }
    /**
     * Fetches body of a block given its hash and number.
     */
    async getBody(blockHash, blockNumber) {
        const body = await this.get(operation_1.DBTarget.Body, { blockHash, blockNumber });
        return (0, ethereumjs_util_1.arrToBufArr)(ethereumjs_rlp_1.RLP.decode(Uint8Array.from(body)));
    }
    /**
     * Fetches header of a block given its hash and number.
     */
    async getHeader(blockHash, blockNumber) {
        const encodedHeader = await this.get(operation_1.DBTarget.Header, { blockHash, blockNumber });
        const headerValues = (0, ethereumjs_util_1.arrToBufArr)(ethereumjs_rlp_1.RLP.decode(Uint8Array.from(encodedHeader)));
        const opts = { common: this._common };
        if (blockNumber === BigInt(0)) {
            opts.hardforkByTTD = await this.getTotalDifficulty(blockHash, BigInt(0));
        }
        else {
            // Lets fetch the parent hash but not by number since this block might not
            // be in canonical chain
            const headerData = (0, ethereumjs_block_1.valuesArrayToHeaderData)(headerValues);
            const parentHash = headerData.parentHash;
            opts.hardforkByTTD = await this.getTotalDifficulty(parentHash, blockNumber - BigInt(1));
        }
        return ethereumjs_block_1.BlockHeader.fromValuesArray(headerValues, opts);
    }
    /**
     * Fetches total difficulty for a block given its hash and number.
     */
    async getTotalDifficulty(blockHash, blockNumber) {
        const td = await this.get(operation_1.DBTarget.TotalDifficulty, { blockHash, blockNumber });
        return (0, ethereumjs_util_1.bufferToBigInt)(Buffer.from(ethereumjs_rlp_1.RLP.decode(Uint8Array.from(td))));
    }
    /**
     * Performs a block hash to block number lookup.
     */
    async hashToNumber(blockHash) {
        const value = await this.get(operation_1.DBTarget.HashToNumber, { blockHash });
        return (0, ethereumjs_util_1.bufferToBigInt)(value);
    }
    /**
     * Performs a block number to block hash lookup.
     */
    async numberToHash(blockNumber) {
        if (blockNumber < BigInt(0)) {
            throw new NotFoundError(blockNumber);
        }
        return this.get(operation_1.DBTarget.NumberToHash, { blockNumber });
    }
    /**
     * Fetches a key from the db. If `opts.cache` is specified
     * it first tries to load from cache, and on cache miss will
     * try to put the fetched item on cache afterwards.
     */
    async get(dbOperationTarget, key) {
        const dbGetOperation = operation_1.DBOp.get(dbOperationTarget, key);
        const cacheString = dbGetOperation.cacheString;
        const dbKey = dbGetOperation.baseDBOp.key;
        const dbOpts = dbGetOperation.baseDBOp;
        if (cacheString !== undefined) {
            if (this._cache[cacheString] === undefined) {
                throw new Error(`Invalid cache: ${cacheString}`);
            }
            let value = this._cache[cacheString].get(dbKey);
            if (!value) {
                value = await this._db.get(dbKey, dbOpts);
                if (value) {
                    this._cache[cacheString].set(dbKey, value);
                }
            }
            return value;
        }
        return this._db.get(dbKey, dbOpts);
    }
    /**
     * Performs a batch operation on db.
     */
    async batch(ops) {
        const convertedOps = ops.map((op) => op.baseDBOp);
        // update the current cache for each operation
        ops.map((op) => op.updateCache(this._cache));
        return this._db.batch(convertedOps);
    }
}
exports.DBManager = DBManager;
//# sourceMappingURL=manager.js.map