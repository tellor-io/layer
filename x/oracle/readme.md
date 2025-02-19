# `x/oracle`

## Abstract

This module enables reporting of data to the network. Reporters can commit and then reveal their report, or immediatiely reveal. Reports can only be made for the cycle list and tipped queries. For more information, reference the [ADRs](#adrs) below.

## ADRs

- [adr002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr002.md) - queryId time frame structure
- [adr1001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1001.md) - distribution of base rewards
- [adr1002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1002.md) - dual delegation
- [adr1003](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1003.md) - time based rewards eligibility
- [adr1004](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1004.md) - fees on tips
- [adr1005](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1005.md) - handling of tips after report
- [adr1008](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1008.md) - voting power by group

## Transactions

### UpdateParams
Update the module's parameters through governance.

### UpdateCyclelist
Update the cycle list through governance.

### UpdateQueryDataLimit
Update the query data size limit through governance.

### SubmitValue
Allow a reporter to submit a value for a query. Cycle list and supported tipped queries are reported automatically if a reporter has the reporter daemon is enabled.
- `./layerd tx oracle submit-value [creator] [query_data] [value]`

- `./layerd tx oracle submit-value tellor1p88ju0yhutmf5p2u798xv3umaa7ujw7gch9r4f 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747278000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 000000000000000000000000000000000000000000000058528649cf80ee0000`

### Tip
Allow a user to tip a query.
- `./layerd tx oracle tip [tipper] [query_data] [amount]`

- `./layerd tx oracle tip tellor1p88ju0yhutmf5p2u798xv3umaa7ujw7gch9r4f 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747278000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 1000000loya`

## Getters

### Params
- `./layerd query oracle params`

### GetReportsbyQid
- `./layerd query oracle get-reportsby-qid [query_id]`

### GetReportsbyReporter
- `./layerd query oracle get-reportsby-reporter [reporter]`

### GetReportsbyReporterQid
- `./layerd query oracle get-reportsby-reporter-qid [reporter] [query_id]`

### GetReportsByAggregate
- `./layerd query oracle get-reports-by-aggregate [query_id] [timestamp]`

### GetCurrentTip
- `./layerd query oracle get-current-tip [query_data]`

### GetUserTipTotal
- `./layerd query oracle get-user-tip-total [tipper]`

### GetDataBefore
- `./layerd query oracle get-data-before [query_id] [timestamp]`

### GetDataAfter
- `./layerd query oracle get-data-after [query_id] [timestamp]`

### RetrieveData
- `./layerd query oracle retrieve-data [query_id] [timestamp]`

### GetTimeBasedRewards
- `./layerd query oracle get-time-based-rewards`

### CurrentCyclelistQuery
- `./layerd query oracle current-cyclelist-query`

### NextCyclelistQuery
- `./layerd query oracle next-cyclelist-query`

### GetCurrentAggregateReport
- `./layerd query oracle get-current-aggregate-report [query_id]`

### GetAggregateBeforeByReporter
- `./layerd query oracle get-aggregate-before-by-reporter [query_id] [timestamp] [reporter]`

### TippedQueries
- `./layerd query oracle tipped-queries`

### GetCurrentQueryByQueryId
- `./layerd query oracle get-current-query-by-query-id [query_id]`

### GetQueryDataLimit
- `./layerd query oracle get-query-data-limit`

### ReportedIdsByReporter
- `./layerd query oracle reported-ids-by-reporter [reporter]`

### GetCycleList
- `./layerd query oracle get-cycle-list`

## EndBlock

### SetAggregatedReport
SetAggregatedReport fetches the Query iterator for queries
that have revealed reports, then iterates over the queries and checks whether the query has expired.
If the query has expired, it fetches all the microReports for a query.Id and aggregates them based
on the query spec's aggregate method.
If the query has a tip then that tip is distributed to the micro-reports' reporters,
proportional to their reporting power.
In addition, all the micro-reports that are part of a cyclelist are gathered and their reporters are
rewarded with the time-based rewards.

### RotateQueries
Rotates through the cycle list.

## Mocks

`make mock-gen-dispute`


