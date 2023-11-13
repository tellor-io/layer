"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.blockFromRpc = void 0;
const ethereumjs_tx_1 = require("@nomicfoundation/ethereumjs-tx");
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const header_from_rpc_1 = require("./header-from-rpc");
const index_1 = require("./index");
function normalizeTxParams(_txParams) {
    const txParams = Object.assign({}, _txParams);
    txParams.gasLimit = (0, ethereumjs_util_1.toType)(txParams.gasLimit ?? txParams.gas, ethereumjs_util_1.TypeOutput.BigInt);
    txParams.data = txParams.data === undefined ? txParams.input : txParams.data;
    // check and convert gasPrice and value params
    txParams.gasPrice = txParams.gasPrice !== undefined ? BigInt(txParams.gasPrice) : undefined;
    txParams.value = txParams.value !== undefined ? BigInt(txParams.value) : undefined;
    // strict byte length checking
    txParams.to =
        txParams.to !== null && txParams.to !== undefined
            ? (0, ethereumjs_util_1.setLengthLeft)((0, ethereumjs_util_1.toBuffer)(txParams.to), 20)
            : null;
    txParams.v = (0, ethereumjs_util_1.toType)(txParams.v, ethereumjs_util_1.TypeOutput.BigInt);
    return txParams;
}
/**
 * Creates a new block object from Ethereum JSON RPC.
 *
 * @param blockParams - Ethereum JSON RPC of block (eth_getBlockByNumber)
 * @param uncles - Optional list of Ethereum JSON RPC of uncles (eth_getUncleByBlockHashAndIndex)
 * @param options - An object describing the blockchain
 * @deprecated
 */
function blockFromRpc(blockParams, uncles = [], options) {
    const header = (0, header_from_rpc_1.blockHeaderFromRpc)(blockParams, options);
    const transactions = [];
    const opts = { common: header._common };
    for (const _txParams of blockParams.transactions ?? []) {
        const txParams = normalizeTxParams(_txParams);
        const tx = ethereumjs_tx_1.TransactionFactory.fromTxData(txParams, opts);
        transactions.push(tx);
    }
    const uncleHeaders = uncles.map((uh) => (0, header_from_rpc_1.blockHeaderFromRpc)(uh, options));
    return index_1.Block.fromBlockData({ header, transactions, uncleHeaders }, options);
}
exports.blockFromRpc = blockFromRpc;
//# sourceMappingURL=from-rpc.js.map