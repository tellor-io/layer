// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.19;

import { TellorDataBridge, OracleAttestationData, Validator, Signature } from "../bridge/TellorDataBridge.sol";

interface IERC20 {
    function transfer(address to, uint256 amount) external returns (bool);
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
}

// escrows TRB for a Layer oracle query; fronters who tip on Layer get repaid here once the attested settlement report is relayed
contract CrossChainTipEscrow {
    struct Escrow {
        address tipper;
        bytes32 queryId; // keccak256 of the query data this escrow pays for
        uint256 amount; // escrowed tip in 1e18 units, divisible by 1e12
        uint256 reward; // speed bonus pool, may be 0
        uint256 gracePeriod; // seconds of full reward
        uint256 decayWindow; // seconds the reward takes to decay to zero
        uint256 createdAt;
        uint256 deadline; // refund allowed after this
        bool settled;
    }

    // the delivered report, same shape as SimpleLayerUser.PriceData but value stays bytes
    struct OracleData {
        bytes value; // reported value
        uint256 timestamp; // aggregate report timestamp
        uint256 aggregatePower; // reporter power behind the data report
        uint256 previousTimestamp; // previous report timestamp
        uint256 nextTimestamp; // next report timestamp
        uint256 relayTimestamp; // time relayed data included in block
        uint256 attestationTimestamp; // time of attestation
    }

    TellorDataBridge public immutable dataBridge;
    IERC20 public immutable token;
    uint256 public constant LOYA_MULTIPLIER = 1e12; // loya (6 dec) -> TRB (18 dec)
    uint256 public constant MS_PER_SECOND = 1000;
    uint256 public constant MAX_ATTESTATION_AGE = 10 minutes;
    uint256 public constant CLOCK_SKEW_TOLERANCE = 5 minutes;
    uint256 public constant MIN_TIMEOUT = 1 hours;
    uint256 public escrowCount;
    mapping(uint256 => Escrow) public escrows;
    mapping(uint256 => OracleData) internal oracleData;

    event Tipped(
        uint256 indexed escrowId,
        address indexed tipper,
        bytes32 indexed queryId,
        bytes queryData,
        uint256 amount,
        uint256 reward,
        uint256 deadline
    );
    event Settled(
        uint256 indexed escrowId,
        address[] funders,
        uint256[] amountsPaid,
        uint256 bonusPaid,
        uint256 returnedToTipper
    );
    event Refunded(uint256 indexed escrowId, uint256 amount);
    event DataDelivered(uint256 indexed escrowId, bytes32 indexed dataQueryId, bytes value, uint256 aggregateTimestampMs);

    constructor(address _dataBridge, address _token) {
        dataBridge = TellorDataBridge(_dataBridge);
        token = IERC20(_token);
    }

    // lock a tip (plus optional speed bonus) for a query until someone fronts it on Layer
    function tip(
        bytes calldata _queryData,
        uint256 _amount,
        uint256 _reward,
        uint256 _timeout,
        uint256 _gracePeriod,
        uint256 _decayWindow
    ) external returns (uint256 _escrowId) {
        require(_amount > 0, "amount must be positive");
        require(_amount % LOYA_MULTIPLIER == 0, "amount must be divisible by 1e12");
        require(_timeout >= MIN_TIMEOUT, "timeout below minimum");
        require(_reward == 0 || _decayWindow >= 1, "decay window required with reward");
        require(token.transferFrom(msg.sender, address(this), _amount + _reward), "transfer failed");
        _escrowId = ++escrowCount;
        escrows[_escrowId] = Escrow({
            tipper: msg.sender,
            queryId: keccak256(_queryData),
            amount: _amount,
            reward: _reward,
            gracePeriod: _gracePeriod,
            decayWindow: _decayWindow,
            createdAt: block.timestamp,
            deadline: block.timestamp + _timeout,
            settled: false
        });
        emit Tipped(_escrowId, msg.sender, keccak256(_queryData), _queryData, _amount, _reward, block.timestamp + _timeout);
    }

    // verify the attested settlement report, repay funders in order plus bonus, deliver the data to the tipper
    function settle(
        uint256 _escrowId,
        OracleAttestationData calldata _attestData,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) external {
        Escrow storage _e = escrows[_escrowId];
        require(_e.tipper != address(0), "escrow does not exist");
        require(!_e.settled, "escrow already settled");
        // a settlement report can only ever pay its own escrow
        require(_attestData.queryId == getSettlementQueryId(_escrowId), "not this escrow's settlement report");
        // 2/3 validator signatures are the entire authenticity proof
        dataBridge.verifyOracleData(_attestData, _currentValidatorSet, _sigs);
        _checkAttestationFreshness(_attestData.attestationTimestamp);
        _e.settled = true;
        _executeSettlement(_escrowId, _e, _attestData);
    }

    function _checkAttestationFreshness(uint256 _attestationTimestampMs) internal view {
        uint256 _attSec = _attestationTimestampMs / MS_PER_SECOND;
        require(_attSec <= block.timestamp + CLOCK_SKEW_TOLERANCE, "attestation timestamp in future");
        if (_attSec < block.timestamp) {
            require(block.timestamp - _attSec < MAX_ATTESTATION_AGE, "attestation too old");
        }
    }

    function _executeSettlement(uint256 _escrowId, Escrow storage _e, OracleAttestationData calldata _attestData) internal {
        (address[] memory _funders, uint256[] memory _amountsLoya) = _deliverData(_escrowId, _e, _attestData);
        (uint256[] memory _paid, uint256 _totalPaid) = _capPayouts(_e.amount, _amountsLoya);
        uint256 _bonusPaid = _payFunders(_funders, _paid, _totalPaid, _computeBonus(_e, _totalPaid));
        // unpaid tip and undecayed reward go back to the tipper
        uint256 _returnToTipper = (_e.amount - _totalPaid) + (_e.reward - _bonusPaid);
        if (_returnToTipper > 0) {
            require(token.transfer(_e.tipper, _returnToTipper), "tipper transfer failed");
        }
        emit Settled(_escrowId, _funders, _paid, _bonusPaid, _returnToTipper);
    }

    // decode the settlement, check it matches this escrow, store the report before any payouts
    function _deliverData(uint256 _escrowId, Escrow storage _e, OracleAttestationData calldata _attestData)
        internal
        returns (address[] memory _funders, uint256[] memory _amountsLoya)
    {
        bytes32 _dataQueryId;
        uint256 _aggTsMs;
        uint256 _dataPower;
        uint256 _dataPrevTsMs;
        bytes memory _dataValue;
        (_dataQueryId, _aggTsMs, _dataPower, _dataPrevTsMs, _dataValue, _funders, _amountsLoya) =
            abi.decode(_attestData.report.value, (bytes32, uint256, uint256, uint256, bytes, address[], uint256[]));

        require(_dataQueryId == _e.queryId, "settlement is for a different query");
        require(_funders.length == _amountsLoya.length && _funders.length > 0, "malformed funders list");
        require(_aggTsMs / MS_PER_SECOND + CLOCK_SKEW_TOLERANCE >= _e.createdAt, "data predates escrow");

        // nextTimestamp is 0: the data report is the newest at settlement and this record is a one-shot snapshot
        oracleData[_escrowId] = OracleData(
            _dataValue,
            _aggTsMs,
            _dataPower,
            _dataPrevTsMs,
            0,
            block.timestamp,
            _attestData.attestationTimestamp
        );
        emit DataDelivered(_escrowId, _dataQueryId, _dataValue, _aggTsMs);
    }

    // pay funders in listed order until the escrowed amount runs out
    function _capPayouts(uint256 _escrowAmount, uint256[] memory _amountsLoya)
        internal
        pure
        returns (uint256[] memory _paid, uint256 _totalPaid)
    {
        uint256 _remaining = _escrowAmount;
        _paid = new uint256[](_amountsLoya.length);
        for (uint256 _i = 0; _i < _amountsLoya.length; _i++) {
            uint256 _due = _amountsLoya[_i] * LOYA_MULTIPLIER;
            uint256 _pay = _due > _remaining ? _remaining : _due;
            _paid[_i] = _pay;
            _totalPaid += _pay;
            _remaining -= _pay;
            if (_remaining == 0) {
                break;
            }
        }
    }

    function _computeBonus(Escrow storage _e, uint256 _totalPaid) internal view returns (uint256) {
        if (_e.reward == 0 || _totalPaid == 0) {
            return 0;
        }
        uint256 _elapsed = block.timestamp - _e.createdAt;
        if (_elapsed <= _e.gracePeriod) {
            return _e.reward;
        }
        if (_elapsed < _e.gracePeriod + _e.decayWindow) {
            return (_e.reward * (_e.gracePeriod + _e.decayWindow - _elapsed)) / _e.decayWindow;
        }
        return 0;
    }

    // split the bonus pro-rata by amount actually paid
    function _payFunders(address[] memory _funders, uint256[] memory _paid, uint256 _totalPaid, uint256 _bonus)
        internal
        returns (uint256 _bonusPaid)
    {
        for (uint256 _i = 0; _i < _funders.length; _i++) {
            if (_paid[_i] == 0) {
                continue;
            }
            uint256 _funderBonus = _bonus > 0 ? (_bonus * _paid[_i]) / _totalPaid : 0;
            _bonusPaid += _funderBonus;
            require(token.transfer(_funders[_i], _paid[_i] + _funderBonus), "funder transfer failed");
        }
    }

    // tipper reclaims tip plus reward after the deadline if nothing settled
    function refund(uint256 _escrowId) external {
        Escrow storage _e = escrows[_escrowId];
        require(_e.tipper != address(0), "escrow does not exist");
        require(msg.sender == _e.tipper, "only tipper can refund");
        require(!_e.settled, "escrow already settled");
        require(block.timestamp > _e.deadline, "deadline not reached");
        _e.settled = true;
        require(token.transfer(_e.tipper, _e.amount + _e.reward), "refund transfer failed");
        emit Refunded(_escrowId, _e.amount + _e.reward);
    }

    // Layer derives the identical id, so a report binds to exactly one escrow
    function getSettlementQueryId(uint256 _escrowId) public view returns (bytes32) {
        return keccak256(abi.encode("CrossChainSettlement", abi.encode(block.chainid, address(this), _escrowId)));
    }

    // the tipper's read point, shaped like SimpleLayerUser.getCurrentPriceData
    function getCurrentData(uint256 _escrowId) external view returns (OracleData memory) {
        require(oracleData[_escrowId].timestamp != 0, "no data delivered");
        return oracleData[_escrowId];
    }

    // 0 before settlement, 1 after
    function getValueCount(uint256 _escrowId) external view returns (uint256) {
        return oracleData[_escrowId].timestamp != 0 ? 1 : 0;
    }
}
