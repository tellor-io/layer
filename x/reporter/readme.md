# `x/reporter`

## Abstract

This module enables bonded stake holders to either become reporters or select a reporter in order to submit oracle data.

For more information about how reporting works, reference the [ADRs](#adrs) below.

## ADRs

- [adr1001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1001.md) - distribution of base rewards
- [adr1002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1002.md) - dual delegation
- [adr1005](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1005.md) - handling of tips after report
- [adr1008](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1008.md) - voting power by group
- [adr2001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr2001.md) - trb bridge structure

## Transactions

### CreateReporter
Allows a staked delegator to create a reporter with their bonded address and set the condition such as the commission rate they receieve and the minimum tokens a selector needs to have **BONDED** in order to choose the reporter.  
- **Note✏️:** currently you cannot change conditions once they are set.  

- `./layerd tx reporter create-reporter [commission-rate] [min-tokens-required]`

### SelectReporter
Allows a staked delegator to select an existing reporter to give their token power to in order to share in the rewards that are given for reporting oracle data. **Selector** must have the minimum **BONDED** tokens required by the chosen reporter in order to be included with the reporter. Plus a Selector can only join the reporter if they haven't reached the max number of selectors; currently set to 100 selectors.

- `./layerd tx reporter select-reporter [reporter-address]`

### SwitchReporter
Allows a selector to switch to a different reporter as long as they are not capped and the minimum requirements are met.  

- **Note✏️:** if a selector is the reporter they are not allowed to switch.
- **Note✏️:** the selector is locked and not allowed to participate for a period of time, the max buffer window set in `x/registry`.

- `./layerd tx reporter switch-reporter [reporter-address]`

### RemoveSelector
Can be called by anyone to remove a selector from a reporter that is **capped** if they have fell below the minimum tokens required by a reporter.

- `./layerd tx reporter remove-selector [selector-address]`

### UnjailReporter
Allows a reporter that is jailed to be unjailed if the jail period has passed (jail period is set during a dispute).

- `./layerd tx reporter unjail-reporter [reporter-address]`

### WithdrawTip
Allows selectors to directly withdraw reporting rewards and stake them with a BONDED validator.

- `./layerd tx reporter withdraw-tip [selector-address] [validator-address]`

### UpdateParams

## Getters

### Params

### Reporters

### SelectorReporter

### AllowedAmount

### NumOfSelectorsByReporter

### SpaceAvailableByReporter

### AvailableTips

## Mocks

`make mock-gen-reporter`

## CLI

### Example Commands

```sh
./layerd tx reporter create-reporter 0.25 1000000 --from YOUR_ACCOUNT_NAME --chain-id layertest-3 --fees 10loya --yes
```
