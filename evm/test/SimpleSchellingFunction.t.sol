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
    uint256 submissionPeriod = 1 days;
    bool requireOwnerSignature = true;
    address multisig = makeAddr("multisig");

    address alice = makeAddr("alice");
    address bob = makeAddr("bob");
    address carol = makeAddr("carol");

    function setUp() public {
        implementation1 = new BridgeProxyMock(address(0));
        bridgeProxy = new BridgeProxyMock(address(implementation1));
        schelling = new SimpleSchelling(
            address(bridgeProxy),
            minInitAmount,
            extensionPeriod,
            submissionPeriod,
            requireOwnerSignature,
            multisig
        );
    }

    function testPauseBridge() public {
        vm.deal(alice, 100 ether);
        assertEq(100 ether, alice.balance);
        // should revert if not enough value
        vm.startPrank(alice);
        vm.expectRevert(bytes("insufficient amount"));
        schelling.pauseBridge{value: 5 ether}();
        // propose with sufficient value
        schelling.pauseBridge{value: 10 ether}();
        vm.stopPrank();
        uint256 expectedSubDeadline = block.timestamp + submissionPeriod;
        uint256 expectedExpiration = expectedSubDeadline + extensionPeriod;
        (
            address initiator,
            address pImpl,
            uint256 amountFor,
            uint256 amountAgainst,
            uint256 subDeadline,
            uint256 expirationTime,
            bool executed,
            bool ownerSigned,
            SimpleSchelling.ProposalOutcome outcome
        ) = schelling.getProposal(0);
        assertEq(initiator, alice);
        assertEq(pImpl, address(0));
        assertEq(amountFor, 9 ether);
        assertEq(amountAgainst, 0);
        assertEq(subDeadline, expectedSubDeadline);
        assertEq(expirationTime, expectedExpiration);
        assertEq(executed, false);
        assertEq(ownerSigned, false);
        assertEq(uint256(outcome), 0);
        assertEq(90 ether, alice.balance);
        (
            uint256 aliceAmountFor,
            uint256 aliceAmountAgainst
        ) = schelling.getVotes(0, alice);
        assertEq(aliceAmountFor, 9 ether);
        assertEq(aliceAmountAgainst, 0);
        assertEq(bridgeProxy.paused(), false);
    }

    function testGuardianPauseBridge() public {
        vm.deal(alice, 100 ether);
        vm.startPrank(alice);
        schelling.pauseBridge{value: 10 ether}();
        // should revert if not owner
        vm.expectRevert(bytes("not guardian"));
        schelling.guardianPauseBridge(0);
        vm.stopPrank();
        vm.startPrank(multisig);
        schelling.guardianPauseBridge(0);
        vm.expectRevert(bytes("bridge already paused"));
        schelling.guardianPauseBridge(0);
        vm.stopPrank();
        assertEq(bridgeProxy.paused(), true);
        (,,,,,,,bool guardianSigned,) = schelling.getProposal(0);
        assertEq(guardianSigned, true);
    }

    function testSubmitImplementation() public {
        address implementation2 = makeAddr("implementation2");
        vm.deal(alice, 100 ether);
        vm.prank(alice);
        schelling.pauseBridge{value: 10 ether}();
        vm.expectRevert(bytes("not initiator"));
        vm.prank(bob);
        schelling.submitImplementation(0, implementation2);
        vm.prank(alice);
        schelling.submitImplementation(0, implementation2);
        (,address pImpl,,,,,,,) = schelling.getProposal(0);
        assertEq(pImpl, implementation2);
    }

    // function testVote() public {
    //     BridgeProxyMock implementation2 = new BridgeProxyMock(address(0));
    //     vm.deal(alice, 100 ether);
    //     vm.deal(bob, 100 ether);
    //     vm.deal(carol, 100 ether);
    //     // propose fork
    //     vm.prank(alice);
    //     schelling.proposeFork{value: 10 ether}(address(implementation2));
    //     // should revert if 0 value
    //     vm.startPrank(bob);
    //     vm.expectRevert(bytes("insufficient amount"));
    //     schelling.vote{value: 0}(0, true);
    //     // should succeed if sufficient value
    //     schelling.vote{value: 11 ether}(0, false);
    //     uint256 expectedExpiration = block.timestamp + extensionPeriod;
    //     (address pImpl, uint256 amountFor, uint256 amountAgainst, uint256 expirationTime, bool executed) = schelling.getProposal(0);
    // }

    function testExecuteProposal() public {
        address implementation2 = makeAddr("implementation2");
        vm.deal(alice, 100 ether);
        // propose fork
        vm.startPrank(alice);
        schelling.pauseBridge{value: 10 ether}();
        schelling.submitImplementation(0, implementation2);
        vm.stopPrank();
        vm.prank(multisig);
        schelling.guardianPauseBridge(0);
        vm.startPrank(alice);
        vm.expectRevert(bytes("submission period not expired"));
        schelling.executeProposal(0);
        skip(submissionPeriod + 1);
        vm.expectRevert(bytes("proposal not expired"));
        schelling.executeProposal(0);
        skip(extensionPeriod);
        schelling.executeProposal(0);
        vm.expectRevert(bytes("already executed"));
        schelling.executeProposal(0);
        vm.stopPrank();
        (,,,,,, bool executed,, SimpleSchelling.ProposalOutcome outcome) = schelling.getProposal(0);
        assertEq(executed, true);
        assertEq(uint256(outcome), 1);
        assertEq(bridgeProxy.implementation(), implementation2);
        assertEq(bridgeProxy.paused(), false);
    }
}
