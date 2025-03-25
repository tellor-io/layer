#!/usr/bin/env python3
import sys
import re
import csv
from datetime import datetime
import os

# Define the output directory relative to script location
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
OUTPUT_FILE = os.path.join(SCRIPT_DIR, 'results', 'benchmark_results.csv')

# Create results directory if it doesn't exist
os.makedirs(os.path.join(SCRIPT_DIR, 'results'), exist_ok=True)

# Updated regex to be more flexible with benchmark names
bench_regex = re.compile(
    r'Benchmark(?P<benchmark_name>\w+)/(?P<test_case>[\w_]+)-\d+\s+' +
    r'(?P<iterations>\d+)\s+' +
    r'(?P<ns_per_op>\d+)\s+ns/op'
)

def parse_benchmarks(input_lines):
    results = []
    date = datetime.now().strftime('%Y-%m-%d')
    
    for line in input_lines:
        match = bench_regex.search(line)
        if match:
            full_benchmark_name = f"Benchmark{match.group('benchmark_name')}"
            results.append({
                'date': date,
                'benchmark_name': full_benchmark_name,
                'test_case': match.group('test_case'),
                'iterations': int(match.group('iterations')),
                'ns_per_op': int(match.group('ns_per_op')),
                'ms_per_op': float(match.group('ns_per_op')) / 1_000_000,
            })
    return results

def write_csv(results, output_file=OUTPUT_FILE):
    fieldnames = ['date', 'benchmark_name', 'test_case', 'iterations', 'ns_per_op', 'ms_per_op']
    
    file_exists = False
    try:
        with open(output_file, 'r'):
            file_exists = True
    except FileNotFoundError:
        pass

    mode = 'a' if file_exists else 'w'
    with open(output_file, mode, newline='') as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        if not file_exists:
            writer.writeheader()
        writer.writerows(results)
    
    print(f"Wrote {len(results)} benchmark results to {output_file}")

def main():
    input_text = sys.stdin.readlines()
    results = parse_benchmarks(input_text)
    write_csv(results)

if __name__ == '__main__':
    main()