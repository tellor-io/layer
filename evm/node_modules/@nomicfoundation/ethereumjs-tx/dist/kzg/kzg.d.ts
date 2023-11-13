import type { Kzg } from '../depInterfaces';
export declare let kzg: Kzg;
/**
 * @param kzgLib a KZG implementation (defaults to c-kzg)
 * @param trustedSetupPath the full path (e.g. "/home/linux/devnet4.txt") to a kzg trusted setup text file
 */
export declare function initKZG(kzgLib: Kzg, trustedSetupPath: string): void;
//# sourceMappingURL=kzg.d.ts.map