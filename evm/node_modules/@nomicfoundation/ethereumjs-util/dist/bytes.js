"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.bytesToBigInt = exports.bytesToHex = exports.intToUnpaddedBuffer = exports.bigIntToUnpaddedBuffer = exports.bigIntToHex = exports.bufArrToArr = exports.arrToBufArr = exports.validateNoLeadingZeroes = exports.baToJSON = exports.toUtf8 = exports.short = exports.addHexPrefix = exports.toUnsigned = exports.fromSigned = exports.toBytes = exports.unprefixedHexToBytes = exports.intToBytes = exports.hexToBytes = exports.bufferToInt = exports.bigIntToBuffer = exports.bufferToBigInt = exports.bufferToHex = exports.toBuffer = exports.unpadHexString = exports.unpadArray = exports.unpadBuffer = exports.setLengthRight = exports.setLengthLeft = exports.zeros = exports.bigIntToBytes = exports.intToBuffer = exports.intToHex = void 0;
const helpers_1 = require("./helpers");
const internal_1 = require("./internal");
/**
 * Converts a `Number` into a hex `String`
 * @param {Number} i
 * @return {String}
 */
const intToHex = function (i) {
    if (!Number.isSafeInteger(i) || i < 0) {
        throw new Error(`Received an invalid integer type: ${i}`);
    }
    return `0x${i.toString(16)}`;
};
exports.intToHex = intToHex;
/**
 * Converts an `Number` to a `Buffer`
 * @param {Number} i
 * @return {Buffer}
 */
const intToBuffer = function (i) {
    const hex = (0, exports.intToHex)(i);
    return Buffer.from((0, internal_1.padToEven)(hex.slice(2)), 'hex');
};
exports.intToBuffer = intToBuffer;
/**
 * Converts a {@link bigint} to a {@link Uint8Array}
 *  * @param {bigint} num the bigint to convert
 * @returns {Uint8Array}
 */
const bigIntToBytes = (num) => {
    // eslint-disable-next-line @typescript-eslint/no-use-before-define
    return (0, exports.toBytes)('0x' + (0, internal_1.padToEven)(num.toString(16)));
};
exports.bigIntToBytes = bigIntToBytes;
/**
 * Returns a buffer filled with 0s.
 * @param bytes the number of bytes the buffer should be
 */
const zeros = function (bytes) {
    return Buffer.allocUnsafe(bytes).fill(0);
};
exports.zeros = zeros;
/**
 * Pads a `Buffer` with zeros till it has `length` bytes.
 * Truncates the beginning or end of input if its length exceeds `length`.
 * @param msg the value to pad (Buffer)
 * @param length the number of bytes the output should be
 * @param right whether to start padding form the left or right
 * @return (Buffer)
 */
const setLength = function (msg, length, right) {
    const buf = (0, exports.zeros)(length);
    if (right) {
        if (msg.length < length) {
            msg.copy(buf);
            return buf;
        }
        return msg.slice(0, length);
    }
    else {
        if (msg.length < length) {
            msg.copy(buf, length - msg.length);
            return buf;
        }
        return msg.slice(-length);
    }
};
/**
 * Left Pads a `Buffer` with leading zeros till it has `length` bytes.
 * Or it truncates the beginning if it exceeds.
 * @param msg the value to pad (Buffer)
 * @param length the number of bytes the output should be
 * @return (Buffer)
 */
const setLengthLeft = function (msg, length) {
    (0, helpers_1.assertIsBuffer)(msg);
    return setLength(msg, length, false);
};
exports.setLengthLeft = setLengthLeft;
/**
 * Right Pads a `Buffer` with trailing zeros till it has `length` bytes.
 * it truncates the end if it exceeds.
 * @param msg the value to pad (Buffer)
 * @param length the number of bytes the output should be
 * @return (Buffer)
 */
const setLengthRight = function (msg, length) {
    (0, helpers_1.assertIsBuffer)(msg);
    return setLength(msg, length, true);
};
exports.setLengthRight = setLengthRight;
/**
 * Trims leading zeros from a `Buffer`, `String` or `Number[]`.
 * @param a (Buffer|Array|String)
 * @return (Buffer|Array|String)
 */
const stripZeros = function (a) {
    let first = a[0];
    while (a.length > 0 && first.toString() === '0') {
        a = a.slice(1);
        first = a[0];
    }
    return a;
};
/**
 * Trims leading zeros from a `Buffer`.
 * @param a (Buffer)
 * @return (Buffer)
 */
