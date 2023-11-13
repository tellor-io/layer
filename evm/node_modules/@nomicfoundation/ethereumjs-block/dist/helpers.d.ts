import type { BlockHeader } from './header';
import type { BlockHeaderBuffer, HeaderData } from './types';
/**
 * Returns a 0x-prefixed hex number string from a hex string or string integer.
 * @param {string} input string to check, convert, and return
 */
export declare const numberToHex: (input?: string) => string | undefined;
export declare function valuesArrayToHeaderData(values: BlockHeaderBuffer): HeaderData;
export declare function getDifficulty(headerData: HeaderData): bigint | null;
/**
 * Calculates the excess data gas for a post EIP 4844 block given the parent block header.
 * @param parent header for the parent block
 * @param newBlobs number of blobs contained in block
 * @returns the excess data gas for the prospective next block
 *
 * Note: This function expects that it is only being called on a valid block as it does not have
 * access to the "current" block's common instance to verify if 4844 is active or not.
 */
export declare const calcExcessDataGas: (parent: BlockHeader, newBlobs: number) => bigint;
/**
 * Approximates `factor * e ** (numerator / denominator)` using Taylor expansion
 */
export declare const fakeExponential: (factor: bigint, numerator: bigint, denominator: bigint) => bigint;
/**
 * Returns the price per unit of data gas for a blob transaction in the current/pending block
 * @param header the parent header for the current block (or current head of the chain)
 * @returns the price in gwei per unit of data gas spent
 */
export declare const getDataGasPrice: (header: BlockHeader) => bigint;
/**
 * Returns the total fee for data gas spent on `numBlobs` in the current/pending block
 * @param numBlobs
 * @param parent parent header of the current/pending block
 * @returns the total data gas fee for a transaction assuming it contains `numBlobs`
 */
export declare const calcDataFee: (numBlobs: number, parent: BlockHeader) => bigint;
//# sourceMappingURL=helpers.d.ts.map