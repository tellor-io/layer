function n(n){return n>=0?n:-n}function t(n){if("number"==typeof n&&(n=BigInt(n)),1n===n)return 1;let t=1;do{t++;}while((n>>=1n)>1n);return t}function r(n,t){if("number"==typeof n&&(n=BigInt(n)),"number"==typeof t&&(t=BigInt(t)),n<=0n||t<=0n)throw new RangeError("a and b MUST be > 0");let r=0n,e=1n,o=1n,i=0n;for(;0n!==n;){const u=t/n,f=t%n,g=r-o*u,c=e-i*u;t=n,n=f,r=o,e=i,o=g,i=c;}return {g:t,x:r,y:e}}function e(n,t){if("number"==typeof n&&(n=BigInt(n)),"number"==typeof t&&(t=BigInt(t)),t<=0n)throw new RangeError("n must be > 0");const r=n%t;return r<0n?r+t:r}function o(n,t){const o=r(e(n,t),t);if(1n!==o.g)throw new RangeError(`${n.toString()} does not have inverse modulo ${t.toString()}`);return e(o.x,t)}function i(n,t,r){if(n.length!==t.length)throw new RangeError("The remainders and modulos arrays should have the same length");const i=r??t.reduce(((n,t)=>n*t),1n);return t.reduce(((t,r,u)=>{const f=i/r;return e(t+f*o(f,r)%i*n[u]%i,i)}),0n)}function u(t,r){let e="number"==typeof t?BigInt(n(t)):n(t),o="number"==typeof r?BigInt(n(r)):n(r);if(0n===e)return o;if(0n===o)return e;let i=0n;for(;0n===(1n&(e|o));)e>>=1n,o>>=1n,i++;for(;0n===(1n&e);)e>>=1n;do{for(;0n===(1n&o);)o>>=1n;if(e>o){const n=e;e=o,o=n;}o-=e;}while(0n!==o);return e<<i}function f(t,r){return "number"==typeof t&&(t=BigInt(t)),"number"==typeof r&&(r=BigInt(r)),0n===t&&0n===r?BigInt(0):n(t/u(t,r)*r)}function g(n,t){return n>=t?n:t}function c(n,t){return n>=t?t:n}function m(n,t){const r=BigInt(t);return e(n.map((n=>BigInt(n)%r)).reduce(((n,t)=>n+t%r),0n),r)}function p(n,t){const r=BigInt(t);return e(n.map((n=>BigInt(n)%r)).reduce(((n,t)=>n*t%r),1n),r)}function a(n){return n.map((n=>n[0]**(n[1]-1n)*(n[0]-1n))).reduce(((n,t)=>t*n),1n)}function s(t,r,u,f){if("number"==typeof t&&(t=BigInt(t)),"number"==typeof r&&(r=BigInt(r)),"number"==typeof u&&(u=BigInt(u)),u<=0n)throw new RangeError("n must be > 0");if(1n===u)return 0n;if(t=e(t,u),r<0n)return o(s(t,n(r),u,f),u);if(void 0!==f)return function(n,t,r,e){const o=e.map((n=>n[0]**n[1])),u=e.map((n=>a([n]))),f=u.map(((r,e)=>s(n,t%r,o[e])));return i(f,o,r)}(t,r,u,function(n){const t={};return n.forEach((n=>{if("bigint"==typeof n||"number"==typeof n){const r=String(n);void 0===t[r]?t[r]={p:BigInt(n),k:1n}:t[r].k+=1n;}else {const r=String(n[0]);void 0===t[r]?t[r]={p:BigInt(n[0]),k:BigInt(n[1])}:t[r].k+=BigInt(n[1]);}})),Object.values(t).map((n=>[n.p,n.k]))}(f));let g=1n;for(;r>0;)r%2n===1n&&(g=g*t%u),r/=2n,t=t**2n%u;return g}

function fromBuffer(buf) {
    let ret = 0n;
    for (const i of buf.values()) {
        const bi = BigInt(i);
        ret = (ret << 8n) + bi;
    }
    return ret;
}