const unpadBuffer = function (a) {
    (0, helpers_1.assertIsBuffer)(a);
    return stripZeros(a);
};
exports.unpadBuffer = unpadBuffer;
/**
 * Trims leading zeros from an `Array` (of numbers).
 * @param a (number[])
 * @return (number[])
 */
const unpadArray = function (a) {
    (0, helpers_1.assertIsArray)(a);
    return stripZeros(a);
};
exports.unpadArray = unpadArray;
/**
 * Trims leading zeros from a hex-prefixed `String`.
 * @param a (String)
 * @return (String)
 */
const unpadHexString = function (a) {
    (0, helpers_1.assertIsHexString)(a);
    a = (0, internal_1.stripHexPrefix)(a);
    return ('0x' + stripZeros(a));
};
exports.unpadHexString = unpadHexString;
/**
 * Attempts to turn a value into a `Buffer`.
 * Inputs supported: `Buffer`, `String` (hex-prefixed), `Number`, null/undefined, `BigInt` and other objects
 * with a `toArray()` or `toBuffer()` method.
 * @param v the value
 */
const toBuffer = function (v) {
    if (v === null || v === undefined) {
        return Buffer.allocUnsafe(0);
    }
    if (Buffer.isBuffer(v)) {
        return Buffer.from(v);
    }
    if (Array.isArray(v) || v instanceof Uint8Array) {
        return Buffer.from(v);
    }
    if (typeof v === 'string') {
        if (!(0, internal_1.isHexString)(v)) {
            throw new Error(`Cannot convert string to buffer. toBuffer only supports 0x-prefixed hex strings and this string was given: ${v}`);
        }
        return Buffer.from((0, internal_1.padToEven)((0, internal_1.stripHexPrefix)(v)), 'hex');
    }
    if (typeof v === 'number') {
        return (0, exports.intToBuffer)(v);
    }
    if (typeof v === 'bigint') {
        if (v < BigInt(0)) {
            throw new Error(`Cannot convert negative bigint to buffer. Given: ${v}`);
        }
        let n = v.toString(16);
        if (n.length % 2)
            n = '0' + n;
        return Buffer.from(n, 'hex');
    }
    if (v.toArray) {
        // converts a BN to a Buffer
        return Buffer.from(v.toArray());
    }
    if (v.toBuffer) {
        return Buffer.from(v.toBuffer());
    }
    throw new Error('invalid type');
};
exports.toBuffer = toBuffer;
/**
 * Converts a `Buffer` into a `0x`-prefixed hex `String`.
 * @param buf `Buffer` object to convert
 */
const bufferToHex = function (buf) {
    buf = (0, exports.toBuffer)(buf);
    return '0x' + buf.toString('hex');
};
exports.bufferToHex = bufferToHex;
/**
 * Converts a {@link Buffer} to a {@link bigint}
 */
function bufferToBigInt(buf) {
    const hex = (0, exports.bufferToHex)(buf);
    if (hex === '0x') {
        return BigInt(0);
    }
    return BigInt(hex);
}
exports.bufferToBigInt = bufferToBigInt;
/**
 * Converts a {@link bigint} to a {@link Buffer}
 */
function bigIntToBuffer(num) {
    return (0, exports.toBuffer)('0x' + num.toString(16));
}
exports.bigIntToBuffer = bigIntToBuffer;
/**
 * Converts a `Buffer` to a `Number`.
 * @param buf `Buffer` object to convert
 * @throws If the input number exceeds 53 bits.
 */
const bufferToInt = function (buf) {
    const res = Number(bufferToBigInt(buf));
    if (!Number.isSafeInteger(res))
        throw new Error('Number exceeds 53 bits');
    return res;
};
exports.bufferToInt = bufferToInt;
const hexToBytes = (hex) => {
    if (typeof hex !== 'string') {
        throw new Error(`hex argument type ${typeof hex} must be of type string`);
    }
    if (!hex.startsWith('0x')) {
        throw new Error(`prefixed hex input should start with 0x, got ${hex.substring(0, 2)}`);
    }
    hex = hex.slice(2);
    if (hex.length % 2 !== 0) {
        hex = (0, internal_1.padToEven)(hex);
    }
    const byteLen = hex.length / 2;
    const bytes = new Uint8Array(byteLen);
    for (let i = 0; i < byteLen; i++) {
        const byte = parseInt(hex.slice(i * 2, (i + 1) * 2), 16);
        bytes[i] = byte;
    }
    return bytes;
};
exports.hexToBytes = hexToBytes;
/**
 * Converts an {@link number} to a {@link Uint8Array}
 * @param {Number} i
 * @return {Uint8Array}
 */
