/// <reference types="node" />
import { Block } from '@nomicfoundation/ethereumjs-block';
import type { BuildBlockOpts, RunTxResult, SealBlockOpts } from './types';
import type { VM } from './vm';
import type { TypedTransaction } from '@nomicfoundation/ethereumjs-tx';
export declare enum BuildStatus {
    Reverted = "reverted",
    Build = "build",
    Pending = "pending"
}
declare type BlockStatus = {
    status: BuildStatus.Pending | BuildStatus.Reverted;
} | {
    status: BuildStatus.Build;
    block: Block;
};
export declare class BlockBuilder {
    /**
     * The cumulative gas used by the transactions added to the block.
     */
    gasUsed: bigint;
    /**
     *  The cumulative data gas used by the blobs in a block
     */
    dataGasUsed: bigint;
    /**
     * Value of the block, represented by the final transaction fees
     * acruing to the miner.
     */
    private _minerValue;
    private readonly vm;
    private blockOpts;
    private headerData;
    private transactions;
    private transactionResults;
    private withdrawals?;
    private checkpointed;
    private blockStatus;
    get transactionReceipts(): import("./types").TxReceipt[];
    get minerValue(): bigint;
    constructor(vm: VM, opts: BuildBlockOpts);
    /**
     * Throws if the block has already been built or reverted.
     */
    private checkStatus;
    getStatus(): BlockStatus;
    /**
     * Calculates and returns the transactionsTrie for the block.
     */
    transactionsTrie(): Promise<Buffer>;
    /**
     * Calculates and returns the logs bloom for the block.
     */
    logsBloom(): Buffer;
    /**
     * Calculates and returns the receiptTrie for the block.
     */
    receiptTrie(): Promise<Buffer>;
    /**
     * Adds the block miner reward to the coinbase account.
     */
    private rewardMiner;
    /**
     * Adds the withdrawal amount to the withdrawal address
     */
    private processWithdrawals;
    /**
     * Run and add a transaction to the block being built.
     * Please note that this modifies the state of the VM.
     * Throws if the transaction's gasLimit is greater than
     * the remaining gas in the block.
     */
    addTransaction(tx: TypedTransaction, { skipHardForkValidation }?: {
        skipHardForkValidation?: boolean;
    }): Promise<RunTxResult>;
    /**
     * Reverts the checkpoint on the StateManager to reset the state from any transactions that have been run.
     */
    revert(): Promise<void>;
    /**
     * This method returns the finalized block.
     * It also:
     *  - Assigns the reward for miner (PoW)
     *  - Commits the checkpoint on the StateManager
     *  - Sets the tip of the VM's blockchain to this block
     * For PoW, optionally seals the block with params `nonce` and `mixHash`,
     * which is validated along with the block number and difficulty by ethash.
     * For PoA, please pass `blockOption.cliqueSigner` into the buildBlock constructor,
     * as the signer will be awarded the txs amount spent on gas as they are added.
     */
    build(sealOpts?: SealBlockOpts): Promise<Block>;
}
export declare function buildBlock(this: VM, opts: BuildBlockOpts): Promise<BlockBuilder>;
export {};
//# sourceMappingURL=buildBlock.d.ts.map