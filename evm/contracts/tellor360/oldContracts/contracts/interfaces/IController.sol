// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

interface IController{
    function addresses(bytes32) external returns(address);
    function uints(bytes32) external returns(uint256);
    function burn(uint256 _amount) external;
    function changeDeity(address _newDeity) external;
    function changeOwner(address _newOwner) external;
    function changeTellorContract(address _tContract) external;
    function changeControllerContract(address _newController) external;
    function changeGovernanceContract(address _newGovernance) external;
    function changeOracleContract(address _newOracle) external;
    function changeTreasuryContract(address _newTreasury) external;
    function changeUint(bytes32 _target, uint256 _amount) external;
    function migrate() external;
    function mint(address _reciever, uint256 _amount) external;
    function init() external;
    function getDisputeIdByDisputeHash(bytes32 _hash) external view returns (uint256);
    function getLastNewValueById(uint256 _requestId) external view returns (uint256, bool);
    function retrieveData(uint256 _requestId, uint256 _timestamp) external view returns (uint256);
    function getNewValueCountbyRequestId(uint256 _requestId) external view returns (uint256);
    function getAddressVars(bytes32 _data) external view returns (address);
    function getUintVar(bytes32 _data) external view returns (uint256);
    function totalSupply() external view returns (uint256);
    function name() external pure returns (string memory);
    function symbol() external pure returns (string memory);
    function decimals() external pure returns (uint8);
    function allowance(address _user, address _spender) external view  returns (uint256);
    function allowedToTrade(address _user, uint256 _amount) external view returns (bool);
    function approve(address _spender, uint256 _amount) external returns (bool);
    function approveAndTransferFrom(address _from, address _to, uint256 _amount) external returns(bool);
    function balanceOf(address _user) external view returns (uint256);
    function balanceOfAt(address _user, uint256 _blockNumber)external view returns (uint256);
    function transfer(address _to, uint256 _amount)external returns (bool success);
    function transferFrom(address _from,address _to,uint256 _amount) external returns (bool success) ;
    function depositStake() external;
    function requestStakingWithdraw() external;
    function withdrawStake() external;
    function changeStakingStatus(address _reporter, uint _status) external;
    function slashReporter(address _reporter, address _disputer) external;
    function getStakerInfo(address _staker) external view returns (uint256, uint256);
    function getTimestampbyRequestIDandIndex(uint256 _requestID, uint256 _index) external view returns (uint256);
    function getNewCurrentVariables()external view returns (bytes32 _c,uint256[5] memory _r,uint256 _d,uint256 _t);
    //in order to call fallback function
    function beginDispute(uint256 _requestId, uint256 _timestamp,uint256 _minerIndex) external;
    function unlockDisputeFee(uint256 _disputeId) external;
    function vote(uint256 _disputeId, bool _supportsDispute) external;
    function tallyVotes(uint256 _disputeId) external;
    //test functions
    function tipQuery(uint,uint,bytes memory) external;
    function getNewVariablesOnDeck() external view returns (uint256[5] memory idsOnDeck, uint256[5] memory tipsOnDeck);
}
