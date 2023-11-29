// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract MappingContractExample{
    function getTellorID(bytes32 _id) external pure returns(bytes32){
        if (
            _id ==
            0xdfaa6f747f0f012e8f2069d6ecacff25f5cdf0258702051747439949737fc0b5
        ) {
            bytes memory _queryData = abi.encode(
                "SpotPrice",
                abi.encode("eth", "usd")
            );
            _id = keccak256(_queryData);
        } else if (
            _id ==
            0x637b7efb6b620736c247aaa282f3898914c0bef6c12faff0d3fe9d4bea783020
        ) {
            bytes memory _queryData = abi.encode(
                "SpotPrice",
                abi.encode("btc", "usd")
            );
            _id = keccak256(_queryData);
        } else if (
            _id ==
            0x2dfb033e1ae0529b328985942d27f2d5a62213f3a2d97ca8e27ad2864c5af942
        ) {
            bytes memory _queryData = abi.encode(
                "SpotPrice",
                abi.encode("xau", "usd")
            );
            _id = keccak256(_queryData);
        } else if (
            _id ==
            0x9899e35601719f1348e09967349f72c7d04800f17c14992d6dcf2f17fac713ea
        ) {
            bytes memory _queryData = abi.encode(
                "SpotPrice",
                abi.encode("dai", "usd")
            );
            _id = keccak256(_queryData);
        }
        return _id;
    }
}