function randBytes(byteLength, forceLength = false) {
    if (byteLength < 1)
        throw new RangeError('byteLength MUST be > 0');
    return new Promise(function (resolve, reject) {
        {
            const buf = new Uint8Array(byteLength);
            if (byteLength <= 65536) {
                self.crypto.getRandomValues(buf);
            }
            else {
                for (let i = 0; i < Math.ceil(byteLength / 65536); i++) {
                    const begin = i * 65536;
                    const end = ((begin + 65535) < byteLength) ? begin + 65535 : byteLength - 1;
                    self.crypto.getRandomValues(buf.subarray(begin, end));
                }
            }
            if (forceLength)
                buf[0] = buf[0] | 128;
            resolve(buf);
        }
    });
}
function randBytesSync(byteLength, forceLength = false) {
    if (byteLength < 1)
        throw new RangeError('byteLength MUST be > 0');
    {
        const buf = new Uint8Array(byteLength);
        if (byteLength <= 65536) {
            self.crypto.getRandomValues(buf);
        }
        else {
            for (let i = 0; i < Math.ceil(byteLength / 65536); i++) {
                const begin = i * 65536;
                const end = ((begin + 65535) < byteLength) ? begin + 65535 : byteLength - 1;
                self.crypto.getRandomValues(buf.subarray(begin, end));
            }
        }
        if (forceLength)
            buf[0] = buf[0] | 128;
        return buf;
    }
}

function randBits(bitLength, forceLength = false) {
    if (bitLength < 1)
        throw new RangeError('bitLength MUST be > 0');
    const byteLength = Math.ceil(bitLength / 8);
    const bitLengthMod8 = bitLength % 8;
    return new Promise((resolve, reject) => {
        randBytes(byteLength, false).then(function (rndBytes) {
            if (bitLengthMod8 !== 0) {
                rndBytes[0] = rndBytes[0] & (2 ** bitLengthMod8 - 1);
            }
            if (forceLength) {
                const mask = (bitLengthMod8 !== 0) ? 2 ** (bitLengthMod8 - 1) : 128;
                rndBytes[0] = rndBytes[0] | mask;
            }
            resolve(rndBytes);
        });
    });
}
function randBitsSync(bitLength, forceLength = false) {
    if (bitLength < 1)
        throw new RangeError('bitLength MUST be > 0');
    const byteLength = Math.ceil(bitLength / 8);
    const rndBytes = randBytesSync(byteLength, false);
    const bitLengthMod8 = bitLength % 8;
    if (bitLengthMod8 !== 0) {
        rndBytes[0] = rndBytes[0] & (2 ** bitLengthMod8 - 1);
    }
    if (forceLength) {
        const mask = (bitLengthMod8 !== 0) ? 2 ** (bitLengthMod8 - 1) : 128;
        rndBytes[0] = rndBytes[0] | mask;
    }
    return rndBytes;
}

function randBetween(max, min = 1n) {
    if (max <= min)
        throw new RangeError('Arguments MUST be: max > min');
    const interval = max - min;
    const bitLen = t(interval);
    let rnd;
    do {
        const buf = randBitsSync(bitLen);
        rnd = fromBuffer(buf);
    } while (rnd > interval);
    return rnd + min;
}

function _workerUrl(workerCode) {
    workerCode = `(() => {${workerCode}})()`;
    const _blob = new Blob([workerCode], { type: 'text/javascript' });
    return window.URL.createObjectURL(_blob);
}
let _useWorkers = false;
{
    if (self.Worker !== undefined)
        _useWorkers = true;
}

