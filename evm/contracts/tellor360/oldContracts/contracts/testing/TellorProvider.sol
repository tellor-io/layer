// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

interface IMedianOracle{
    function reportDelaySec() external returns(uint256);
    function reportExpirationTimeSec() external returns(uint256);
    function pushReport(uint256 payload) external;
    function purgeReports() external;
}

interface ITellorGetters {
    function getNewValueCountbyRequestId(uint _requestId) external view returns(uint);
    function getTimestampbyRequestIDandIndex(uint _requestID, uint _index) external view returns(uint);
    function retrieveData(uint _requestId, uint _timestamp) external view returns (uint);
}

contract TellorProvider{

    ITellorGetters public tellor;
    IMedianOracle public medianOracle;


    struct TellorTimes{
        uint128 time0;
        uint128 time1;
    }
    TellorTimes public tellorReport;
    uint256 constant TellorID = 10;


    constructor(address _tellor, address _medianOracle) {
        tellor = ITellorGetters(_tellor);
        medianOracle = IMedianOracle(_medianOracle);
    }

    function pushTellor() external {
        (, uint256 value, uint256 _time) = getTellorData();
        //Saving _time in a storage value to quickly verify disputes later
        if(tellorReport.time0 >= tellorReport.time1) {
            tellorReport.time1 = uint128(_time);
        } else {
            tellorReport.time0 = uint128(_time);
        }
        medianOracle.pushReport(value);
    }

    function verifyTellorReports() external {
        //most recent tellor report is in dispute, so let's purge it
        if(tellor.retrieveData(TellorID, tellorReport.time0) == 0 || tellor.retrieveData(TellorID,tellorReport.time1) == 0){
            medianOracle.purgeReports();
        }
    }

    function getTellorData() internal view returns(bool, uint256, uint256){
        uint256 _count = tellor.getNewValueCountbyRequestId(TellorID);
        if(_count > 0) {
            uint256 _time = tellor.getTimestampbyRequestIDandIndex(TellorID, _count - 1);
            uint256 _value = tellor.retrieveData(TellorID, _time);
            return(true, _value, _time);
        }
        return (false, 0, 0);
    }

}
