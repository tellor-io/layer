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
response1=$(./layerd query oracle get-reportsby-reporter-qid tellor10usyr7v4xe2uhtnvg4kwtgtuzh5e4u2378zjj9 83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992 --page-limit 10 --output json)
next_key=$(echo "$response1" | jq -r '.pagination.next_key')

echo "Page 1 results:"
echo "$response1"
echo "Next key: $next_key"

# Page 2 (only if next_key is not null)
if [ "$next_key" != "null" ] && [ -n "$next_key" ]; then
    echo -e "\n--- Page 2 ---"
    response2=$(./layerd query oracle get-reportsby-reporter-qid tellor10usyr7v4xe2uhtnvg4kwtgtuzh5e4u2378zjj9 83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992 --page-key "$next_key" --output json)
    next_key=$(echo "$response2" | jq -r '.pagination.next_key')

    echo "$response2"
    echo "Next key: $next_key"
fi

# Continue as needed
```