function isProbablyPrime(w, iterations = 16, disableWorkers = false) {
    if (typeof w === 'number') {
        w = BigInt(w);
    }
    if (w < 0n)
        throw RangeError('w MUST be >= 0');
    {
        return new Promise((resolve, reject) => {
            const worker = new Worker(_isProbablyPrimeWorkerUrl());
            worker.onmessage = (event) => {
                if (event?.data?._bcu?.isPrime !== undefined) {
                    worker.terminate();
                    resolve(event.data._bcu.isPrime);
                }
            };
            worker.onmessageerror = (event) => {
                reject(event);
            };
            const msg = {
                _bcu: {
                    rnd: w,
                    iterations,
                    id: 0
                }
            };
            worker.postMessage(msg);
        });
    }
}
function _isProbablyPrime(w, iterations) {
    if (w === 2n)
        return true;
    else if ((w & 1n) === 0n || w === 1n)
        return false;
    const firstPrimes = [
        3n,
        5n,
        7n,
        11n,
        13n,
        17n,
        19n,
        23n,
        29n,
        31n,
        37n,
        41n,
        43n,
        47n,
        53n,
        59n,
        61n,
        67n,
        71n,
        73n,
        79n,
        83n,
        89n,
        97n,
        101n,
        103n,
        107n,
        109n,
        113n,
        127n,
        131n,
        137n,
        139n,
        149n,
        151n,
        157n,
        163n,
        167n,
        173n,
        179n,
        181n,
        191n,
        193n,
        197n,
        199n,
        211n,
        223n,
        227n,
        229n,
        233n,
        239n,
        241n,
        251n,
        257n,
        263n,
        269n,
        271n,
        277n,
        281n,
        283n,
        293n,
        307n,
        311n,
        313n,
        317n,
        331n,
        337n,
        347n,
        349n,
        353n,
        359n,
        367n,
        373n,
        379n,
        383n,
        389n,
        397n,
        401n,
        409n,
        419n,
        421n,
        431n,
        433n,
        439n,
        443n,
        449n,
        457n,
        461n,
        463n,
        467n,
        479n,
        487n,
        491n,
        499n,
        503n,
        509n,
        521n,
        523n,
        541n,
        547n,
        557n,
        563n,
        569n,
        571n,
        577n,
        587n,
        593n,
        599n,
        601n,
        607n,
        613n,
        617n,
        619n,
        631n,
        641n,
        643n,
        647n,
        653n,
        659n,
        661n,
        673n,
        677n,
        683n,
        691n,
        701n,
        709n,
        719n,
        727n,
        733n,
        739n,
        743n,
        751n,
        757n,
        761n,
        769n,
        773n,
        787n,
        797n,
        809n,
        811n,
        821n,
        823n,
        827n,
        829n,
        839n,
        853n,
        857n,
        859n,
        863n,
        877n,
        881n,
        883n,
        887n,
        907n,
        911n,
        919n,
        929n,
        937n,
        941n,
        947n,
        953n,
        967n,
        971n,
        977n,
        983n,
        991n,
        997n,
        1009n,
        1013n,
        1019n,
        1021n,
        1031n,
        1033n,
        1039n,
        1049n,
        1051n,
        1061n,
        1063n,
        1069n,
        1087n,
        1091n,
        1093n,
        1097n,
        1103n,
        1109n,
        1117n,
        1123n,
        1129n,
        1151n,
        1153n,
        1163n,
        1171n,
        1181n,
        1187n,
        1193n,
        1201n,
        1213n,
        1217n,
        1223n,
        1229n,
        1231n,
        1237n,
        1249n,
        1259n,
        1277n,
        1279n,
        1283n,
        1289n,
        1291n,
        1297n,
        1301n,
        1303n,
        1307n,
        1319n,
        1321n,
        1327n,
        1361n,
        1367n,
        1373n,
        1381n,
        1399n,
        1409n,
        1423n,
        1427n,
        1429n,
        1433n,
        1439n,
        1447n,
        1451n,
        1453n,
        1459n,
        1471n,
        1481n,
        1483n,
        1487n,
        1489n,
        1493n,
        1499n,
        1511n,
        1523n,
        1531n,
        1543n,
        1549n,
        1553n,
        1559n,
        1567n,
        1571n,
        1579n,
        1583n,
        1597n
    ];
    for (let i = 0; i < firstPrimes.length && (firstPrimes[i] <= w); i++) {
        const p = firstPrimes[i];
        if (w === p)
            return true;
        else if (w % p === 0n)
            return false;
    }
    let a = 0n;
    const d = w - 1n;
    let aux = d;
    while (aux % 2n === 0n) {
        aux /= 2n;
        ++a;
    }
    const m = d / (2n ** a);
    do {
        const b = randBetween(d, 2n);
        let z = s(b, m, w);
        if (z === 1n || z === d)
            continue;
        let j = 1;
        while (j < a) {
            z = s(z, 2n, w);
            if (z === d)
                break;
            if (z === 1n)
                return false;
            j++;
        }
        if (z !== d)
            return false;
    } while (--iterations !== 0);
    return true;
}
function _isProbablyPrimeWorkerUrl() {
    let workerCode = `
  'use strict';
  const ${r.name} = ${r.toString()};
  const ${o.name} = ${o.toString()};
  const ${s.name} = ${s.toString()};
  const ${e.name} = ${e.toString()};
  const ${randBitsSync.name} = ${randBitsSync.toString()};
  const ${randBytesSync.name} = ${randBytesSync.toString()};
  const ${randBetween.name} = ${randBetween.toString()};
  const ${isProbablyPrime.name} = ${_isProbablyPrime.toString()};
  ${t.toString()};
  ${fromBuffer.toString()};`;
    workerCode += `
  onmessage = async function(msg) {
    if (msg !== undefined && msg.data !== undefined && msg.data._bcu !== undefined && msg.data._bcu.id !== undefined && msg.data._bcu.iterations !== undefined && msg.data._bcu.rnd !== undefined) {
      const msgToParent = {
        _bcu: {
          isPrime: await ${isProbablyPrime.name}(msg.data._bcu.rnd, msg.data._bcu.iterations),
          value: msg.data._bcu.rnd,
          id: msg.data._bcu.id
        }
      };
      postMessage(msgToParent);
    }
  }`;
    return _workerUrl(workerCode);
}

