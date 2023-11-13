"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeTxParams = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const normalizeTxParams = (_txParams) => {
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
};
exports.normalizeTxParams = normalizeTxParams;
//# sourceMappingURL=fromRpc.js.map