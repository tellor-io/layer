// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

/// @title RoleManager
/// @notice Role manager for the TokenBridgeV2
contract RoleManager {
    /*Storage*/
    mapping(bytes32 => RoleInfo) public roles;
    mapping(bytes32 => RoleUpdate) public roleUpdateProposals;

    struct RoleInfo {
        address roleAddress;
        uint256 roleUpdateDelay;
    }

    struct RoleUpdate {
        address newAddress;
        uint256 newUpdateDelay;
        uint256 proposalTime;
    }

    /*Events*/
    event RoleUpdateProposed(bytes32 _role, address _newAddress, uint256 _newUpdateDelay);
    event RoleUpdateAccepted(bytes32 _role, address _newAddress);
    event RoleUpdateRejected(bytes32 _role);

    /*Functions*/
    /// @notice constructor
    /// @param _guardian the address of the guardian
    /// @param _guardianUpdateDelay the delay before the guardian update can be accepted
    constructor(address _guardian, uint256 _guardianUpdateDelay) {
        roles[keccak256("MAIN_GUARDIAN")] = RoleInfo({
            roleAddress: _guardian,
            roleUpdateDelay: _guardianUpdateDelay
        });
    }

    /// @notice proposes a role update
    /// @param _role the role to update
    /// @param _newAddress the new address for the role
    /// @param _newUpdateDelay the delay before the role update can be accepted
    function proposeRoleUpdate(bytes32 _role, address _newAddress, uint256 _newUpdateDelay) external {
        _roleRestricted(keccak256("MAIN_GUARDIAN"));
        RoleUpdate storage _roleUpdate = roleUpdateProposals[_role];
        require(_roleUpdate.newAddress == address(0), "RoleManager: Role update already proposed");
        require(_newAddress != address(0), "RoleManager: New address cannot be the zero address");
        _roleUpdate.newAddress = _newAddress;
        _roleUpdate.newUpdateDelay = _newUpdateDelay;
        _roleUpdate.proposalTime = block.timestamp;
        emit RoleUpdateProposed(_role, _newAddress, _newUpdateDelay);
    }

    /// @notice accepts a role update
    /// @param _role the role to update
    function acceptRoleUpdate(bytes32 _role) external {
        _roleRestricted(keccak256("MAIN_GUARDIAN"));
        RoleUpdate storage _roleUpdateProposal = roleUpdateProposals[_role];
        require(_roleUpdateProposal.newAddress != address(0), "RoleManager: Role update not proposed");
        RoleInfo storage _roleInfo = roles[_role];
        require(block.timestamp - _roleUpdateProposal.proposalTime > _roleInfo.roleUpdateDelay, "RoleManager: Role update delay not passed");
        _roleInfo.roleAddress = _roleUpdateProposal.newAddress;
        _roleInfo.roleUpdateDelay = _roleUpdateProposal.newUpdateDelay;
        delete roleUpdateProposals[_role];
        emit RoleUpdateAccepted(_role, _roleUpdateProposal.newAddress);
    }

    /// @notice rejects a role update
    /// @param _role the role to update
    function rejectRoleUpdate(bytes32 _role) external {
        _roleRestricted(keccak256("MAIN_GUARDIAN"));
        RoleUpdate storage _roleUpdateProposal = roleUpdateProposals[_role];
        require(_roleUpdateProposal.newAddress != address(0), "RoleManager: Role update not proposed");
        delete roleUpdateProposals[_role];
        emit RoleUpdateRejected(_role);
    }

    /* Internal Functions */
    /// @notice restricts the function to the role
    /// @param _role the role to restrict the function to
    function _roleRestricted(bytes32 _role) internal view {
        require(msg.sender == roles[_role].roleAddress, "RoleManager: Only role can call this function");
    }
}