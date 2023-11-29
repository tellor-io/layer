// SPDX-License-Identifier: MIT
pragma solidity 0.7.4;

import "../Utilities.sol";
import "../Tellor.sol";

/**
 * @title Utilities Tests
 * @dev These are the getter function for the code functions in the
 * Utility contract
 */
contract UtilitiesTest is Utilities,Tellor{

    /**
     * @dev Constructor to set extension address
     * @param _ext Extension address
    */
    constructor(address _ext) Tellor(_ext){
    }

    /**
     * @dev Gets the top 5 of the array provided
     * @param requests is an array of length 51
     * @return _max _index the top 5 and their respective index within the array
     */
    function testgetMax5(uint256[51] memory requests)
        public
        pure
        returns (uint256[5] memory _max, uint256[5] memory _index)
    {
        (_max, _index) = _getMax5(requests);
    }

    function testGetMin(uint256[51] memory requests) public pure
    returns (uint256 _min, uint256 _index){
        (_min, _index) = _getMin(requests);
    }
}
