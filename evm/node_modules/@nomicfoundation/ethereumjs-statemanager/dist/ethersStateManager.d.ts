/// <reference types="node" />
import { Account } from '@nomicfoundation/ethereumjs-util';
import { ethers } from 'ethers';
import { Cache } from './cache';
import { BaseStateManager } from '.';
import type { Proof, StateManager } from '.';
import type { StorageDump } from './interface';
import type { Address } from '@nomicfoundation/ethereumjs-util';
export interface EthersStateManagerOpts {
    provider: string | ethers.providers.StaticJsonRpcProvider | ethers.providers.JsonRpcProvider;
    blockTag: bigint | 'earliest';
}
export declare class EthersStateManager extends BaseStateManager implements StateManager {
    private provider;
    private contractCache;
    private storageCache;
    private blockTag;
    _cache: Cache;
    constructor(opts: EthersStateManagerOpts);
    copy(): EthersStateManager;
    /**
     * Sets the new block tag used when querying the provider and clears the
     * internal cache.
     * @param blockTag - the new block tag to use when querying the provider
     */
    setBlockTag(blockTag: bigint | 'earliest'): void;
    /**
     * Clears the internal cache so all accounts, contract code, and storage slots will
     * initially be retrieved from the provider
     */
    clearCache(): void;
    /**
     * Gets the code corresponding to the provided `address`.
     * @param address - Address to get the `code` for
     * @returns {Promise<Buffer>} - Resolves with the code corresponding to the provided address.
     * Returns an empty `Buffer` if the account has no associated code.
     */
    getContractCode(address: Address): Promise<Buffer>;
    /**
     * Adds `value` to the state trie as code, and sets `codeHash` on the account
     * corresponding to `address` to reference this.
     * @param address - Address of the `account` to add the `code` for
     * @param value - The value of the `code`
     */
    putContractCode(address: Address, value: Buffer): Promise<void>;
    /**
     * Gets the storage value associated with the provided `address` and `key`. This method returns
     * the shortest representation of the stored value.
     * @param address - Address of the account to get the storage for
     * @param key - Key in the account's storage to get the value for. Must be 32 bytes long.
     * @returns {Buffer} - The storage value for the account
     * corresponding to the provided address at the provided key.
     * If this does not exist an empty `Buffer` is returned.
     */
    getContractStorage(address: Address, key: Buffer): Promise<Buffer>;
    /**
     * Adds value to the cache for the `account`
     * corresponding to `address` at the provided `key`.
     * @param address - Address to set a storage value for
     * @param key - Key to set the value at. Must be 32 bytes long.
     * @param value - Value to set at `key` for account corresponding to `address`.
     * Cannot be more than 32 bytes. Leading zeros are stripped.
     * If it is empty or filled with zeros, deletes the value.
     */
    putContractStorage(address: Address, key: Buffer, value: Buffer): Promise<void>;
    /**
     * Clears all storage entries for the account corresponding to `address`.
     * @param address - Address to clear the storage of
     */
    clearContractStorage(address: Address): Promise<void>;
    /**
     * Dumps the RLP-encoded storage values for an `account` specified by `address`.
     * @param address - The address of the `account` to return storage for
     * @returns {Promise<StorageDump>} - The state of the account as an `Object` map.
     * Keys are the storage keys, values are the storage values as strings.
     * Both are represented as `0x` prefixed hex strings.
     */
    dumpStorage(address: Address): Promise<StorageDump>;
    /**
     * Checks if an `account` exists at `address`
     * @param address - Address of the `account` to check
     */
    accountExists(address: Address): Promise<boolean>;
    /**
     * Gets the code corresponding to the provided `address`.
     * @param address - Address to get the `code` for
     * @returns {Promise<Buffer>} - Resolves with the code corresponding to the provided address.
     * Returns an empty `Buffer` if the account has no associated code.
     */
    getAccount(address: Address): Promise<Account>;
    /**
     * Retrieves an account from the provider and stores in the local trie
     * @param address Address of account to be retrieved from provider
     * @private
     */
    getAccountFromProvider(address: Address): Promise<Account>;
    /**
     * Saves an account into state under the provided `address`.
     * @param address - Address under which to store `account`
     * @param account - The account to store
     */
    putAccount(address: Address, account: Account): Promise<void>;
    /**
     * Get an EIP-1186 proof from the provider
     * @param address address to get proof of
     * @param storageSlots storage slots to get proof of
     * @returns an EIP-1186 formatted proof
     */
    getProof(address: Address, storageSlots?: Buffer[]): Promise<Proof>;
    /**
     * Checkpoints the current state of the StateManager instance.
     * State changes that follow can then be committed by calling
     * `commit` or `reverted` by calling rollback.
     *
     * Partial implementation, called from the subclass.
     */
    checkpoint(): Promise<void>;
    /**
     * Commits the current change-set to the instance since the
     * last call to checkpoint.
     *
     * Partial implementation, called from the subclass.
     */
    commit(): Promise<void>;
    /**
     * Reverts the current change-set to the instance since the
     * last call to checkpoint.
     *
     * Partial implementation , called from the subclass.
     */
    revert(): Promise<void>;
    flush(): Promise<void>;
    /**
     * @deprecated This method is not used by the Ethers State Manager and is a stub required by the State Manager interface
     */
    getStateRoot: () => Promise<Buffer>;
    /**
     * @deprecated This method is not used by the Ethers State Manager and is a stub required by the State Manager interface
     */
    setStateRoot: (_root: Buffer) => Promise<void>;
    /**
     * @deprecated This method is not used by the Ethers State Manager and is a stub required by the State Manager interface
     */
    hasStateRoot: () => never;
}
//# sourceMappingURL=ethersStateManager.d.ts.map