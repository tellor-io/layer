#!/usr/bin/env python3
"""
Block Timing Analysis Script
Analyzes collected block timing data to identify patterns, correlations, and bottlenecks.
"""

import json
import argparse
import sys
from pathlib import Path
from typing import List, Dict, Optional
from collections import defaultdict
import statistics


class BlockTimingAnalyzer:
    """Analyzes block timing data from JSONL files."""
    
    def __init__(self, input_file: str):
        self.input_file = Path(input_file)
        self.blocks = []
        self.load_data()
        
    def load_data(self):
        """Load block timing data from JSONL file."""
        if not self.input_file.exists():
            print(f"Error: Input file not found: {self.input_file}")
            sys.exit(1)
        
        print(f"Loading data from {self.input_file}...")
        with open(self.input_file, 'r') as f:
            for line in f:
                try:
                    block = json.loads(line.strip())
                    self.blocks.append(block)
                except json.JSONDecodeError:
                    continue
        
        print(f"Loaded {len(self.blocks)} blocks")
        
    def calculate_statistics(self) -> Dict:
        """Calculate comprehensive statistics from collected data."""
        if not self.blocks:
            return {}
        
        stats = {
            'analysis_period': {
                'start_height': self.blocks[0]['height'],
                'end_height': self.blocks[-1]['height'],
                'start_time': self.blocks[0].get('timestamp', ''),
                'end_time': self.blocks[-1].get('timestamp', ''),
                'total_blocks': len(self.blocks)
            }
        }
        
        # Block time statistics
        block_times = [b['total_block_time_seconds'] for b in self.blocks if b.get('total_block_time_seconds', 0) > 0]
        if block_times:
            stats['block_time_stats'] = {
                'mean': round(statistics.mean(block_times), 3),
                'median': round(statistics.median(block_times), 3),
                'min': round(min(block_times), 3),
                'max': round(max(block_times), 3),
                'std_dev': round(statistics.stdev(block_times), 3) if len(block_times) > 1 else 0
            }
        
        # Consensus statistics
        consensus_data = [b.get('consensus', {}) for b in self.blocks if 'consensus' in b]
        if consensus_data:
            propose_times = [c['propose_duration_ms'] for c in consensus_data if 'propose_duration_ms' in c]
            prevote_times = [c['prevote_duration_ms'] for c in consensus_data if 'prevote_duration_ms' in c]
            precommit_times = [c['precommit_duration_ms'] for c in consensus_data if 'precommit_duration_ms' in c]
            rounds_gt_0 = sum(1 for c in consensus_data if c.get('rounds', 0) > 0)
            
            stats['consensus_stats'] = {
                'mean_propose_ms': round(statistics.mean(propose_times), 1) if propose_times else 0,
                'mean_prevote_ms': round(statistics.mean(prevote_times), 1) if prevote_times else 0,
                'mean_precommit_ms': round(statistics.mean(precommit_times), 1) if precommit_times else 0,
                'rounds_gt_0_count': rounds_gt_0,
                'total_consensus_mean_ms': round(statistics.mean([c['total_consensus_ms'] for c in consensus_data if 'total_consensus_ms' in c]), 1) if consensus_data else 0
            }
        
        # Module execution statistics
        stats['module_stats'] = self._calculate_module_stats()
        
        # Tip correlation
        stats['correlation'] = self._calculate_tip_correlation()
        
        return stats
    
    def _calculate_module_stats(self) -> Dict:
        """Calculate per-module execution time statistics."""
        module_stats = {
            'begin_blocker': {},
            'end_blocker': {}
        }
        
        # Collect all module times
        begin_times = defaultdict(list)
        end_times = defaultdict(list)
        
        for block in self.blocks:
            execution = block.get('execution', {})
            
            for module, time_ms in execution.get('begin_block_modules', {}).items():
                if module != 'total':
                    begin_times[module].append(time_ms)
            
            for module, time_ms in execution.get('end_block_modules', {}).items():
                if module != 'total':
                    end_times[module].append(time_ms)
        
        # Calculate stats for each module
        for module, times in begin_times.items():
            if times:
                module_stats['begin_blocker'][module] = {
                    'mean': round(statistics.mean(times), 1),
                    'median': round(statistics.median(times), 1),
                    'min': round(min(times), 1),
                    'max': round(max(times), 1),
                    'std_dev': round(statistics.stdev(times), 1) if len(times) > 1 else 0
                }
        
        for module, times in end_times.items():
            if times:
                module_stats['end_blocker'][module] = {
                    'mean': round(statistics.mean(times), 1),
                    'median': round(statistics.median(times), 1),
                    'min': round(min(times), 1),
                    'max': round(max(times), 1),
                    'std_dev': round(statistics.stdev(times), 1) if len(times) > 1 else 0
                }
        
        return module_stats
    
    def _calculate_tip_correlation(self) -> Dict:
        """Calculate correlation between tips and block time."""
        blocks_with_tips = [b for b in self.blocks if b.get('analysis', {}).get('has_tips', False)]
        blocks_without_tips = [b for b in self.blocks if not b.get('analysis', {}).get('has_tips', False)]
        
        tip_times = [b['total_block_time_seconds'] for b in blocks_with_tips if b.get('total_block_time_seconds', 0) > 0]
        no_tip_times = [b['total_block_time_seconds'] for b in blocks_without_tips if b.get('total_block_time_seconds', 0) > 0]
        
        correlation = {
            'blocks_with_tips': len(blocks_with_tips),
            'blocks_without_tips': len(blocks_without_tips),
            'avg_block_time_with_tips': round(statistics.mean(tip_times), 3) if tip_times else 0,
            'avg_block_time_without_tips': round(statistics.mean(no_tip_times), 3) if no_tip_times else 0,
        }
        
        # Calculate difference
        if tip_times and no_tip_times:
            diff = correlation['avg_block_time_with_tips'] - correlation['avg_block_time_without_tips']
            correlation['difference_seconds'] = round(diff, 3)
            correlation['percent_increase'] = round((diff / correlation['avg_block_time_without_tips']) * 100, 1)
        
        return correlation
    
    def identify_slow_blocks(self, threshold_std_dev: float = 2.0) -> List[Dict]:
        """Identify blocks that are significantly slower than average."""
        block_times = [b['total_block_time_seconds'] for b in self.blocks if b.get('total_block_time_seconds', 0) > 0]
        if len(block_times) < 2:
            return []
        
        mean_time = statistics.mean(block_times)
        std_dev = statistics.stdev(block_times)
        threshold = mean_time + (threshold_std_dev * std_dev)
        
        slow_blocks = []
        for block in self.blocks:
            block_time = block.get('total_block_time_seconds', 0)
            if block_time > threshold:
                deviation = block_time - mean_time
                slow_blocks.append({
                    'height': block['height'],
                    'block_time': block_time,
                    'deviation_from_mean': round(deviation, 3),
                    'std_devs_above_mean': round(deviation / std_dev, 2),
                    'has_tips': block.get('analysis', {}).get('has_tips', False),
                    'tip_count': block.get('analysis', {}).get('tip_count', 0),
                    'num_txs': block.get('transactions', {}).get('count', 0),
                    'slowest_module': self._find_slowest_module(block)
                })
        
        return sorted(slow_blocks, key=lambda x: x['block_time'], reverse=True)
    
    def _find_slowest_module(self, block: Dict) -> str:
        """Find the slowest module in a block."""
        execution = block.get('execution', {})
        all_modules = {}
        
        for module, time_ms in execution.get('begin_block_modules', {}).items():
            if module != 'total':
                all_modules[f'begin_blocker.{module}'] = time_ms
        
        for module, time_ms in execution.get('end_block_modules', {}).items():
            if module != 'total':
                all_modules[f'end_blocker.{module}'] = time_ms
        
        if all_modules:
            slowest = max(all_modules.items(), key=lambda x: x[1])
            return f"{slowest[0]} ({slowest[1]:.1f}ms)"
        
        return "unknown"
    
    def print_summary(self, stats: Dict):
        """Print a formatted summary of statistics."""
        print("\n" + "="*80)
        print("BLOCK TIMING ANALYSIS SUMMARY")
        print("="*80)
        
        # Analysis period
        period = stats.get('analysis_period', {})
        print(f"\nAnalysis Period:")
        print(f"  Heights: {period.get('start_height', '?')} - {period.get('end_height', '?')}")
        print(f"  Total Blocks: {period.get('total_blocks', 0)}")
        
        # Block time stats
        block_stats = stats.get('block_time_stats', {})
        if block_stats:
            print(f"\nBlock Time Statistics:")
            print(f"  Mean:     {block_stats.get('mean', 0):.3f}s")
            print(f"  Median:   {block_stats.get('median', 0):.3f}s")
            print(f"  Min:      {block_stats.get('min', 0):.3f}s")
            print(f"  Max:      {block_stats.get('max', 0):.3f}s")
            print(f"  Std Dev:  {block_stats.get('std_dev', 0):.3f}s")
        
        # Consensus stats
        consensus_stats = stats.get('consensus_stats', {})
        if consensus_stats:
            print(f"\nConsensus Statistics:")
            print(f"  Mean Propose:    {consensus_stats.get('mean_propose_ms', 0):.1f}ms")
            print(f"  Mean Prevote:    {consensus_stats.get('mean_prevote_ms', 0):.1f}ms")
            print(f"  Mean Precommit:  {consensus_stats.get('mean_precommit_ms', 0):.1f}ms")
            print(f"  Total Consensus: {consensus_stats.get('total_consensus_mean_ms', 0):.1f}ms")
            print(f"  Blocks with Multiple Rounds: {consensus_stats.get('rounds_gt_0_count', 0)}")
        
        # Module stats - BeginBlocker
        module_stats = stats.get('module_stats', {})
        begin_stats = module_stats.get('begin_blocker', {})
        if begin_stats:
            print(f"\nBeginBlocker Module Times (mean):")
            sorted_modules = sorted(begin_stats.items(), key=lambda x: x[1]['mean'], reverse=True)
            for module, times in sorted_modules[:10]:  # Top 10
                print(f"  {module:20s}: {times['mean']:8.1f}ms (max: {times['max']:.1f}ms)")
        
        # Module stats - EndBlocker
        end_stats = module_stats.get('end_blocker', {})
        if end_stats:
            print(f"\nEndBlocker Module Times (mean):")
            sorted_modules = sorted(end_stats.items(), key=lambda x: x[1]['mean'], reverse=True)
            for module, times in sorted_modules[:10]:  # Top 10
                print(f"  {module:20s}: {times['mean']:8.1f}ms (max: {times['max']:.1f}ms)")
        
        # Tip correlation
        correlation = stats.get('correlation', {})
        if correlation:
            print(f"\nTip Correlation Analysis:")
            print(f"  Blocks with Tips:    {correlation.get('blocks_with_tips', 0)}")
            print(f"  Blocks without Tips: {correlation.get('blocks_without_tips', 0)}")
            print(f"  Avg Time (with tips):    {correlation.get('avg_block_time_with_tips', 0):.3f}s")
            print(f"  Avg Time (without tips): {correlation.get('avg_block_time_without_tips', 0):.3f}s")
            if 'difference_seconds' in correlation:
                print(f"  Difference:              {correlation['difference_seconds']:.3f}s ({correlation.get('percent_increase', 0):.1f}% increase)")
        
        print("\n" + "="*80)
    
    def print_slow_blocks(self, slow_blocks: List[Dict]):
        """Print details of slow blocks."""
        if not slow_blocks:
            print("\nNo significantly slow blocks detected.")
            return
        
        print(f"\n{'='*80}")
        print(f"SLOW BLOCKS (>{len(slow_blocks)} blocks identified)")
        print(f"{'='*80}\n")
        
        for i, block in enumerate(slow_blocks[:20], 1):  # Show top 20
            print(f"{i}. Block {block['height']}:")
            print(f"   Time: {block['block_time']:.3f}s "
                  f"({block['std_devs_above_mean']:.1f} std devs above mean)")
            print(f"   Tips: {'YES' if block['has_tips'] else 'no'} "
                  f"(count: {block['tip_count']})")
            print(f"   Txs: {block['num_txs']}")
            print(f"   Slowest Module: {block['slowest_module']}")
            print()
    
    def export_summary(self, stats: Dict, output_file: str):
        """Export summary statistics to JSON file."""
        output_path = Path(output_file)
        with open(output_path, 'w') as f:
            json.dump(stats, f, indent=2)
        print(f"Summary exported to: {output_path}")


