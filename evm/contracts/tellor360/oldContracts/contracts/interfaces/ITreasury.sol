// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

interface ITreasury{
    function issueTreasury(uint256 _amount, uint256 _rate, uint256 _duration) external;
    function payTreasury(address _investor,uint256 _id) external;
    function buyTreasury(uint256 _id,uint256 _amount) external;
    function getTreasuryDetails(uint256 _id) external view returns(uint256,uint256,uint256,uint256);
    function getTreasuryAccount(uint256 _id, address _investor) external view returns(uint256);
    function getTreasuryOwners(uint256 _id) external view returns(address[] memory);
    function getTreasuryFundsByUser(address _user) external view returns(uint256);
    function wasPaid(uint256 _id, address _investor) external view returns(bool);
    function verify() external pure returns(uint);
}
