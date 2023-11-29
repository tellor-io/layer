// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./BaseToken.sol";
import "./NewTransition.sol";
import "./interfaces/ITellorFlex.sol";

/**
 @author Tellor Inc.
 @title Tellor360
 @dev This is the controller contract which defines the functionality for
 * changing the oracle contract address, as well as minting and migrating tokens
*/
contract Tellor360 is BaseToken, NewTransition {
    // Events
    event NewOracleAddress(address _newOracle, uint256 _timestamp);
    event NewProposedOracleAddress(
        address _newProposedOracle,
        uint256 _timestamp
    );

    // Functions
    /**
     * @dev Constructor used to store new flex oracle address
     * @param _flexAddress is the new oracle contract which will replace the
     * tellorX oracle
     */
    constructor(address _flexAddress) {
        require(_flexAddress != address(0), "oracle address must be non-zero");
        addresses[keccak256("_ORACLE_CONTRACT_FOR_INIT")] = _flexAddress;
    }

    /**
     * @dev Use this function to initiate the contract
     */
    function init() external {
        require(uints[keccak256("_INIT")] == 0, "should only happen once");
        uints[keccak256("_INIT")] = 1;
        // retrieve new oracle address from Tellor360 contract address storage
        NewTransition _newController = NewTransition(
            addresses[_TELLOR_CONTRACT]
        );
        address _flexAddress = _newController.getAddressVars(
            keccak256("_ORACLE_CONTRACT_FOR_INIT")
        );
        //on switch over, require tellorFlex values are over 12 hours old
        //then when we switch, the governance switch can be instantaneous
        bytes32 _id = 0x83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992;
        uint256 _firstTimestamp = IOracle(_flexAddress)
            .getTimestampbyQueryIdandIndex(_id, 0);
        require(
            block.timestamp - _firstTimestamp >= 12 hours,
            "contract should be at least 12 hours old"
        );
        addresses[_ORACLE_CONTRACT] = _flexAddress; //used by Liquity+AMPL for this contract's reads
        //init minting uints (timestamps)
        uints[keccak256("_LAST_RELEASE_TIME_TEAM")] = block.timestamp;
        uints[keccak256("_LAST_RELEASE_TIME_DAO")] = block.timestamp - 12 weeks;
        // transfer dispute fees collected during transition period to team
        _doTransfer(
            addresses[_GOVERNANCE_CONTRACT],
            addresses[_OWNER],
            balanceOf(addresses[_GOVERNANCE_CONTRACT])
        );
    }

    /**
     * @dev Mints tokens of the sender from the old contract to the sender
     */
    function migrate() external {
        require(!migrated[msg.sender], "Already migrated");
        _doMint(
            msg.sender,
            BaseToken(addresses[_OLD_TELLOR]).balanceOf(msg.sender)
        );
        migrated[msg.sender] = true;
    }

    /**
     * @dev Use this function to withdraw released tokens to the oracle
     */
    function mintToOracle() external {
        require(uints[keccak256("_INIT")] == 1, "tellor360 not initiated");
        // X - 0.02X = 144 daily time based rewards. X = 146.94
        uint256 _releasedAmount = (146.94 ether *
            (block.timestamp - uints[keccak256("_LAST_RELEASE_TIME_DAO")])) /
            86400;
        uints[keccak256("_LAST_RELEASE_TIME_DAO")] = block.timestamp;
        uint256 _stakingRewards = (_releasedAmount * 2) / 100;
        _doMint(addresses[_ORACLE_CONTRACT], _releasedAmount - _stakingRewards);
        // Send staking rewards
        _doMint(address(this), _stakingRewards);
        _allowances[address(this)][
            addresses[_ORACLE_CONTRACT]
        ] = _stakingRewards;
        ITellorFlex(addresses[_ORACLE_CONTRACT]).addStakingRewards(
            _stakingRewards
        );
    }

    /**
     * @dev Use this function to withdraw released tokens to the team
     */
    function mintToTeam() external {
        require(uints[keccak256("_INIT")] == 1, "tellor360 not initiated");
        uint256 _releasedAmount = (146.94 ether *
            (block.timestamp - uints[keccak256("_LAST_RELEASE_TIME_TEAM")])) /
            (86400);
        uints[keccak256("_LAST_RELEASE_TIME_TEAM")] = block.timestamp;
        _doMint(addresses[_OWNER], _releasedAmount);
    }

    /**
     * @dev This function allows team to gain control of any tokens sent directly to this
     * contract (and send them back))
     */
    function transferOutOfContract() external {
        _doTransfer(address(this), addresses[_OWNER], balanceOf(address(this)));
    }

    /**
     * @dev Use this function to update the oracle contract
     */
    function updateOracleAddress() external {
        bytes32 _queryID = keccak256(
            abi.encode("TellorOracleAddress", abi.encode(bytes("")))
        );
        bytes memory _proposedOracleAddressBytes;
        (, _proposedOracleAddressBytes, ) = IOracle(addresses[_ORACLE_CONTRACT])
            .getDataBefore(_queryID, block.timestamp - 12 hours);
        address _proposedOracle = abi.decode(
            _proposedOracleAddressBytes,
            (address)
        );
        // If the oracle address being reported is the same as the proposed oracle then update the oracle contract
        // only if 7 days have passed since the new oracle address was made official
        // and if 12 hours have passed since query id 1 was first reported on the new oracle contract
        if (_proposedOracle == addresses[keccak256("_PROPOSED_ORACLE")]) {
            require(
                block.timestamp >
                    uints[keccak256("_TIME_PROPOSED_UPDATED")] + 7 days,
                "must wait 7 days after proposing new oracle"
            );
            bytes32 _id = 0x83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992;
            uint256 _firstTimestamp = IOracle(_proposedOracle)
                .getTimestampbyQueryIdandIndex(_id, 0);
            require(
                block.timestamp - _firstTimestamp >= 12 hours,
                "contract should be at least 12 hours old"
            );
            addresses[_ORACLE_CONTRACT] = _proposedOracle;
            emit NewOracleAddress(_proposedOracle, block.timestamp);
        }
        // Otherwise if the current reported oracle is not the proposed oracle, then propose it and
        // start the clock on the 7 days before it can be made official
        else {
            require(_isValid(_proposedOracle), "invalid oracle address");
            addresses[keccak256("_PROPOSED_ORACLE")] = _proposedOracle;
            uints[keccak256("_TIME_PROPOSED_UPDATED")] = block.timestamp;
            emit NewProposedOracleAddress(_proposedOracle, block.timestamp);
        }
    }

    /**
     * @dev Used during the upgrade process to verify valid Tellor Contracts
     */
    function verify() external pure returns (uint256) {
        return 9999;
    }

    /**Internal Functions */
    /**
     * @dev Used during the upgrade process to verify valid Tellor Contracts and ensure
     * they have the right signature
     * @param _contract is the address of the Tellor contract to verify
     * @return bool of whether or not the address is a valid Tellor contract
     */
    function _isValid(address _contract) internal returns (bool) {
        (bool _success, bytes memory _data) = address(_contract).call(
            abi.encodeWithSelector(0xfc735e99, "") // verify() signature
        );
        require(
            _success && abi.decode(_data, (uint256)) > 9000, // An arbitrary number to ensure that the contract is valid
            "New contract is invalid"
        );
        return true;
    }
}
