// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "../bridge/BlobstreamO.sol";

contract BlobstreamOTestnet is BlobstreamO {
    constructor(address _guardian) BlobstreamO(_guardian) {}

     /// @notice This function is called by the guardian to reset the validator set
    /// on testnet. Not to be used on mainnet.
    /// @param _powerThreshold Amount of voting power needed to approve operations.
    /// @param _validatorTimestamp The timestamp of the block where validator set is updated.
    /// @param _validatorSetCheckpoint The hash of the validator set.
    function guardianResetValidatorSetTestnet(
        uint256 _powerThreshold,
        uint256 _validatorTimestamp,
        bytes32 _validatorSetCheckpoint
    ) external {
        if (msg.sender != guardian) {
            revert NotGuardian();
        }
        powerThreshold = _powerThreshold;
        validatorTimestamp = _validatorTimestamp;
        lastValidatorSetCheckpoint = _validatorSetCheckpoint;
    }
}