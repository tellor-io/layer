// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract StakingToken is ERC20 {

  constructor() ERC20("Test Tellor Tribute", "TTRB") {}

  function mint(address _to, uint256 _amount) public {
    _mint(_to, _amount);
  }

}
