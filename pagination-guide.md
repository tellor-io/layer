# Layer Blockchain Pagination Guide

Use `--help` on any query and check the available flags to see if the query supports pagination.

## Pagination Flags

- `--page-limit`: Maximum number of results to return per page. Defaults to 10 for GetReportsByReporter and GetReportersNoStakeReports
- `--page-reverse`: Return results in reverse order
- `--page-offset`: Number of results to skip from the beginning
- `--page-key`: NextKey from previous query response (for continuation)


## Usage

### Page Limit, Page Reverse

```bash
# Get 5 oldest reports by reporter
layerd query oracle get-reportsby-reporter tellor1abc... --page-limit 5

# Get 5 newest reports by reporter
layerd query oracle get-reportsby-reporter tellor1abc... --page-limit 5 --page-reverse

# Get most recent governance proposal
layerd query gov proposals --page-limit 1 --page-reverse

# Get the second oldest report
layerd query oracle get-reportsby-reporter tellor1abc... --page-offset 1 --page-limit 1

# Get the second newest report
layerd query oracle get-reportsby-reporter tellor1abc... --page-offset 1 --page-reverse --page-limit 1

# Get 10 reports starting with key eyJrZXkiOiJhYmMxMjM...
layerd query oracle get-reportsby-reporter tellor1abc... --page-key "eyJrZXkiOiJhYmMxMjM..."
```

### Offset

```bash
# Skip first 5 results, get next 3
layerd query oracle get-reportsby-reporter tellor1abc... --page-offset 5 --page-limit 3

# Skip first 10 proposals, get next 5
layerd query gov proposals --page-offset 10 --page-limit 5
```


### Multiple Page Examples

#### Offset-Based
```bash
# Page 1
layerd query <module> <command> <args> --page-limit 10 --page-offset 0

# Page 2  
layerd query <module> <command> <args> --page-limit 10 --page-offset 10

# Page 3
layerd query <module> <command> <args> --page-limit 10 --page-offset 20
```

#### NextKey-Based (Recommended)
The `NextKey` is returned in the pagination response when there are more results available. Use it to fetch subsequent pages:

```bash
# First page, returns nextKey at end of query results
layerd query <module> <command> <args> --page-limit 5

# Use NextKey for subsequent pages
layerd query <module> <command> <args> --page-key <NextKey_from_previous_response>
```

Example:
```bash
# Page 1
response1=$(layerd query <module> <command> <args> --page-limit 10)
next_key=$(echo "$response1" | jq -r '.pagination.next_key')

# Page 2
response2=$(layerd query <module> <command> <args> --page-key "$next_key")
next_key=$(echo "$response2" | jq -r '.pagination.next_key')

# Continue until next_key is null
```
