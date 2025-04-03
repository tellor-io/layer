import json
import os
import csv
from datetime import datetime

os.makedirs("scripts/benchmark/results", exist_ok=True)
output_file = "scripts/benchmark/results/benchmark_results.csv"
timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

def parse_benchmark_line(output):
    # example format: "BenchmarkBridgeEndBlock-8   \t    8433\t    638903 ns/op\t 1259462 B/op\t   13410 allocs/op\n"
    parts = output.strip().split('\t')
    if len(parts) < 2:
        return None
        
    # clean and split the parts
    parts = [p.strip() for p in parts]
    
    result = {
        'name': parts[0].split()[0],  # Take first part of name (removes the -8)
        'ns_per_op': None,
        'allocs_per_op': None,
        'bytes_per_op': None
    }
    
    # parse each metric
    for part in parts:
        if 'ns/op' in part:
            result['ns_per_op'] = float(part.split()[0])
        elif 'allocs/op' in part:
            result['allocs_per_op'] = float(part.split()[0])
        elif 'B/op' in part:
            result['bytes_per_op'] = float(part.split()[0])
    
    return result

# read and parse the JSON
with open("scripts/benchmark/results/benchmark_results.json", 'r') as f:
    data = [json.loads(line) for line in f if line.strip()]

# filter for benchmark entries and parse them
benchmarks = []
for d in data:
    output = d.get("Output", "")
    if output and "Benchmark" in output and "ns/op" in output:
        try:
            result = parse_benchmark_line(output)
            if result:  # only add if we got a valid parse
                benchmarks.append(result)
        except (IndexError, ValueError) as e:
            print(f"Failed to parse line: {output}")
            print(f"Error: {e}")
            continue

print(f"Found {len(benchmarks)} benchmark results")

# check if file exists to determine if we need headers
file_exists = os.path.isfile(output_file)

# write to CSV, appending if file exists
mode = 'a' if file_exists else 'w'
with open(output_file, mode, newline='') as f:
    writer = csv.writer(f)
    
    # write headers only if creating new file
    if not file_exists:
        writer.writerow(['Timestamp', 'Benchmark', 'Ns/Op', 'Allocs/Op', 'Bytes/Op'])
    
    # write benchmark data
    for b in benchmarks:
        writer.writerow([
            timestamp,
            b['name'],
            b['ns_per_op'],
            b['allocs_per_op'] if b['allocs_per_op'] is not None else 'N/A',
            b['bytes_per_op'] if b['bytes_per_op'] is not None else 'N/A'
        ])

print(f"Results appended to {output_file}")