const intToBytes = (i) => {
    const hex = (0, exports.intToHex)(i);
    return (0, exports.hexToBytes)(hex);
};
exports.intToBytes = intToBytes;
const _unprefixedHexToBytes = (data) => {
    const hex = data.startsWith('0x') ? data.substring(2) : data;
    if (typeof hex !== 'string')
        throw new Error('hex string expected, got ' + typeof hex);
    const len = hex.length;
    if (len % 2)
        throw new Error('padded hex string expected, got unpadded hex of length ' + len);
    const array = new Uint8Array(len / 2);
    for (let i = 0; i < array.length; i++) {
        const j = i * 2;
        const hexByte = hex.slice(j, j + 2);
        const byte = Number.parseInt(hexByte, 16);
        if (Number.isNaN(byte) || byte < 0)
            throw new Error('Invalid byte sequence');
        array[i] = byte;
    }
    return array;
};
const unprefixedHexToBytes = (inp) => {
    if (inp.slice(0, 2) === '0x') {
        throw new Error('hex string is prefixed with 0x, should be unprefixed');
    }
    else {
        return _unprefixedHexToBytes((0, internal_1.padToEven)(inp));
    }
};
exports.unprefixedHexToBytes = unprefixedHexToBytes;
/**
 * Attempts to turn a value into a `Uint8Array`.
 * Inputs supported: `Buffer`, `Uint8Array`, `String` (hex-prefixed), `Number`, null/undefined, `BigInt` and other objects
 * with a `toArray()` or `toBytes()` method.
 * @param {ToBytesInputTypes} v the value
 * @return {Uint8Array}
 */
const toBytes = (v) => {
    if (v === null || v === undefined) {
        return new Uint8Array();
    }
    if (Array.isArray(v) || v instanceof Uint8Array) {
        return Uint8Array.from(v);
    }
    if (typeof v === 'string') {
        if (!(0, internal_1.isHexString)(v)) {
            throw new Error(`Cannot convert string to Uint8Array. toBytes only supports 0x-prefixed hex strings and this string was given: ${v}`);
        }
        return (0, exports.hexToBytes)(v);
    }
    if (typeof v === 'number') {
        return (0, exports.intToBytes)(v);
    }
    if (typeof v === 'bigint') {
        if (v < BigInt(0)) {
            throw new Error(`Cannot convert negative bigint to Uint8Array. Given: ${v}`);
        }
        let n = v.toString(16);
        if (n.length % 2)
            n = '0' + n;
        return (0, exports.unprefixedHexToBytes)(n);
    }
    if (v.toBytes !== undefined) {
        // converts a `TransformableToBytes` object to a Uint8Array
        return v.toBytes();
    }
    throw new Error('invalid type');
};
exports.toBytes = toBytes;
/**
 * Interprets a `Buffer` as a signed integer and returns a `BigInt`. Assumes 256-bit numbers.
 * @param num Signed integer value
 */
const fromSigned = function (num) {
    return BigInt.asIntN(256, bufferToBigInt(num));
};
exports.fromSigned = fromSigned;
/**
 * Converts a `BigInt` to an unsigned integer and returns it as a `Buffer`. Assumes 256-bit numbers.
 * @param num
 */
const toUnsigned = function (num) {
    return bigIntToBuffer(BigInt.asUintN(256, num));
};
exports.toUnsigned = toUnsigned;
/**
 * Adds "0x" to a given `String` if it does not already start with "0x".
 */
const addHexPrefix = function (str) {
    if (typeof str !== 'string') {
        return str;
    }
    return (0, internal_1.isHexPrefixed)(str) ? str : '0x' + str;
};
exports.addHexPrefix = addHexPrefix;
/**
 * Shortens a string  or buffer's hex string representation to maxLength (default 50).
 *
 * Examples:
 *
 * Input:  '657468657265756d000000000000000000000000000000000000000000000000'
 * Output: '657468657265756d0000000000000000000000000000000000…'
 */
