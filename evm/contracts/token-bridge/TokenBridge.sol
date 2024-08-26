// SPDX-License-Identifier: MIT
pragma solidity ^0.8.22;

import "../bridge/BlobstreamO.sol";
import { LayerTransition } from "./LayerTransition.sol";

/// @title TokenBridge
/// @dev This is the tellor token bridge to move tokens from
/// Ethereum to layer.  No one needs to do this.  The only reason you 
/// move your tokens over is to become a reporter/validator/tipper.  It works by
/// using layer itself as the bridge and then reads the lightclient contract for 
/// bridging back.  There is a long delay in bridging back (enforced by layer) of 12 hours
contract TokenBridge is LayerTransition{
    /*Storage*/
    BlobstreamO public bridge;
    uint256 public depositId;//counterOfHowManydeposits have been made
    uint256 public depositLimitUpdateTime;//last time the limit was updated
    uint256 public depositLimitRecord;//amount you can bridge per limit period
    uint256 public constant DEPOSIT_LIMIT_UPDATE_INTERVAL = 12 hours;
    uint256 public immutable DEPOSIT_LIMIT_DENOMINATOR = 100e18 / 20e18; // 100/depositLimitPercentage

    mapping(uint256 => bool) public withdrawalClaimed; // withdrawal id => claimed status
    mapping(address => uint256) public tokensToClaim; // recipient => extra amount to claim
    mapping(uint256 => DepositDetails) public deposits; // deposit id => deposit details

    struct DepositDetails {
        address sender;
        string recipient;
        uint256 amount;
        uint256 tip;
        uint256 blockHeight;
    }

    /*Events*/
    event Deposit(uint256 _depositId, address _sender, string _recipient, uint256 _amount, uint256 _tip);
    event Withdrawal(uint256 _depositId, string _sender, address _recipient, uint256 _amount);

    /*Functions*/
    /// @notice constructor
    /// @param _token address of tellor token for bridging
    /// @param _blobstream address of BlobstreamO data bridge
    /// @param _tellorFlex address of oracle(tellorFlex) on chain
    constructor(address _token, address _blobstream, address _tellorFlex) LayerTransition(_tellorFlex, _token){
        bridge = BlobstreamO(_blobstream);
    }

    /// @notice claim extra withdrawals that were not fully withdrawn
    /// @param _recipient address of the recipient
    function claimExtraWithdraw(address _recipient) external {
        uint256 _amountConverted = tokensToClaim[_recipient];
        require(_amountConverted > 0, "amount must be > 0");
        uint256 _depositLimit = _refreshDepositLimit();
        require(_depositLimit > 0, "TokenBridge: depositLimit must be > 0");
        if(_depositLimit < _amountConverted){
            tokensToClaim[_recipient] = tokensToClaim[_recipient] - _depositLimit;
            _amountConverted = _depositLimit;
            require(token.transfer(_recipient, _amountConverted), "TokenBridge: transfer failed");
        }
        else{
            tokensToClaim[_recipient] = 0;
            require(token.transfer(_recipient, _amountConverted), "TokenBridge: transfer failed");
        }
        depositLimitRecord -= _amountConverted;
    }

    /// @notice deposits tokens from Ethereum to layer
    /// @param _amount total amount of tokens to bridge over
    /// @param _tip amount of tokens to tip the claimDeposit caller on layer
    /// @param _layerRecipient your cosmos address on layer (don't get it wrong!!)
    function depositToLayer(uint256 _amount, uint256 _tip, string memory _layerRecipient) external {
        require(_amount > 0, "TokenBridge: amount must be greater than 0");
        require(_amount <= _refreshDepositLimit(), "TokenBridge: amount exceeds deposit limit for time period");
        require(_tip <= _amount, "TokenBridge: tip must be less than or equal to amount");
        require(token.transferFrom(msg.sender, address(this), _amount), "TokenBridge: transferFrom failed");
        depositId++;
        depositLimitRecord -= _amount;
        deposits[depositId] = DepositDetails(msg.sender, _layerRecipient, _amount, _tip, block.number);
        emit Deposit(depositId, msg.sender, _layerRecipient, _amount, _tip);
    }

    /// @notice This withdraws tokens from layer to mainnet Ethereum
    /// @param _attestData The data being verified
    /// @param _valset array of current validator set
    /// @param _sigs Signatures
    /// @param _depositId depositId from the layer side
    function withdrawFromLayer(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _valset,
        Signature[] calldata _sigs,
        uint256 _depositId
    ) external {
        require(_attestData.queryId == keccak256(abi.encode("TRBBridge", abi.encode(false, _depositId))), "TokenBridge: invalid queryId");
        require(!withdrawalClaimed[_depositId], "TokenBridge: withdrawal already claimed");
        require(block.timestamp - (_attestData.report.timestamp / 1000) > 12 hours, "TokenBridge: premature attestation");
        require(block.timestamp - (_attestData.attestationTimestamp / 1000) < 12 hours, "TokenBridge: attestation too old");
        bridge.verifyOracleData(_attestData, _valset, _sigs);
        require(_attestData.report.aggregatePower >= bridge.powerThreshold(), "Report aggregate power must be greater than or equal to _minimumPower");
        withdrawalClaimed[_depositId] = true;    
        (address _recipient, string memory _layerSender,uint256 _amountLoya,) = abi.decode(_attestData.report.value, (address, string, uint256, uint256));
        uint256 _amountConverted = _amountLoya * 1e12; 
        uint256 _depositLimit = _refreshDepositLimit();
        if(_depositLimit < _amountConverted){
            tokensToClaim[_recipient] = tokensToClaim[_recipient] + (_amountConverted - _depositLimit);
            _amountConverted = _depositLimit;
            require(token.transfer(_recipient, _amountConverted), "TokenBridge: transfer failed");
        }
        else{
            require(token.transfer(_recipient, _amountConverted), "TokenBridge: transfer failed");
        }
        depositLimitRecord -= _amountConverted;
        emit Withdrawal(_depositId, _layerSender, _recipient, _amountConverted);
    }

    /* View Functions */
    /// @notice refreshes the deposit limit every 12 hours so no one can spam layer with new tokens
    function depositLimit() external view returns (uint256) {
        if (block.timestamp - depositLimitUpdateTime > DEPOSIT_LIMIT_UPDATE_INTERVAL) {
            return token.balanceOf(address(this)) / DEPOSIT_LIMIT_DENOMINATOR;
        }
        else{
            return depositLimitRecord;
        }
    }

    /* Internal Functions */
    /// @notice refreshes the deposit limit every 12 hours so no one can spam layer with new tokens
    function _refreshDepositLimit() internal returns (uint256) {
        if (block.timestamp - depositLimitUpdateTime > DEPOSIT_LIMIT_UPDATE_INTERVAL) {
            uint256 _tokenBalance = token.balanceOf(address(this));
            if (_tokenBalance < 100 ether) {
                token.mintToOracle();
                _tokenBalance = token.balanceOf(address(this));
            }
            depositLimitRecord = _tokenBalance / DEPOSIT_LIMIT_DENOMINATOR;
            depositLimitUpdateTime = block.timestamp;
        } 
        return depositLimitRecord;
    }
}
