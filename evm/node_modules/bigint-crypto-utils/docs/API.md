# bigint-crypto-utils - v3.3.0

## Table of contents

### Interfaces

- [Egcd](interfaces/Egcd.md)

### Type Aliases

- [PrimeFactor](API.md#primefactor)
- [PrimeFactorization](API.md#primefactorization)
- [PrimePower](API.md#primepower)

### Functions

- [abs](API.md#abs)
- [bitLength](API.md#bitlength)
- [crt](API.md#crt)
- [eGcd](API.md#egcd)
- [gcd](API.md#gcd)
- [isProbablyPrime](API.md#isprobablyprime)
- [lcm](API.md#lcm)
- [max](API.md#max)
- [min](API.md#min)
- [modAdd](API.md#modadd)
- [modInv](API.md#modinv)
- [modMultiply](API.md#modmultiply)
- [modPow](API.md#modpow)
- [phi](API.md#phi)
- [prime](API.md#prime)
- [primeSync](API.md#primesync)
- [randBetween](API.md#randbetween)
- [randBits](API.md#randbits)
- [randBitsSync](API.md#randbitssync)
- [randBytes](API.md#randbytes)
- [randBytesSync](API.md#randbytessync)
- [toZn](API.md#tozn)

## Type Aliases

### PrimeFactor

Ƭ **PrimeFactor**: `number` \| `bigint` \| [`PrimePower`](API.md#primepower)

___

### PrimeFactorization

Ƭ **PrimeFactorization**: [`bigint`, `bigint`][]

___

### PrimePower

Ƭ **PrimePower**: [`number` \| `bigint`, `number` \| `bigint`]

## Functions

### abs

▸ **abs**(`a`): `number` \| `bigint`

Absolute value. abs(a)==a if a>=0. abs(a)==-a if a<0

#### Parameters

| Name | Type |
| :------ | :------ |
| `a` | `number` \| `bigint` |

#### Returns

`number` \| `bigint`

The absolute value of a

___

### bitLength

▸ **bitLength**(`a`): `number`

Returns the (minimum) length of a number expressed in bits.

#### Parameters

| Name | Type |
| :------ | :------ |
| `a` | `number` \| `bigint` |

#### Returns

`number`

The bit length

___

### crt

▸ **crt**(`remainders`, `modulos`, `modulo?`): `bigint`

Chinese remainder theorem states that if one knows the remainders of the Euclidean division of an integer n by several integers, then one can determine uniquely the remainder of the division of n by the product of these integers, under the condition that the divisors are pairwise coprime (no two divisors share a common factor other than 1). Provided that n_i are pairwise coprime, and a_i any integers, this function returns a solution for the following system of equations:
   x ≡ a_1 mod n_1
   x ≡ a_2 mod n_2
   ⋮
   x ≡ a_k mod n_k

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `remainders` | `bigint`[] | the array of remainders a_i. For example [17n, 243n, 344n] |
| `modulos` | `bigint`[] | the array of modulos n_i. For example [769n, 2017n, 47701n] |
| `modulo?` | `bigint` | the product of all modulos. Provided here just to save some operations if it is already known |

#### Returns

`bigint`

x

___

### eGcd

▸ **eGcd**(`a`, `b`): [`Egcd`](interfaces/Egcd.md)

An iterative implementation of the extended euclidean algorithm or extended greatest common divisor algorithm.
Take positive integers a, b as input, and return a triple (g, x, y), such that ax + by = g = gcd(a, b).

**`Throws`**

RangeError if a or b are <= 0

#### Parameters

| Name | Type |
| :------ | :------ |
| `a` | `number` \| `bigint` |
| `b` | `number` \| `bigint` |

#### Returns

[`Egcd`](interfaces/Egcd.md)

A triple (g, x, y), such that ax + by = g = gcd(a, b).

___

### gcd

▸ **gcd**(`a`, `b`): `bigint`

Greatest common divisor of two integers based on the iterative binary algorithm.

#### Parameters

| Name | Type |
| :------ | :------ |
| `a` | `number` \| `bigint` |
| `b` | `number` \| `bigint` |

#### Returns

`bigint`

The greatest common divisor of a and b

___

### isProbablyPrime

▸ **isProbablyPrime**(`w`, `iterations?`, `disableWorkers?`): `Promise`<`boolean`\>

The test first tries if any of the first 250 small primes are a factor of the input number and then passes several
iterations of Miller-Rabin Probabilistic Primality Test (FIPS 186-4 C.3.1)

**`Throws`**

RangeError if w<0

#### Parameters

| Name | Type | Default value | Description |
| :------ | :------ | :------ | :------ |
| `w` | `number` \| `bigint` | `undefined` | A positive integer to be tested for primality |
| `iterations` | `number` | `16` | The number of iterations for the primality test. The value shall be consistent with Table C.1, C.2 or C.3 of FIPS 186-4 |
| `disableWorkers` | `boolean` | `false` | Disable the use of workers for the primality test |

#### Returns

`Promise`<`boolean`\>

A promise that resolves to a boolean that is either true (a probably prime number) or false (definitely composite)

___

### lcm

▸ **lcm**(`a`, `b`): `bigint`

The least common multiple computed as abs(a*b)/gcd(a,b)

#### Parameters

| Name | Type |
| :------ | :------ |
| `a` | `number` \| `bigint` |
| `b` | `number` \| `bigint` |

#### Returns

`bigint`

The least common multiple of a and b

___

### max

▸ **max**(`a`, `b`): `number` \| `bigint`

Maximum. max(a,b)==a if a>=b. max(a,b)==b if a<b

#### Parameters

| Name | Type |
| :------ | :------ |
| `a` | `number` \| `bigint` |
| `b` | `number` \| `bigint` |

#### Returns

`number` \| `bigint`

Maximum of numbers a and b

___

### min

▸ **min**(`a`, `b`): `number` \| `bigint`

Minimum. min(a,b)==b if a>=b. min(a,b)==a if a<b

#### Parameters

| Name | Type |
| :------ | :------ |
| `a` | `number` \| `bigint` |
| `b` | `number` \| `bigint` |

#### Returns

`number` \| `bigint`

Minimum of numbers a and b

___

### modAdd

▸ **modAdd**(`addends`, `n`): `bigint`

Modular addition of (a_1 + ... + a_r) mod n

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `addends` | (`number` \| `bigint`)[] | an array of the numbers a_i to add. For example [3, 12353251235n, 1243, -12341232545990n] |
| `n` | `number` \| `bigint` | the modulo |

#### Returns

`bigint`

The smallest positive integer that is congruent with (a_1 + ... + a_r) mod n

___

### modInv

▸ **modInv**(`a`, `n`): `bigint`

Modular inverse.

**`Throws`**

RangeError if a does not have inverse modulo n

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `a` | `number` \| `bigint` | The number to find an inverse for |
| `n` | `number` \| `bigint` | The modulo |

#### Returns

`bigint`

The inverse modulo n

___

### modMultiply

▸ **modMultiply**(`factors`, `n`): `bigint`

Modular addition of (a_1 * ... * a_r) mod n
 *

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `factors` | (`number` \| `bigint`)[] | an array of the numbers a_i to multiply. For example [3, 12353251235n, 1243, -12341232545990n] * |
| `n` | `number` \| `bigint` | the modulo * |

#### Returns

`bigint`

The smallest positive integer that is congruent with (a_1 * ... * a_r) mod n

___

### modPow

▸ **modPow**(`b`, `e`, `n`, `primeFactorization?`): `bigint`

Modular exponentiation b**e mod n. Currently using the right-to-left binary method if the prime factorization is not provided, or the chinese remainder theorem otherwise.

**`Throws`**

RangeError if n <= 0

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `b` | `number` \| `bigint` | base |
| `e` | `number` \| `bigint` | exponent |
| `n` | `number` \| `bigint` | modulo |
| `primeFactorization?` | [`PrimeFactor`](API.md#primefactor)[] | an array of the prime factors, for example [5n, 5n, 13n, 27n], or prime powers as [p, k], for instance [[5, 2], [13, 1], [27, 1]]. If the prime factorization is provided the chinese remainder theorem is used to greatly speed up the exponentiation. |

#### Returns

`bigint`

b**e mod n

___

### phi

▸ **phi**(`primeFactorization`): `bigint`

A function that computes the Euler's totien function of a number n, whose prime power factorization is known

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `primeFactorization` | [`PrimeFactorization`](API.md#primefactorization) | an array of arrays containing the prime power factorization of a number n. For example, for n = (p1**k1)*(p2**k2)*...*(pr**kr), one should provide [[p1, k1], [p2, k2], ... , [pr, kr]] |

#### Returns

`bigint`

phi((p1**k1)*(p2**k2)*...*(pr**kr))

___

### prime

▸ **prime**(`bitLength`, `iterations?`): `Promise`<`bigint`\>

A probably-prime (Miller-Rabin), cryptographically-secure, random-number generator.
The browser version uses web workers to parallelise prime look up. Therefore, it does not lock the UI
main process, and it can be much faster (if several cores or cpu are available).
The node version can also use worker_threads if they are available (enabled by default with Node 11 and
and can be enabled at runtime executing node --experimental-worker with node >=10.5.0).

**`Throws`**

RangeError if bitLength < 1

#### Parameters

| Name | Type | Default value | Description |
| :------ | :------ | :------ | :------ |
| `bitLength` | `number` | `undefined` | The required bit length for the generated prime |
| `iterations` | `number` | `16` | The number of iterations for the Miller-Rabin Probabilistic Primality Test |

#### Returns

`Promise`<`bigint`\>

A promise that resolves to a bigint probable prime of bitLength bits.

___

### primeSync

▸ **primeSync**(`bitLength`, `iterations?`): `bigint`

A probably-prime (Miller-Rabin), cryptographically-secure, random-number generator.
The sync version is NOT RECOMMENDED since it won't use workers and thus it'll be slower and may freeze thw window in browser's javascript. Please consider using prime() instead.

**`Throws`**

RangeError if bitLength < 1

#### Parameters

| Name | Type | Default value | Description |
| :------ | :------ | :------ | :------ |
| `bitLength` | `number` | `undefined` | The required bit length for the generated prime |
| `iterations` | `number` | `16` | The number of iterations for the Miller-Rabin Probabilistic Primality Test |

#### Returns

`bigint`

A bigint probable prime of bitLength bits.

___

### randBetween

▸ **randBetween**(`max`, `min?`): `bigint`

Returns a cryptographically secure random integer between [min,max].

**`Throws`**

RangeError if max <= min

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `max` | `bigint` | Returned value will be <= max |
| `min` | `bigint` | Returned value will be >= min |

#### Returns

`bigint`

A cryptographically secure random bigint between [min,max]

___

### randBits

▸ **randBits**(`bitLength`, `forceLength?`): `Promise`<`Uint8Array` \| `Buffer`\>

Secure random bits for both node and browsers. Node version uses crypto.randomFill() and browser one self.crypto.getRandomValues()

**`Throws`**

RangeError if bitLength < 1

#### Parameters

| Name | Type | Default value | Description |
| :------ | :------ | :------ | :------ |
| `bitLength` | `number` | `undefined` | The desired number of random bits |
| `forceLength` | `boolean` | `false` | Set to true if you want to force the output to have a specific bit length. It basically forces the msb to be 1 |

#### Returns

`Promise`<`Uint8Array` \| `Buffer`\>

A Promise that resolves to a UInt8Array/Buffer (Browser/Node.js) filled with cryptographically secure random bits

___

### randBitsSync

▸ **randBitsSync**(`bitLength`, `forceLength?`): `Uint8Array` \| `Buffer`

Secure random bits for both node and browsers. Node version uses crypto.randomFill() and browser one self.crypto.getRandomValues()

**`Throws`**

RangeError if bitLength < 1

#### Parameters

| Name | Type | Default value | Description |
| :------ | :------ | :------ | :------ |
| `bitLength` | `number` | `undefined` | The desired number of random bits |
| `forceLength` | `boolean` | `false` | Set to true if you want to force the output to have a specific bit length. It basically forces the msb to be 1 |

#### Returns

`Uint8Array` \| `Buffer`

A Uint8Array/Buffer (Browser/Node.js) filled with cryptographically secure random bits

___

### randBytes

▸ **randBytes**(`byteLength`, `forceLength?`): `Promise`<`Uint8Array` \| `Buffer`\>

Secure random bytes for both node and browsers. Node version uses crypto.randomBytes() and browser one self.crypto.getRandomValues()

**`Throws`**

RangeError if byteLength < 1

#### Parameters

| Name | Type | Default value | Description |
| :------ | :------ | :------ | :------ |
| `byteLength` | `number` | `undefined` | The desired number of random bytes |
| `forceLength` | `boolean` | `false` | Set to true if you want to force the output to have a bit length of 8*byteLength. It basically forces the msb to be 1 |

#### Returns

`Promise`<`Uint8Array` \| `Buffer`\>

A promise that resolves to a UInt8Array/Buffer (Browser/Node.js) filled with cryptographically secure random bytes

___

### randBytesSync

▸ **randBytesSync**(`byteLength`, `forceLength?`): `Uint8Array` \| `Buffer`

Secure random bytes for both node and browsers. Node version uses crypto.randomFill() and browser one self.crypto.getRandomValues()
This is the synchronous version, consider using the asynchronous one for improved efficiency.

**`Throws`**

RangeError if byteLength < 1

#### Parameters

| Name | Type | Default value | Description |
| :------ | :------ | :------ | :------ |
| `byteLength` | `number` | `undefined` | The desired number of random bytes |
| `forceLength` | `boolean` | `false` | Set to true if you want to force the output to have a bit length of 8*byteLength. It basically forces the msb to be 1 |

#### Returns

`Uint8Array` \| `Buffer`

A UInt8Array/Buffer (Browser/Node.js) filled with cryptographically secure random bytes

___

### toZn

▸ **toZn**(`a`, `n`): `bigint`

Finds the smallest positive element that is congruent to a in modulo n

**`Remarks`**

a and b must be the same type, either number or bigint

**`Throws`**

RangeError if n <= 0

#### Parameters

| Name | Type | Description |
| :------ | :------ | :------ |
| `a` | `number` \| `bigint` | An integer |
| `n` | `number` \| `bigint` | The modulo |

#### Returns

`bigint`

A bigint with the smallest positive representation of a modulo n