class ComparativeAnalyzer:
    """Compares two sets of block timing data."""
    
    def __init__(self, baseline_file: str, test_file: str):
        self.baseline = BlockTimingAnalyzer(baseline_file)
        self.test = BlockTimingAnalyzer(test_file)
        
    def compare(self):
        """Compare baseline and test datasets."""
        baseline_stats = self.baseline.calculate_statistics()
        test_stats = self.test.calculate_statistics()
        
        print("\n" + "="*80)
        print("COMPARATIVE ANALYSIS")
        print("="*80)
        
        # Compare block times
        baseline_bt = baseline_stats.get('block_time_stats', {})
        test_bt = test_stats.get('block_time_stats', {})
        
        if baseline_bt and test_bt:
            print(f"\nBlock Time Comparison:")
            print(f"  Baseline Mean: {baseline_bt['mean']:.3f}s")
            print(f"  Test Mean:     {test_bt['mean']:.3f}s")
            diff = test_bt['mean'] - baseline_bt['mean']
            percent_change = (diff / baseline_bt['mean']) * 100
            print(f"  Difference:    {diff:+.3f}s ({percent_change:+.1f}%)")
        
        # Compare consensus times
        baseline_cs = baseline_stats.get('consensus_stats', {})
        test_cs = test_stats.get('consensus_stats', {})
        
        if baseline_cs and test_cs:
            print(f"\nConsensus Time Comparison:")
            baseline_total = baseline_cs.get('total_consensus_mean_ms', 0)
            test_total = test_cs.get('total_consensus_mean_ms', 0)
            print(f"  Baseline: {baseline_total:.1f}ms")
            print(f"  Test:     {test_total:.1f}ms")
            diff = test_total - baseline_total
            print(f"  Difference: {diff:+.1f}ms")
        
        # Compare module times
        print(f"\nModule Execution Time Changes:")
        self._compare_modules(baseline_stats, test_stats)
        
        print("\n" + "="*80)
    
    def _compare_modules(self, baseline_stats: Dict, test_stats: Dict):
        """Compare module execution times between baseline and test."""
        baseline_modules = baseline_stats.get('module_stats', {})
        test_modules = test_stats.get('module_stats', {})
        
        # Compare EndBlocker modules (usually more significant)
        baseline_end = baseline_modules.get('end_blocker', {})
        test_end = test_modules.get('end_blocker', {})
        
        all_modules = set(baseline_end.keys()) | set(test_end.keys())
        
        changes = []
        for module in all_modules:
            baseline_time = baseline_end.get(module, {}).get('mean', 0)
            test_time = test_end.get(module, {}).get('mean', 0)
            diff = test_time - baseline_time
            if baseline_time > 0:
                percent = (diff / baseline_time) * 100
            else:
                percent = 0
            changes.append({
                'module': module,
                'baseline': baseline_time,
                'test': test_time,
                'diff': diff,
                'percent': percent
            })
        
        # Sort by absolute difference
        changes.sort(key=lambda x: abs(x['diff']), reverse=True)
        
        print("\n  EndBlocker Modules (Top 10 changes):")
        for change in changes[:10]:
            print(f"    {change['module']:20s}: "
                  f"{change['baseline']:7.1f}ms → {change['test']:7.1f}ms "
                  f"({change['diff']:+7.1f}ms, {change['percent']:+6.1f}%)")


