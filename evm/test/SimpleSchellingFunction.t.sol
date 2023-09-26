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
        schelling.initiateProposal{value: 5 ether}();
        // propose with sufficient value
        schelling.initiateProposal{value: 10 ether}();
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
        schelling.initiateProposal{value: 10 ether}();
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
        schelling.initiateProposal{value: 10 ether}();
        vm.expectRevert(bytes("not initiator"));
        vm.prank(bob);
        schelling.submitImplementation(0, implementation2);
        vm.prank(alice);
        schelling.submitImplementation(0, implementation2);
        (,address pImpl,,,,,,,) = schelling.getProposal(0);
        assertEq(pImpl, implementation2);
    }

    function testVote() public {
        address implementation2 = makeAddr("implementation2");
        vm.deal(alice, 100 ether);
        vm.deal(bob, 100 ether);
        // propose pause bridge
        vm.prank(alice);
        schelling.initiateProposal{value: 10 ether}();
        uint256 proposalTime = block.timestamp;
        // should revert if 0 value
        vm.startPrank(bob);
        vm.expectRevert(bytes("insufficient amount"));
        schelling.vote{value: 0}(0, true);
        // should succeed if sufficient value
        schelling.vote{value: 20 ether}(0, false);
        vm.stopPrank();
        uint256 expectedExpiration = proposalTime + submissionPeriod + extensionPeriod;
        (,,uint256 amountFor,uint256 amountAgainst,,uint256 expirationTime,,,) = schelling.getProposal(0);
        assertEq(amountFor, 9 ether);
        assertEq(amountAgainst, 20 ether);
        assertEq(expirationTime, expectedExpiration);
        assertEq(80 ether, bob.balance);
        (
            uint256 bobAmountFor,
            uint256 bobAmountAgainst
        ) = schelling.getVotes(0, bob);
        assertEq(bobAmountFor, 0);
        assertEq(bobAmountAgainst, 20 ether);

        // advance time beyond submission period
        skip(submissionPeriod + 1);
        vm.prank(alice);
        schelling.vote{value: 20 ether}(0, true);
        vm.stopPrank();
        expectedExpiration = block.timestamp + extensionPeriod;
        (,, amountFor, amountAgainst,, expirationTime,,,) = schelling.getProposal(0);
        assertEq(amountFor, 29 ether);
        assertEq(amountAgainst, 20 ether);
        assertEq(expirationTime, expectedExpiration);
        assertEq(70 ether, alice.balance);
        (
            uint256 aliceAmountFor,
            uint256 aliceAmountAgainst
        ) = schelling.getVotes(0, alice);
        assertEq(aliceAmountFor, 29 ether);
        assertEq(aliceAmountAgainst, 0);

        // skip past expiration
        skip(extensionPeriod + 1);
        vm.prank(alice);
        vm.expectRevert(bytes("proposal expired"));
        schelling.vote{value: 20 ether}(0, true);
    }

    function testExecuteProposal() public {
        address implementation2 = makeAddr("implementation2");
        vm.deal(alice, 100 ether);
        // propose pause bridge
        vm.startPrank(alice);
        schelling.initiateProposal{value: 10 ether}();
        // submit implementation
        schelling.submitImplementation(0, implementation2);
        vm.stopPrank();
        vm.prank(multisig);
        // guardian pause bridge
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
        assertEq(uint256(outcome), uint256(SimpleSchelling.ProposalOutcome.FOR));
        assertEq(bridgeProxy.implementation(), implementation2);
        assertEq(bridgeProxy.paused(), false);

        // failed proposal - more votes against
        address implementation3 = makeAddr("implementation3");
        vm.deal(alice, 100 ether);
        // propose pause bridge
        vm.startPrank(alice);
        schelling.initiateProposal{value: 10 ether}();
        // submit implementation
        schelling.submitImplementation(1, implementation3);
        vm.stopPrank();
        vm.prank(multisig);
        // guardian pause bridge
        schelling.guardianPauseBridge(1);
        skip(submissionPeriod + 1);
        vm.deal(bob, 100 ether);
        vm.startPrank(bob);
        schelling.vote{value: 20 ether}(1, false);
        skip(extensionPeriod + 1);
        schelling.executeProposal(1);
        vm.stopPrank();
        (,,,,,, executed,, outcome) = schelling.getProposal(1);
        assertEq(executed, true);
        assertEq(uint256(outcome), uint256(SimpleSchelling.ProposalOutcome.AGAINST));
        assertEq(bridgeProxy.implementation(), implementation2);
        assertEq(bridgeProxy.paused(), false);

        // failed proposal - initiator never submitted implementation
        vm.deal(alice, 100 ether);
        // propose pause bridge
        vm.startPrank(alice);
        schelling.initiateProposal{value: 10 ether}();
        vm.stopPrank();
        vm.prank(multisig);
        // guardian pause bridge
        schelling.guardianPauseBridge(2);
        skip(submissionPeriod + extensionPeriod + 1);
        vm.prank(bob);
        schelling.executeProposal(2);
        (,,,,,, executed,, outcome) = schelling.getProposal(2);
        assertEq(executed, true);
        assertEq(uint256(outcome), uint256(SimpleSchelling.ProposalOutcome.AGAINST));
        assertEq(bridgeProxy.implementation(), implementation2);
        assertEq(bridgeProxy.paused(), false);

        // invalid proposal - guardian never signed
        address implementation4 = makeAddr("implementation4");
        vm.deal(alice, 100 ether);
        // propose pause bridge
        vm.startPrank(alice);
        schelling.initiateProposal{value: 10 ether}();
        schelling.submitImplementation(3, implementation4);
        skip(submissionPeriod + extensionPeriod + 1);
        schelling.executeProposal(3);
        vm.stopPrank();
        (,,,,,, executed,, outcome) = schelling.getProposal(3);
        assertEq(executed, true);
        assertEq(uint256(outcome), uint256(SimpleSchelling.ProposalOutcome.INVALID));
        assertEq(bridgeProxy.implementation(), implementation2);
        assertEq(bridgeProxy.paused(), false);
    }

    function testUpdateVote() public {
        address implementation2 = makeAddr("implementation2");
        vm.deal(alice, 100 ether);
        vm.deal(bob, 100 ether);
        // propose pause bridge
        vm.startPrank(alice);
        schelling.initiateProposal{value: 10 ether}();
        uint256 proposalTime = block.timestamp;
        vm.expectRevert(bytes("initiator cannot update vote"));
        schelling.updateVote(0, false);
        vm.stopPrank();
        // should revert if 0 value
        vm.startPrank(bob);
        vm.expectRevert(bytes("no vote"));
        schelling.updateVote(0, true);
        // check initial expected values
        uint256 expectedExpiration = proposalTime + submissionPeriod + extensionPeriod;
        (,,uint256 amountFor,uint256 amountAgainst,,uint256 expirationTime,,,) = schelling.getProposal(0);
        // should succeed if positive value

        schelling.vote{value: 20 ether}(0, false);
        schelling.updateVote(0, true);
        // vm.stopPrank();
        // uint256 expectedExpiration = proposalTime + submissionPeriod + extensionPeriod;
        // (,,uint256 amountFor,uint256 amountAgainst,,uint256 expirationTime,,,) = schelling.getProposal(0);
        // assertEq(amountFor, 9 ether);
        // assertEq(amountAgainst, 20 ether);
        // assertEq(expirationTime, expectedExpiration);
        // assertEq(80 ether, bob.balance);
        // (
        //     uint256 bobAmountFor,
        //     uint256 bobAmountAgainst
        // ) = schelling.getVotes(0, bob);
        // assertEq(bobAmountFor, 0);
        // assertEq(bobAmountAgainst, 20 ether);

        // // advance time beyond submission period
        // skip(submissionPeriod + 1);
        // vm.prank(alice);
        // schelling.vote{value: 20 ether}(0, true);
        // vm.stopPrank();
        // expectedExpiration = block.timestamp + extensionPeriod;
        // (,, amountFor, amountAgainst,, expirationTime,,,) = schelling.getProposal(0);
        // assertEq(amountFor, 29 ether);
        // assertEq(amountAgainst, 20 ether);
        // assertEq(expirationTime, expectedExpiration);
        // assertEq(70 ether, alice.balance);
        // (
        //     uint256 aliceAmountFor,
        //     uint256 aliceAmountAgainst
        // ) = schelling.getVotes(0, alice);
        // assertEq(aliceAmountFor, 29 ether);
        // assertEq(aliceAmountAgainst, 0);

        // // skip past expiration
        // skip(extensionPeriod + 1);
        // vm.prank(alice);
        // vm.expectRevert(bytes("proposal expired"));
        // schelling.vote{value: 20 ether}(0, true);
    }
}
