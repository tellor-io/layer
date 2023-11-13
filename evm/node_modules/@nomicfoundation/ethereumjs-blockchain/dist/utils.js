"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.parseGethGenesisState = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
/**
 * Parses the geth genesis state into Blockchain {@link GenesisState}
 * @param json representing the `alloc` key in a Geth genesis file
 */
function parseGethGenesisState(json) {
    const state = {};
    for (let address of Object.keys(json.alloc)) {
        let { balance, code, storage } = json.alloc[address];
        address = (0, ethereumjs_util_1.addHexPrefix)(address);
        balance = (0, ethereumjs_util_1.isHexPrefixed)(balance) ? balance : (0, ethereumjs_util_1.bigIntToHex)(BigInt(balance));
        code = code !== undefined ? (0, ethereumjs_util_1.addHexPrefix)(code) : undefined;
        storage = storage !== undefined ? Object.entries(storage) : undefined;
        state[address] = [balance, code, storage];
    }
    return state;
}
exports.parseGethGenesisState = parseGethGenesisState;
//# sourceMappingURL=utils.js.map