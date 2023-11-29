// SPDX-License-Identifier: MIT
pragma solidity 0.7.4;

/**
 @author Tellor Inc.
 @title Utilities
 @dev Functions for retrieving min and Max in 51 length array (requestQ)
 *Taken partly from: https://github.com/modular-network/ethereum-libraries-array-utils/blob/master/contracts/Array256Lib.sol
*/
contract Utilities {
    /**
     * @dev This is an internal function called by updateOnDeck that gets the top 5 values
     * @param data is an array [51] to determine the top 5 values from
     * @return max the top 5 values and their index values in the data array
     */
    function _getMax5(uint256[51] memory data)
        internal
        pure
        returns (uint256[5] memory max, uint256[5] memory maxIndex)
    {
        uint256 min5 = data[1];
        uint256 minI = 0;
        for (uint256 j = 0; j < 5; j++) {
            max[j] = data[j + 1]; //max[0]=data[1]
            maxIndex[j] = j + 1; //maxIndex[0]= 1
            if (max[j] < min5) {
                min5 = max[j];
                minI = j;
            }
        }
        for (uint256 i = 6; i < data.length; i++) {
            if (data[i] > min5) {
                max[minI] = data[i];
                maxIndex[minI] = i;
                min5 = data[i];
                for (uint256 j = 0; j < 5; j++) {
                    if (max[j] < min5) {
                        min5 = max[j];
                        minI = j;
                    }
                }
            }
        }
    }
}
