"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.precompile14 = exports.BLS_MODULUS = void 0;
const ethereumjs_tx_1 = require("@nomicfoundation/ethereumjs-tx");
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const evm_1 = require("../evm");
const exceptions_1 = require("../exceptions");
exports.BLS_MODULUS = BigInt('52435875175126190479447740508185965837690552500527637822603658699938581184513');
async function precompile14(opts) {
    const gasUsed = opts._common.param('gasPrices', 'kzgPointEvaluationGasPrecompilePrice');
    const version = Number(opts._common.paramByEIP('sharding', 'blobCommitmentVersionKzg', 4844));
    const fieldElementsPerBlob = opts._common.paramByEIP('sharding', 'fieldElementsPerBlob', 4844);
    const versionedHash = opts.data.slice(0, 32);
    const z = opts.data.slice(32, 64);
    const y = opts.data.slice(64, 96);
    const commitment = opts.data.slice(96, 144);
    const kzgProof = opts.data.slice(144, 192);
    if ((0, ethereumjs_util_1.bufferToBigInt)(z) >= exports.BLS_MODULUS || (0, ethereumjs_util_1.bufferToBigInt)(y) >= exports.BLS_MODULUS) {
        return (0, evm_1.EvmErrorResult)(new exceptions_1.EvmError(exceptions_1.ERROR.POINT_GREATER_THAN_BLS_MODULUS), opts.gasLimit);
    }
    if ((0, ethereumjs_util_1.bufferToHex)(Buffer.from((0, ethereumjs_tx_1.computeVersionedHash)(commitment, version))) !==
        (0, ethereumjs_util_1.bufferToHex)(versionedHash)) {
        return (0, evm_1.EvmErrorResult)(new exceptions_1.EvmError(exceptions_1.ERROR.INVALID_COMMITMENT), opts.gasLimit);
    }
    ethereumjs_tx_1.kzg.verifyKzgProof(commitment, z, y, kzgProof);
    // Return value - FIELD_ELEMENTS_PER_BLOB and BLS_MODULUS as padded 32 byte big endian values
    const fieldElementsBuffer = (0, ethereumjs_util_1.setLengthLeft)((0, ethereumjs_util_1.bigIntToBuffer)(fieldElementsPerBlob), 32);
    const modulusBuffer = (0, ethereumjs_util_1.setLengthLeft)((0, ethereumjs_util_1.bigIntToBuffer)(exports.BLS_MODULUS), 32);
    return {
        executionGasUsed: gasUsed,
        returnValue: Buffer.concat([fieldElementsBuffer, modulusBuffer]),
    };
}
exports.precompile14 = precompile14;
//# sourceMappingURL=14-kzg-point-evaluation.js.map