function short(buffer, maxLength = 50) {
    const bufferStr = Buffer.isBuffer(buffer) ? buffer.toString('hex') : buffer;
    if (bufferStr.length <= maxLength) {
        return bufferStr;
    }
    return bufferStr.slice(0, maxLength) + '…';
}
exports.short = short;
/**
 * Returns the utf8 string representation from a hex string.
 *
 * Examples:
 *
 * Input 1: '657468657265756d000000000000000000000000000000000000000000000000'
 * Input 2: '657468657265756d'
 * Input 3: '000000000000000000000000000000000000000000000000657468657265756d'
 *
 * Output (all 3 input variants): 'ethereum'
 *
 * Note that this method is not intended to be used with hex strings
 * representing quantities in both big endian or little endian notation.
 *
 * @param string Hex string, should be `0x` prefixed
 * @return Utf8 string
 */
const toUtf8 = function (hex) {
    const zerosRegexp = /^(00)+|(00)+$/g;
    hex = (0, internal_1.stripHexPrefix)(hex);
    if (hex.length % 2 !== 0) {
        throw new Error('Invalid non-even hex string input for toUtf8() provided');
    }
    const bufferVal = Buffer.from(hex.replace(zerosRegexp, ''), 'hex');
    return bufferVal.toString('utf8');
};
exports.toUtf8 = toUtf8;
/**
 * Converts a `Buffer` or `Array` to JSON.
 * @param ba (Buffer|Array)
 * @return (Array|String|null)
 */
const baToJSON = function (ba) {
    if (Buffer.isBuffer(ba)) {
        return `0x${ba.toString('hex')}`;
    }
    else if (ba instanceof Array) {
        const array = [];
        for (let i = 0; i < ba.length; i++) {
            array.push((0, exports.baToJSON)(ba[i]));
        }
        return array;
    }
};
exports.baToJSON = baToJSON;
/**
 * Checks provided Buffers for leading zeroes and throws if found.
 *
 * Examples:
 *
 * Valid values: 0x1, 0x, 0x01, 0x1234
 * Invalid values: 0x0, 0x00, 0x001, 0x0001
 *
 * Note: This method is useful for validating that RLP encoded integers comply with the rule that all
 * integer values encoded to RLP must be in the most compact form and contain no leading zero bytes
 * @param values An object containing string keys and Buffer values
 * @throws if any provided value is found to have leading zero bytes
 */
const validateNoLeadingZeroes = function (values) {
    for (const [k, v] of Object.entries(values)) {
        if (v !== undefined && v.length > 0 && v[0] === 0) {
            throw new Error(`${k} cannot have leading zeroes, received: ${v.toString('hex')}`);
        }
    }
};
exports.validateNoLeadingZeroes = validateNoLeadingZeroes;
function arrToBufArr(arr) {
    if (!Array.isArray(arr)) {
        return Buffer.from(arr);
    }
    return arr.map((a) => arrToBufArr(a));
}
exports.arrToBufArr = arrToBufArr;
function bufArrToArr(arr) {
    if (!Array.isArray(arr)) {
        return Uint8Array.from(arr ?? []);
    }
    return arr.map((a) => bufArrToArr(a));
}
exports.bufArrToArr = bufArrToArr;
/**
 * Converts a {@link bigint} to a `0x` prefixed hex string
 */
const bigIntToHex = (num) => {
    return '0x' + num.toString(16);
};
exports.bigIntToHex = bigIntToHex;
/**
 * Convert value from bigint to an unpadded Buffer
 * (useful for RLP transport)
 * @param value value to convert
 */
function bigIntToUnpaddedBuffer(value) {
    return (0, exports.unpadBuffer)(bigIntToBuffer(value));
}
exports.bigIntToUnpaddedBuffer = bigIntToUnpaddedBuffer;
function intToUnpaddedBuffer(value) {
    return (0, exports.unpadBuffer)((0, exports.intToBuffer)(value));
}
exports.intToUnpaddedBuffer = intToUnpaddedBuffer;
/****************  Borrowed from @chainsafe/ssz */
// Caching this info costs about ~1000 bytes and speeds up toHexString() by x6
const hexByByte = Array.from({ length: 256 }, (v, i) => i.toString(16).padStart(2, '0'));
const bytesToHex = (bytes) => {
    let hex = '0x';
    if (bytes === undefined || bytes.length === 0)
        return hex;
    for (const byte of bytes) {
        hex += hexByByte[byte];
    }
    return hex;
};
exports.bytesToHex = bytesToHex;
const bytesToBigInt = (bytes) => {
    const hex = (0, exports.bytesToHex)(bytes);
    if (hex === '0x') {
        return BigInt(0);
    }
    return BigInt(hex);
};
exports.bytesToBigInt = bytesToBigInt;
//# sourceMappingURL=bytes.js.map