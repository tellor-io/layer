"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Withdrawals = exports.Withdrawal = exports.Bytes20 = exports.UintBigInt64 = exports.UintNum64 = void 0;
const ssz_1 = require("@chainsafe/ssz");
const constants_1 = require("./constants");
exports.UintNum64 = new ssz_1.UintNumberType(8);
exports.UintBigInt64 = new ssz_1.UintBigintType(8);
exports.Bytes20 = new ssz_1.ByteVectorType(20);
exports.Withdrawal = new ssz_1.ContainerType({
    index: exports.UintBigInt64,
    validatorIndex: exports.UintBigInt64,
    address: exports.Bytes20,
    amount: exports.UintBigInt64,
}, { typeName: 'Withdrawal', jsonCase: 'eth2' });
exports.Withdrawals = new ssz_1.ListCompositeType(exports.Withdrawal, constants_1.MAX_WITHDRAWALS_PER_PAYLOAD);
//# sourceMappingURL=ssz.js.map