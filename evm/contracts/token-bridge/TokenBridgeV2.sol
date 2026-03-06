// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "../interfaces/ITellorDataBridge.sol";
import { LayerTransition } from "./LayerTransition.sol";
import { RoleManager } from "./RoleManager.sol";

/// @author Tellor Inc.
/// @title TokenBridgeV2
/// @dev Tellor token bridge for moving tokens between Ethereum and Tellor Layer.
/// Deposits are recorded on-chain and relayed to Layer via the oracle; withdraws from Layer back
/// to Ethereum are attested via TellorDataBridge (validator signatures). Security measures include
/// a 12-hour delay in both directions, per-period token rate limits, and a temporary guardian-approved
/// pause that requires burning a large tribute (PAUSE_TRIBUTE_AMOUNT).
contract TokenBridgeV2 is LayerTransition, RoleManager {
    /*Storage*/
    ITellorDataBridge public dataBridge;
    address public deployer; // address that deployed the bridge
    uint256 public depositId; // counter of how many deposits have been made
    uint256 public depositLimitUpdateTime; // last time the deposit limit was updated
    uint256 public depositLimitRecord; // amount you can deposit per limit period
    BridgeState public bridgeState; // state of the bridge
    uint256 public bridgeStateUpdateTime; // last time the bridge state was updated
    bool public initialized; // whether the bridge has been initialized
    uint256 public lastPauseTimestamp; // last time the bridge was paused
    uint256 public pauseProposalId; // counter of how many pause proposals have been made
    uint256 public totalPauseTributeBalance; // total amount of tokens held as pause tribute
    uint256 public withdrawLimitUpdateTime; // last time the withdraw limit was updated
    uint256 public withdrawLimitRecord; // amount you can withdraw per limit period
    uint256 public constant DEPOSIT_LIMIT_DENOMINATOR = 100e18 / 20e18; // 100/depositLimitPercentage
    uint256 public constant MS_PER_SECOND = 1000; // factor to convert milliseconds to seconds
    uint256 public constant PAUSE_PERIOD = 21 days; // bridge pause period duration
    uint256 public constant PAUSE_TRIBUTE_AMOUNT = 10000 ether; // amount of tokens burned to pause bridge
    uint256 public constant TOKEN_DECIMAL_PRECISION_MULTIPLIER = 1e12; // multiplier to convert from loya to 1e18
    uint256 public constant TWELVE_HOUR_CONSTANT = 12 hours; // deposit and withdraw limits update interval
    uint256 public constant WITHDRAW_LIMIT_DENOMINATOR = 100e18 / 5e18; // 100/withdrawLimitPercentage
    uint256 public constant PAUSE_TRIBUTE_LOCK_TIME = 7 days; // time period before a non-approved pause tribute can be refunded

    mapping(uint256 depositId => DepositDetails) public deposits; // deposit id => deposit details
    mapping(uint256 pauseProposalId => PauseProposal pauseProposal) public pauseProposals; // pause proposal id => pause proposal
    mapping(address recipient => uint256 extraAmountToClaim) public tokensToClaim; // recipient => extra amount to claim
    mapping(uint256 withdrawId => bool claimed) public withdrawClaimed; // withdraw id => claimed status
    mapping(uint256 withdrawId => WithdrawDetails withdrawDetails) public withdrawDetails; // withdraw id => withdraw details
    mapping(address recipient => uint256[] withdrawIds) public recipientWithdrawIds; // recipient => withdraw ids

    struct DepositDetails {
        address sender;
        string recipient;
        uint256 amount;
        uint256 tip;
        uint256 blockHeight;
    }

    struct PauseProposal {
        address proposer;
        uint256 proposalTime;
        PauseProposalState state;
        string layerAddress;
    }

    struct WithdrawDetails {
        uint256 withdrawId;
        address recipient;
        uint256 amount;
        uint256 pendingAmount;
        uint256 lastVerifiedTime;
    }

    enum BridgeState {
        UNPAUSED,
        PAUSED
    }

    enum PauseProposalState {
        NONE,
        PENDING,
        APPROVED,
        REFUNDED
    }

    /*Events*/
    event BridgeStateUpdated(BridgeState _newState);
    event DataBridgeUpdated(address _dataBridge);
    event Deposit(uint256 _depositId, address _sender, string _recipient, uint256 _amount, uint256 _tip);
    event ExtraWithdrawClaimed(uint256 _withdrawId, address _recipient, uint256 _amount);
    event PauseApproved(uint256 _proposalId, address _proposer, uint256 _proposalTime);
    event PauseProposed(uint256 _proposalId, address _proposer, uint256 _proposalTime);
    event TokensToClaimUpdated(address _recipient, uint256 _amount);
    event PauseRefunded(uint256 _proposalId, address _proposer, uint256 _proposalTime);
    event Withdraw(uint256 _depositId, string _sender, address _recipient, uint256 _amount);
    event ExtraWithdrawReverified(uint256 _withdrawId, address _recipient, uint256 _amount);

    /*Functions*/
    /// @notice constructor
    /// @param _token address of tellor token for bridging
    /// @param _dataBridge address of TellorDataBridge data bridge
    /// @param _tellorFlex address of oracle(tellorFlex) on chain
    /// @param _mainGuardian address of the main guardian
    /// @param _subGuardian address of the sub guardian
    /// @param _defaultRoleUpdateDelay default delay before a role update can be accepted
    constructor(
        address _token,
        address _dataBridge,
        address _tellorFlex,
        address _mainGuardian,
        address _subGuardian,
        uint256 _defaultRoleUpdateDelay
    ) LayerTransition(_tellorFlex, _token) RoleManager(_mainGuardian, _defaultRoleUpdateDelay) {
        dataBridge = ITellorDataBridge(_dataBridge);
        deployer = msg.sender;

        roles[keccak256("APPROVE_PAUSE")] = RoleInfo({
            roleAddress: _subGuardian,
            roleUpdateDelay: _defaultRoleUpdateDelay
        });
        roles[keccak256("UPDATE_DATA_BRIDGE")] = RoleInfo({
            roleAddress: _mainGuardian,
            roleUpdateDelay: _defaultRoleUpdateDelay
        });
    }

    /// @notice initializes the bridge, only on testnet
    /// @param _depositId the last deposit id used
    /// @param _withdrawId the last withdraw id used
    function init(uint256 _depositId, uint256 _withdrawId) external {
        require(msg.sender == deployer, "TokenBridgeV2: only deployer can initialize");
        require(!initialized, "TokenBridgeV2: already initialized");
        depositId = _depositId;
        // set withdraws up to _withdrawId to all claimed
        for (uint256 i = 0; i <= _withdrawId; i++) {
            withdrawClaimed[i] = true;
        }
        initialized = true;
    }

    /// @notice approves a pause proposal
    /// @param _proposalId the id of the pause proposal
    function approvePause(uint256 _proposalId) public {
        _roleRestricted(keccak256("APPROVE_PAUSE"));
        require(bridgeState == BridgeState.UNPAUSED, "TokenBridgeV2: can only propose pause when bridge is unpaused");
        PauseProposal storage _proposal = pauseProposals[_proposalId];
        require(_proposal.state == PauseProposalState.PENDING, "TokenBridgeV2: proposal is not pending");
        bridgeState = BridgeState.PAUSED;
        bridgeStateUpdateTime = block.timestamp;
        lastPauseTimestamp = block.timestamp;
        _proposal.state = PauseProposalState.APPROVED;
        totalPauseTributeBalance -= PAUSE_TRIBUTE_AMOUNT;
        token.transfer(address(0xdEaD), PAUSE_TRIBUTE_AMOUNT);
        emit BridgeStateUpdated(BridgeState.PAUSED);
        emit PauseApproved(_proposalId, _proposal.proposer, block.timestamp);
    }

    /// @notice claims extra withdraw amount
    /// @param _withdrawId the withdraw id
    function claimExtraWithdrawByWithdrawId(uint256 _withdrawId) public {
        require(bridgeState != BridgeState.PAUSED, "TokenBridgeV2: bridge is paused");
        require(initialized, "TokenBridgeV2: not initialized");
        WithdrawDetails storage _withdrawDetails = withdrawDetails[_withdrawId];
        require(_withdrawDetails.lastVerifiedTime > lastPauseTimestamp, "TokenBridgeV2: must re-verify withdraws after pause");
        uint256 _amountConverted = _withdrawDetails.pendingAmount;
        require(_amountConverted > 0, "TokenBridgeV2: no pending amount");
        uint256 _withdrawLimit = _refreshWithdrawLimit(_amountConverted);
        require(_withdrawLimit > 0, "TokenBridgeV2: withdraw limit must be > 0");
        if (_withdrawLimit < _amountConverted) {
            _amountConverted = _withdrawLimit;
        }
        _withdrawDetails.pendingAmount -= _amountConverted;
        tokensToClaim[_withdrawDetails.recipient] -= _amountConverted;
        withdrawLimitRecord -= _amountConverted;
        require(token.transfer(_withdrawDetails.recipient, _amountConverted), "TokenBridgeV2: transfer failed");
        emit ExtraWithdrawClaimed(_withdrawId, _withdrawDetails.recipient, _amountConverted);
        emit TokensToClaimUpdated(_withdrawDetails.recipient, tokensToClaim[_withdrawDetails.recipient]);
    }

    /// @notice re-verifies an extra withdraw after the bridge has been paused
    /// @param _attestData the oracle data being verified
    /// @param _valset the validator set
    /// @param _sigs the attestations
    /// @param _withdrawId the withdraw id
    function reverifyExtraWithdraw(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _valset,
        Signature[] calldata _sigs,
        uint256 _withdrawId
    ) external {
        require(bridgeState != BridgeState.PAUSED, "TokenBridgeV2: bridge is paused");
        require(initialized, "TokenBridgeV2: not initialized");
        (address _recipient /*sender*/, , uint256 _amountLoya /*tip*/, ) = abi.decode(_attestData.report.value, (address, string, uint256, uint256));
        uint256 _amountConverted = _amountLoya * TOKEN_DECIMAL_PRECISION_MULTIPLIER;
        WithdrawDetails storage _withdrawDetails = withdrawDetails[_withdrawId];
        require(_withdrawDetails.pendingAmount > 0, "TokenBridgeV2: pending amount is zero");
        require(_withdrawDetails.lastVerifiedTime < lastPauseTimestamp, "TokenBridgeV2: last verified timestamp recent enough");
        require(_withdrawDetails.amount == _amountConverted, "TokenBridgeV2: amount does not match record");
        require(_withdrawDetails.recipient == _recipient, "TokenBridgeV2: recipient address does not match record");
        _verifyWithdraw(_attestData, _valset, _sigs, _withdrawId);
        // Update last verified time so tokens can be withdrawn
        _withdrawDetails.lastVerifiedTime = block.timestamp;
        emit ExtraWithdrawReverified(_withdrawId, _recipient, _amountConverted);
    }

    /// @notice deposits tokens from Ethereum to layer
    /// @param _amount total amount of tokens to bridge over
    /// @param _tip amount of tokens to tip the claimDeposit caller on layer
    /// @param _layerRecipient your cosmos address on layer (don't get it wrong!!)
    function depositToLayer(uint256 _amount, uint256 _tip, string memory _layerRecipient) external {
        require(initialized, "TokenBridgeV2: not initialized");
        require(_amount > 0.1 ether, "TokenBridgeV2: amount must be greater than 0.1 tokens");
        require(_amount % TOKEN_DECIMAL_PRECISION_MULTIPLIER == 0, "TokenBridgeV2: amount must be divisible by 1e12");
        require(_amount <= _refreshDepositLimit(_amount), "TokenBridgeV2: amount exceeds deposit limit for time period");
        require(_tip <= _amount, "TokenBridgeV2: tip must be less than or equal to amount");
        if (_tip > 0) {
            require(_tip >= 1e12, "TokenBridgeV2: tip must be greater than or equal to 1 loya");
        }
        depositId++;
        depositLimitRecord -= _amount;
        deposits[depositId] = DepositDetails(msg.sender, _layerRecipient, _amount, _tip, block.number);
        require(token.transferFrom(msg.sender, address(this), _amount), "TokenBridgeV2: transferFrom failed");
        emit Deposit(depositId, msg.sender, _layerRecipient, _amount, _tip);
    }

    /// @notice proposes a pause of the bridge
    /// @param _layerAddress the address of the layer contract
    function proposePauseBridge(string calldata _layerAddress) external {
        require(bridgeState == BridgeState.UNPAUSED, "TokenBridgeV2: can only propose pause when bridge is unpaused");
        totalPauseTributeBalance += PAUSE_TRIBUTE_AMOUNT;
        uint256 _pauseProposalId = pauseProposalId;
        pauseProposalId++;
        pauseProposals[_pauseProposalId] = PauseProposal(msg.sender, block.timestamp, PauseProposalState.PENDING, _layerAddress);
        require(token.transferFrom(msg.sender, address(this), PAUSE_TRIBUTE_AMOUNT), "TokenBridgeV2: transfer failed");
        emit PauseProposed(_pauseProposalId, msg.sender, block.timestamp);
    }

    /// @notice refunds a pause proposal
    /// @param _proposalId the id of the pause proposal
    function refundPauseProposal(uint256 _proposalId) external {
        PauseProposal storage _proposal = pauseProposals[_proposalId];
        require(msg.sender == _proposal.proposer || msg.sender == roles[keccak256("APPROVE_PAUSE")].roleAddress, "TokenBridgeV2: only proposer or sub guardian can refund pause proposal");
        require(_proposal.state == PauseProposalState.PENDING, "TokenBridgeV2: proposal is not pending");
        require(block.timestamp - _proposal.proposalTime > PAUSE_TRIBUTE_LOCK_TIME, "TokenBridgeV2: must wait before refunding pause proposal");
        _proposal.state = PauseProposalState.REFUNDED;
        totalPauseTributeBalance -= PAUSE_TRIBUTE_AMOUNT;
        require(token.transfer(_proposal.proposer, PAUSE_TRIBUTE_AMOUNT), "TokenBridgeV2: transfer failed");
        emit PauseRefunded(_proposalId, _proposal.proposer, block.timestamp);
    }

    /// @notice updates the data bridge
    /// @param _dataBridge the address of the new data bridge
    function updateDataBridge(address _dataBridge) external {
        _roleRestricted(keccak256("UPDATE_DATA_BRIDGE"));
        require(bridgeState == BridgeState.PAUSED, "TokenBridgeV2: can only update data bridge when bridge is paused");
        dataBridge = ITellorDataBridge(_dataBridge);
        emit DataBridgeUpdated(_dataBridge);
    }

    /// @notice unpauses the bridge after the pause period has passed, can be called by anyone
    function unpauseBridge() external {
        require(bridgeState == BridgeState.PAUSED, "TokenBridgeV2: bridge is not paused");
        require(block.timestamp - bridgeStateUpdateTime > PAUSE_PERIOD, "TokenBridgeV2: must wait before unpausing");
        bridgeState = BridgeState.UNPAUSED;
        emit BridgeStateUpdated(BridgeState.UNPAUSED);
    }

    /// @notice This withdraws tokens from layer to mainnet Ethereum
    /// @param _attestData The oracle data being verified, including the withdraw info
    /// @param _valset array of current validator set
    /// @param _sigs attestations
    /// @param _withdrawId withdrawId from the layer side
    function withdrawFromLayer(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _valset,
        Signature[] calldata _sigs,
        uint256 _withdrawId
    ) external {
        require(bridgeState != BridgeState.PAUSED, "TokenBridgeV2: bridge is paused");
        require(initialized, "TokenBridgeV2: not initialized");
        require(!withdrawClaimed[_withdrawId], "TokenBridgeV2: withdraw already claimed");
        _verifyWithdraw(_attestData, _valset, _sigs, _withdrawId);
        (address _recipient, string memory _layerSender, uint256 _amountLoya /*tip*/, ) = abi.decode(
            _attestData.report.value,
            (address, string, uint256, uint256)
        );
        uint256 _amountConverted = _amountLoya * TOKEN_DECIMAL_PRECISION_MULTIPLIER;
        uint256 _withdrawLimit = _refreshWithdrawLimit(_amountConverted);
        if (_withdrawLimit < _amountConverted) {
            tokensToClaim[_recipient] = tokensToClaim[_recipient] + (_amountConverted - _withdrawLimit);
            withdrawDetails[_withdrawId] = WithdrawDetails(
                _withdrawId,
                _recipient,
                _amountConverted,
                _amountConverted - _withdrawLimit,
                block.timestamp
            );
            recipientWithdrawIds[_recipient].push(_withdrawId);

            _amountConverted = _withdrawLimit;
            emit TokensToClaimUpdated(_recipient, tokensToClaim[_recipient]);
        }
        withdrawLimitRecord -= _amountConverted;
        withdrawClaimed[_withdrawId] = true;
        require(token.transfer(_recipient, _amountConverted), "TokenBridgeV2: transfer failed");
        emit Withdraw(_withdrawId, _layerSender, _recipient, _amountConverted);
    }

    /* View Functions */
    /// @notice returns the amount of tokens that can be deposited in the current 12 hour period
    /// @return amount of tokens that can be deposited
    function depositLimit() external view returns (uint256) {
        if (block.timestamp - depositLimitUpdateTime > TWELVE_HOUR_CONSTANT) {
            return (_getTokenBalanceLessPauseTribute() + _getMintAmount()) / DEPOSIT_LIMIT_DENOMINATOR;
        } else {
            return depositLimitRecord;
        }
    }

    /// @notice returns the withdraw limit
    /// @return amount of tokens that can be withdrawn
    function withdrawLimit() external view returns (uint256) {
        if (block.timestamp - withdrawLimitUpdateTime > TWELVE_HOUR_CONSTANT) {
            return (_getTokenBalanceLessPauseTribute() + _getMintAmount()) / WITHDRAW_LIMIT_DENOMINATOR;
        } else {
            return withdrawLimitRecord;
        }
    }

    /* Internal Functions */
    /// @notice returns the amount of tokens pending to be minted to this contract
    /// @return amount of tokens pending to be minted
    function _getMintAmount() internal view returns (uint256) {
        uint256 _releasedAmount = (146.94 ether * (block.timestamp - token.getUintVar(keccak256("_LAST_RELEASE_TIME_DAO")))) / 86400;
        return _releasedAmount;
    }

    /// @notice returns the amount of tokens in the contract less the pause tribute
    /// @return amount of tokens in the contract less the pause tribute
    function _getTokenBalanceLessPauseTribute() internal view returns (uint256) {
        return token.balanceOf(address(this)) - totalPauseTributeBalance;
    }

    /// @notice refreshes the deposit limit every 12 hours so no one can spam layer with new tokens
    /// @param _amount of tokens to deposit
    /// @return max amount of tokens that can be deposited
    function _refreshDepositLimit(uint256 _amount) internal returns (uint256) {
        if (block.timestamp - depositLimitUpdateTime > TWELVE_HOUR_CONSTANT) {
            uint256 _tokenBalance = _getTokenBalanceLessPauseTribute();
            if (_tokenBalance < _amount) {
                token.mintToOracle();
                _tokenBalance = _getTokenBalanceLessPauseTribute();
            }
            depositLimitRecord = _tokenBalance / DEPOSIT_LIMIT_DENOMINATOR;
            depositLimitUpdateTime = block.timestamp;
        }
        return depositLimitRecord;
    }

    /// @notice refreshes the withdraw limit every 12 hours so no one can spam layer with new tokens
    /// @param _amount of tokens to withdraw
    /// @return max amount of tokens that can be withdrawn
    function _refreshWithdrawLimit(uint256 _amount) internal returns (uint256) {
        if (block.timestamp - withdrawLimitUpdateTime > TWELVE_HOUR_CONSTANT) {
            uint256 _tokenBalance = _getTokenBalanceLessPauseTribute();
            if (_tokenBalance < _amount) {
                token.mintToOracle();
                _tokenBalance = _getTokenBalanceLessPauseTribute();
            }
            withdrawLimitRecord = _tokenBalance / WITHDRAW_LIMIT_DENOMINATOR;
            withdrawLimitUpdateTime = block.timestamp;
        }
        return withdrawLimitRecord;
    }

    /// @notice verifies the withdraw info
    /// @param _attestData the oracle data being verified
    /// @param _valset the validator set
    /// @param _sigs the attestations
    /// @param _withdrawId the withdraw id
    function _verifyWithdraw(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _valset,
        Signature[] calldata _sigs,
        uint256 _withdrawId
    ) internal view {
        require(_attestData.queryId == keccak256(abi.encode("TRBBridgeV2", abi.encode(false, _withdrawId))), "TokenBridgeV2: invalid queryId");
        require(
            block.timestamp - (_attestData.report.timestamp / MS_PER_SECOND) > TWELVE_HOUR_CONSTANT,
            "TokenBridgeV2: must wait 12 hours before relaying withdraw"
        );
        require(block.timestamp - (_attestData.attestationTimestamp / MS_PER_SECOND) < TWELVE_HOUR_CONSTANT, "TokenBridgeV2: attestation too old");
        require(
            _attestData.report.aggregatePower >= dataBridge.powerThreshold(),
            "TokenBridgeV2: report aggregate power must be greater than or equal to power threshold"
        );
        dataBridge.verifyOracleData(_attestData, _valset, _sigs);
    }
}