function prime(bitLength, iterations = 16) {
    if (bitLength < 1)
        throw new RangeError('bitLength MUST be > 0');
    if (!_useWorkers) {
        let rnd = 0n;
        do {
            rnd = fromBuffer(randBitsSync(bitLength, true));
        } while (!_isProbablyPrime(rnd, iterations));
        return new Promise((resolve) => { resolve(rnd); });
    }
    return new Promise((resolve, reject) => {
        const workerList = [];
        const _onmessage = (msg, newWorker) => {
            if (msg._bcu.isPrime) {
                for (let j = 0; j < workerList.length; j++) {
                    workerList[j].terminate();
                }
                while (workerList.length > 0) {
                    workerList.pop();
                }
                resolve(msg._bcu.value);
            }
            else {
                const buf = randBitsSync(bitLength, true);
                const rnd = fromBuffer(buf);
                try {
                    const msgToWorker = {
                        _bcu: {
                            rnd,
                            iterations,
                            id: msg._bcu.id
                        }
                    };
                    newWorker.postMessage(msgToWorker);
                }
                catch (error) {
                }
            }
        };
        {
            const workerURL = _isProbablyPrimeWorkerUrl();
            for (let i = 0; i < self.navigator.hardwareConcurrency - 1; i++) {
                const newWorker = new Worker(workerURL);
                newWorker.onmessage = (event) => _onmessage(event.data, newWorker);
                workerList.push(newWorker);
            }
        }
        for (let i = 0; i < workerList.length; i++) {
            randBits(bitLength, true).then(function (buf) {
                const rnd = fromBuffer(buf);
                workerList[i].postMessage({
                    _bcu: {
                        rnd,
                        iterations,
                        id: i
                    }
                });
            }).catch(reject);
        }
    });
}
function primeSync(bitLength, iterations = 16) {
    if (bitLength < 1)
        throw new RangeError('bitLength MUST be > 0');
    let rnd = 0n;
    do {
        rnd = fromBuffer(randBitsSync(bitLength, true));
    } while (!_isProbablyPrime(rnd, iterations));
    return rnd;
}

export { n as abs, t as bitLength, i as crt, r as eGcd, u as gcd, isProbablyPrime, f as lcm, g as max, c as min, m as modAdd, o as modInv, p as modMultiply, s as modPow, a as phi, prime, primeSync, randBetween, randBits, randBitsSync, randBytes, randBytesSync, e as toZn };
