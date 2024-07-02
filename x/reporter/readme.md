# `x/reporter`

## Abstract

This module enables bonded stake holders to either become reporters or select a reporter in order to submit oracle data.  

`MsgCreateReporter`

- required minimum is to 1Trb bonded to be a reporter.
- Allows a staked delegator to create a reporter with their bonded address and set the condition such as the commission rate they receieve and the minimum tokens a selector needs to have **BONDED** in order to choose the reporter.  
- **Note✏️:** currently you cannot change conditions once they are set.  

`MsgSelectReporter`

- Allows a staked delegator to select an existing reporter to give their token power to in order to share in the rewards that are given for reporting oracle data.
- **Selector** must have the minimum **BONDED** tokens required by the chosen reporter in order to be included with the reporter. Plus a Selector can only join the reporter if they haven't reached the max number of selectors; currently set to 100 selectors.

`MsgRemoveSelect`

- Can be called by anyone to remove a selector from a reporter that is **capped** if they have fell below the minimum tokens required by a reporter.

`MsgSwitchReporter`

- Allows a selector to switch to a different reporter as long as they are not capped and the minimum requirements are met.
- **Note✏️:**
  - if a selector is the reporter they are not allowed to switch.
  - the selector is locked and not allowed to participate for a period of time, the max buffer window set in `x/registry`.

For more information about how reporting works, reference the [ADRs](#adrs) below.

## ADRs

- adr1001 - distribution of base rewards
- adr1002 - dual delegation
- adr1005 - handling of tips after report
- adr1008 - voting power by group
- adr2001 - trb bridge structure

## Transactions

-`CreateReporter`
-`SelectReporter`
-`SwitchReporter`
-`RemoveSelector`
-`UnjailReporter`
-`WithdrawTip`
-`UpdateParams`

## Getters

- `Params` - get module parameters
- `Reporters` - get all staked reporters
- `DelegatorReporter` - get reporter a delegator is staked with.

## Mocks

1. cd into registry/mocks
2. run `make mock-gen`

## CLI

### Example Commands

```sh
/layerd tx reporter create-reporter "100000000000000000" "1000000" --from alice --keyring-backend $KEYRING_BACKEND --chain-id layer --home ~/.layer/alice --keyring-dir ~/.layer/alice --yes
```
