import { ByteVectorType, ContainerType, ListCompositeType, UintBigintType, UintNumberType } from '@chainsafe/ssz';
export declare const UintNum64: UintNumberType;
export declare const UintBigInt64: UintBigintType;
export declare const Bytes20: ByteVectorType;
export declare const Withdrawal: ContainerType<{
    index: UintBigintType;
    validatorIndex: UintBigintType;
    address: ByteVectorType;
    amount: UintBigintType;
}>;
export declare const Withdrawals: ListCompositeType<ContainerType<{
    index: UintBigintType;
    validatorIndex: UintBigintType;
    address: ByteVectorType;
    amount: UintBigintType;
}>>;
//# sourceMappingURL=ssz.d.ts.map