def main():
    parser = argparse.ArgumentParser(
        description='Analyze collected block timing data'
    )
    parser.add_argument(
        '--input',
        required=True,
        help='Input JSONL file with block timing data'
    )
    parser.add_argument(
        '--output',
        help='Output file for summary statistics (JSON)'
    )
    parser.add_argument(
        '--compare',
        action='store_true',
        help='Compare two datasets (requires --baseline and --test)'
    )
    parser.add_argument(
        '--baseline',
        help='Baseline dataset for comparison'
    )
    parser.add_argument(
        '--test',
        help='Test dataset for comparison'
    )
    parser.add_argument(
        '--slow-threshold',
        type=float,
        default=2.0,
        help='Standard deviations above mean to consider a block slow (default: 2.0)'
    )
    
    args = parser.parse_args()
    
    if args.compare:
        if not args.baseline or not args.test:
            print("Error: --compare requires both --baseline and --test")
            sys.exit(1)
        
        analyzer = ComparativeAnalyzer(args.baseline, args.test)
        analyzer.compare()
    else:
        analyzer = BlockTimingAnalyzer(args.input)
        stats = analyzer.calculate_statistics()
        analyzer.print_summary(stats)
        
        # Identify slow blocks
        slow_blocks = analyzer.identify_slow_blocks(threshold_std_dev=args.slow_threshold)
        analyzer.print_slow_blocks(slow_blocks)
        
        # Export if requested
        if args.output:
            analyzer.export_summary(stats, args.output)


if __name__ == '__main__':
    main()

