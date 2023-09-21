// SPDX-License-Identifier: UNLICENSED
pragma solidity 0.8.3;

import "forge-std/Test.sol";
import "../src/forking/SimpleSchelling.sol";
import "../src/testing/BridgeProxyMock.sol";

contract CounterTest is Test {
    BridgeProxyMock public bridgeProxy;
    BridgeProxyMock public implementation1;
    SimpleSchelling public schelling;
    uint256 minInitAmount = 10 ether;
    uint256 extensionPeriod = 1 days;

    address alice = makeAddr("alice");
    address bob = makeAddr("bob");
    address carol = makeAddr("carol");

    function setUp() public {
        implementation1 = new BridgeProxyMock(address(0));
        bridgeProxy = new BridgeProxyMock(address(implementation1));
        schelling = new SimpleSchelling(address(bridgeProxy), minInitAmount, extensionPeriod);
    }

    function testProposeFork() public {
        BridgeProxyMock implementation2 = new BridgeProxyMock(address(0));
        vm.deal(alice, 100 ether);
        assertEq(100 ether, alice.balance);
        // should revert if not enough value
        vm.startPrank(alice);
        vm.expectRevert(bytes("insufficient amount"));
        schelling.proposeFork{value: 5 ether}(address(implementation2));
        // propose with sufficient value
        schelling.proposeFork{value: 10 ether}(address(implementation2));
        vm.stopPrank();
        uint256 expectedExpiration = block.timestamp + extensionPeriod;
        (address pImpl, uint256 amountFor, uint256 amountAgainst, uint256 expirationTime, bool executed) = schelling.proposals(0);
        assertEq(pImpl, address(implementation2));
        assertEq(amountFor, 10 ether);
        assertEq(amountAgainst, 0);
        assertEq(expirationTime, expectedExpiration);
        assertEq(executed, false);
        assertEq(90 ether, alice.balance);
        (uint256 aliceAmountFor, uint256 aliceAmountAgainst, bool claimedFor, bool claimedAgainst) = schelling.getVotes(0, alice);
        assertEq(aliceAmountFor, 10 ether);
        assertEq(aliceAmountAgainst, 0);
        assertEq(claimedFor, false);
        assertEq(claimedAgainst, false);
    }

    function testVote() public {
        BridgeProxyMock implementation2 = new BridgeProxyMock(address(0));
        vm.deal(alice, 100 ether);
        vm.deal(bob, 100 ether);
        vm.deal(carol, 100 ether);
        // propose fork
        vm.prank(alice);
        schelling.proposeFork{value: 10 ether}(address(implementation2));
        // should revert if 0 value
        vm.startPrank(bob);
        vm.expectRevert(bytes("insufficient amount"));
        schelling.vote{value: 0}(0, true);
        // should succeed if sufficient value
        schelling.vote{value: 11 ether}(0, false);
        uint256 expectedExpiration = block.timestamp + extensionPeriod;
        (address pImpl, uint256 amountFor, uint256 amountAgainst, uint256 expirationTime, bool executed) = schelling.getProposal(0);
    }
}
