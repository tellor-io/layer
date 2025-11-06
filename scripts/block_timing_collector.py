#!/usr/bin/env python3
"""
Per-Block Timing Analysis Collector
Collects detailed timing data for every block including consensus phases,
module execution times, and transaction details.
"""

import requests
import json
import time
import re
import argparse
import yaml
import sys
import threading
from datetime import datetime, timedelta
from pathlib import Path
from collections import deque, defaultdict
from typing import Dict, List, Optional, Any
import base64


class ConsensusStateTracker:
    """Tracks consensus state transitions and timing for each block."""
    
    def __init__(self):
        self.current_height = None
        self.current_round = 0
        self.phase_start_times = {}
        self.phase_durations = {}
        self.last_state = None
        
    def parse_consensus_state(self, consensus_data: Dict) -> Optional[Dict]:
        """Parse consensus state and track phase transitions."""
        if not consensus_data or 'result' not in consensus_data:
            return None
        
        result = consensus_data['result']
        # Handle both direct and nested round_state structures
        if 'round_state' in result:
            round_state = result['round_state']
            height_round_step = round_state.get('height/round/step', '')
        else:
            height_round_step = result.get('height/round/step', '')
        
        # Parse height/round/step format: "12345/0/1"
        parts = height_round_step.split('/')
        if len(parts) != 3:
            return None
        
        height = int(parts[0])
        round_num = int(parts[1])
        step = int(parts[2])
        
        # Step meanings: 1=Propose, 2=Prevote, 3=Precommit, 4=Commit
        step_names = {1: 'propose', 2: 'prevote', 3: 'precommit', 4: 'commit'}
        current_phase = step_names.get(step, 'unknown')
        
        now = time.time()
        
        # Track phase transitions
        if self.current_height != height:
            # New block started
            if self.current_height is not None:
                # Finalize previous block timing
                result_timing = self._finalize_block_timing()
                self.current_height = height
                self.current_round = round_num
                self.phase_start_times = {current_phase: now}
                self.phase_durations = {}
                return result_timing
            else:
                self.current_height = height
                self.current_round = round_num
                self.phase_start_times = {current_phase: now}
        elif self.last_state != current_phase:
            # Phase transition within same block
            if self.last_state and self.last_state in self.phase_start_times:
                duration = (now - self.phase_start_times[self.last_state]) * 1000
                self.phase_durations[self.last_state] = duration
            self.phase_start_times[current_phase] = now
        
        self.last_state = current_phase
        self.current_round = round_num
        
        return None
    
    def _finalize_block_timing(self) -> Dict:
        """Finalize timing for completed block."""
        # Calculate any remaining phase duration
        if self.last_state and self.last_state in self.phase_start_times:
            duration = (time.time() - self.phase_start_times[self.last_state]) * 1000
            self.phase_durations[self.last_state] = duration
        
        total_ms = sum(self.phase_durations.values())
        
        return {
            'height': self.current_height,
            'rounds': self.current_round,
            'propose_duration_ms': self.phase_durations.get('propose', 0),
            'prevote_duration_ms': self.phase_durations.get('prevote', 0),
            'precommit_duration_ms': self.phase_durations.get('precommit', 0),
            'commit_duration_ms': self.phase_durations.get('commit', 0),
            'total_consensus_ms': total_ms
        }


class MetricsDeltaCalculator:
    """Calculates per-block deltas from cumulative Prometheus metrics."""
    
    def __init__(self):
        self.previous_metrics = None
        
    def parse_prometheus_metrics(self, metrics_text: str) -> Dict:
        """Parse Prometheus metrics into structured data."""
        if not metrics_text:
            return {}
        
        metrics = {
            'begin_blocker': {},
            'end_blocker': {},
            'consensus_height': None,
            'consensus_round': None
        }
        
        # Parse begin_blocker_sum metrics
        for match in re.finditer(r'begin_blocker_sum\{[^}]*module="([^"]+)"[^}]*\}\s+([\d.]+)', metrics_text):
            module, value = match.groups()
            metrics['begin_blocker'][module] = float(value)
        
        # Parse end_blocker_sum metrics
        for match in re.finditer(r'end_blocker_sum\{[^}]*module="([^"]+)"[^}]*\}\s+([\d.]+)', metrics_text):
            module, value = match.groups()
            metrics['end_blocker'][module] = float(value)
        
        # Parse consensus height
        height_match = re.search(r'(?:tendermint_consensus_height|consensus_height)\s+(\d+)', metrics_text)
        if height_match:
            metrics['consensus_height'] = int(height_match.group(1))
        
        # Parse consensus round
        round_match = re.search(r'(?:tendermint_consensus_round|consensus_round)\s+(\d+)', metrics_text)
        if round_match:
            metrics['consensus_round'] = int(round_match.group(1))
        
        return metrics
    
    def calculate_deltas(self, current_metrics: Dict) -> Optional[Dict]:
        """Calculate per-block deltas from cumulative metrics."""
        if not self.previous_metrics:
            self.previous_metrics = current_metrics
            return None
        
        deltas = {
            'begin_blocker': {},
            'end_blocker': {}
        }
        
        # Calculate begin_blocker deltas
        for module, value in current_metrics.get('begin_blocker', {}).items():
            prev_value = self.previous_metrics.get('begin_blocker', {}).get(module, 0)
            delta = value - prev_value
            if delta >= 0:  # Ignore negative deltas (counter resets)
                deltas['begin_blocker'][module] = delta
        
        # Calculate end_blocker deltas
        for module, value in current_metrics.get('end_blocker', {}).items():
            prev_value = self.previous_metrics.get('end_blocker', {}).get(module, 0)
            delta = value - prev_value
            if delta >= 0:
                deltas['end_blocker'][module] = delta
        
        self.previous_metrics = current_metrics
        return deltas


class TransactionAnalyzer:
    """Analyzes transactions to identify message types and extract details."""
    
    @staticmethod
    def debug_gas_location(block_results: Dict, height: int):
        """Debug helper to find where gas data is located."""
        if not block_results or 'result' not in block_results:
            print(f"  [DEBUG {height}] No block_results")
            return
        
        result = block_results['result']
        print(f"  [DEBUG {height}] Top-level keys: {list(result.keys())}")
        print(f"  [DEBUG {height}] gas_used: {result.get('gas_used')}")
        print(f"  [DEBUG {height}] gas_wanted: {result.get('gas_wanted')}")
        
        if 'txs_results' in result and result['txs_results']:
            first_tx = result['txs_results'][0]
            print(f"  [DEBUG {height}] First tx keys: {list(first_tx.keys())}")
            print(f"  [DEBUG {height}] First tx gas_used: {first_tx.get('gas_used')}")
    
    @staticmethod
    def analyze_block_txs(block_data: Dict, block_results: Dict, debug: bool = False) -> Dict:
        """Analyze transactions in a block."""
        tx_analysis = {
            'count': 0,
            'gas_used': 0,
            'gas_wanted': 0,
            'message_types': {},
            'tips': []
        }
        
        if not block_data or 'result' not in block_data:
            return tx_analysis
        
        block = block_data['result'].get('block', {})
        txs = block.get('data', {}).get('txs', [])
        tx_analysis['count'] = len(txs)
        
        # Get gas info from results
        if block_results and 'result' in block_results:
            result = block_results['result']
            
            # Try different possible locations for gas data
            # SDK v0.50.9 and earlier: result.gas_used, result.gas_wanted
            # SDK v0.53.4: might be in different location
            
            # Try top-level first
            gas_used = result.get('gas_used')
            gas_wanted = result.get('gas_wanted')
            
            # If not found, try summing from individual tx results
            if not gas_used or gas_used == "0":
                tx_results = result.get('txs_results', [])
                if tx_results:
                    total_gas_used = 0
                    total_gas_wanted = 0
                    for tx_result in tx_results:
                        try:
                            tx_gas_used = tx_result.get('gas_used', 0)
                            tx_gas_wanted = tx_result.get('gas_wanted', 0)
                            total_gas_used += int(tx_gas_used) if tx_gas_used else 0
                            total_gas_wanted += int(tx_gas_wanted) if tx_gas_wanted else 0
                        except:
                            pass
                    if total_gas_used > 0:
                        gas_used = total_gas_used
                        gas_wanted = total_gas_wanted
            
            try:
                tx_analysis['gas_used'] = int(gas_used) if gas_used else 0
            except:
                tx_analysis['gas_used'] = 0
            
            try:
                tx_analysis['gas_wanted'] = int(gas_wanted) if gas_wanted else 0
            except:
                tx_analysis['gas_wanted'] = 0
            
            # Parse transaction results for message types
            tx_results = result.get('txs_results', [])
            for tx_result in tx_results:
                events = tx_result.get('events', [])
                for event in events:
                    if event.get('type') == 'message':
                        for attr in event.get('attributes', []):
                            if attr.get('key') == 'action':
                                msg_type = attr.get('value', '')
                                # Count message types
                                tx_analysis['message_types'][msg_type] = \
                                    tx_analysis['message_types'].get(msg_type, 0) + 1
                    
                    # Extract tip details
                    if event.get('type') == 'tip':
                        tip_data = {}
                        for attr in event.get('attributes', []):
                            key = attr.get('key', '')
                            value = attr.get('value', '')
                            if key == 'query_id':
                                tip_data['query_id'] = value
                            elif key == 'amount':
                                tip_data['amount'] = value
                            elif key == 'tipper':
                                tip_data['tipper'] = value
                        if tip_data:
                            tx_analysis['tips'].append(tip_data)
        
        return tx_analysis


class LogParser:
    """Parses application logs for ABCI timing information."""
    
    def __init__(self, log_file_path: str):
        self.log_file_path = Path(log_file_path).expanduser()
        self.last_position = 0
        
    def parse_abci_timing(self) -> List[Dict]:
        """Parse recent ABCI timing entries from logs."""
        if not self.log_file_path.exists():
            return []
        
        abci_timings = []
        
        try:
            with open(self.log_file_path, 'r') as f:
                # Seek to last position
                f.seek(self.last_position)
                new_lines = f.readlines()
                self.last_position = f.tell()
                
                for line in new_lines:
                    if '[ABCI_TIMING]' in line:
                        # Parse the log line
                        timing = self._parse_abci_log_line(line)
                        if timing:
                            abci_timings.append(timing)
        except Exception as e:
            print(f"Warning: Could not parse logs: {e}")
        
        return abci_timings
    
    @staticmethod
    def _parse_abci_log_line(line: str) -> Optional[Dict]:
        """Parse a single ABCI timing log line."""
        # Example format: ... [ABCI_TIMING] height=12345 finalize_block_ms=250 num_txs=5
        match = re.search(r'height=(\d+)', line)
        if not match:
            return None
        
        height = int(match.group(1))
        
        timing = {'height': height}
        
        # Extract finalize_block_ms
        fb_match = re.search(r'finalize_block_ms=(\d+)', line)
        if fb_match:
            timing['finalize_block_ms'] = int(fb_match.group(1))
        
        # Extract num_txs
        tx_match = re.search(r'num_txs=(\d+)', line)
        if tx_match:
            timing['num_txs'] = int(tx_match.group(1))
        
        return timing


class BlockTimingCollector:
    """Main collector that combines all data sources."""
    
    def __init__(self, config: Dict):
        self.config = config
        self.rpc_url = config['endpoints']['rpc']
        self.metrics_url = config['endpoints']['metrics']
        
        self.consensus_tracker = ConsensusStateTracker()
        self.metrics_calculator = MetricsDeltaCalculator()
        self.tx_analyzer = TransactionAnalyzer()
        
        log_path = config['monitoring'].get('log_file_path', '~/.layer/layer.log')
        self.log_parser = LogParser(log_path)
        
        self.output_dir = Path(config['monitoring']['output_dir'])
        self.output_dir.mkdir(parents=True, exist_ok=True)
        
        self.last_height = None
        self.last_block_time = None
        self.block_data_buffer = {}
        
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        self.output_file = self.output_dir / f'block_timing_{timestamp}.jsonl'
        
    def fetch_json(self, url: str, timeout: int = 5) -> Optional[Dict]:
        """Fetch JSON data from URL."""
        try:
            response = requests.get(url, timeout=timeout)
            if response.status_code == 200:
                return response.json()
        except Exception as e:
            print(f"Error fetching {url}: {e}")
        return None
    
    def fetch_text(self, url: str, timeout: int = 5) -> Optional[str]:
        """Fetch text data from URL."""
        try:
            response = requests.get(url, timeout=timeout)
            if response.status_code == 200:
                return response.text
        except Exception as e:
            print(f"Error fetching {url}: {e}")
        return None
    
    def collect_block_data(self, height: int) -> Dict:
        """Collect all data for a specific block."""
        print(f"Collecting data for block {height}...")
        
        block_data = {}
        
        # 1. Fetch block details
        block = self.fetch_json(f"{self.rpc_url}/block?height={height}")
        block_results = self.fetch_json(f"{self.rpc_url}/block_results?height={height}")
        
        if not block:
            return {}
        
        # Extract basic info
        block_header = block.get('result', {}).get('block', {}).get('header', {})
        block_data['height'] = int(block_header.get('height', height))
        block_data['timestamp'] = block_header.get('time', '')
        block_data['proposer'] = block_header.get('proposer_address', '')
        
        # Calculate block time
        if self.last_block_time:
            try:
                # Handle both Z and +00:00 timezone formats
                # CometBFT uses nanoseconds (9 digits), Python only supports microseconds (6 digits)
                timestamp_str = block_data['timestamp']
                if timestamp_str.endswith('Z'):
                    timestamp_str = timestamp_str[:-1] + '+00:00'
                
                # Truncate nanoseconds to microseconds (keep only 6 decimal places)
                if '.' in timestamp_str and '+' in timestamp_str:
                    before_decimal, after_decimal = timestamp_str.split('.')
                    fractional_and_tz = after_decimal.split('+')
                    fractional = fractional_and_tz[0][:6]  # Keep only 6 digits
                    tz = fractional_and_tz[1]
                    timestamp_str = f"{before_decimal}.{fractional}+{tz}"
                
                current_time = datetime.fromisoformat(timestamp_str)
                block_time = (current_time - self.last_block_time).total_seconds()
                block_data['total_block_time_seconds'] = round(block_time, 3)
            except Exception as e:
                print(f"  Warning: Could not calculate block time: {e}")
                block_data['total_block_time_seconds'] = 0
        else:
            block_data['total_block_time_seconds'] = 0
        
        # Update last block time for next iteration
        try:
            timestamp_str = block_data['timestamp']
            if timestamp_str.endswith('Z'):
                timestamp_str = timestamp_str[:-1] + '+00:00'
            
            # Truncate nanoseconds to microseconds
            if '.' in timestamp_str and '+' in timestamp_str:
                before_decimal, after_decimal = timestamp_str.split('.')
                fractional_and_tz = after_decimal.split('+')
                fractional = fractional_and_tz[0][:6]  # Keep only 6 digits
                tz = fractional_and_tz[1]
                timestamp_str = f"{before_decimal}.{fractional}+{tz}"
            
            self.last_block_time = datetime.fromisoformat(timestamp_str)
        except Exception as e:
            print(f"  Warning: Could not parse timestamp: {e}")
        
        # 2. Scrape metrics and calculate deltas
        metrics_text = self.fetch_text(self.metrics_url)
        if metrics_text:
            current_metrics = self.metrics_calculator.parse_prometheus_metrics(metrics_text)
            deltas = self.metrics_calculator.calculate_deltas(current_metrics)
            
            if deltas:
                # Calculate totals
                begin_total = sum(deltas['begin_blocker'].values())
                end_total = sum(deltas['end_blocker'].values())
                
                block_data['execution'] = {
                    'begin_block_modules': {**deltas['begin_blocker'], 'total': round(begin_total, 1)},
                    'end_block_modules': {**deltas['end_blocker'], 'total': round(end_total, 1)},
                    'total_execution_ms': round(begin_total + end_total, 1)
                }
                
                if block_data['total_block_time_seconds'] > 0:
                    percent = (begin_total + end_total) / (block_data['total_block_time_seconds'] * 1000) * 100
                    block_data['execution']['percent_of_block_time'] = round(percent, 1)
        
        # 3. Analyze transactions
        tx_analysis = self.tx_analyzer.analyze_block_txs(block, block_results)
        block_data['transactions'] = tx_analysis
        
        # 4. Check for consensus timing data
        if height in self.block_data_buffer:
            consensus_data = self.block_data_buffer.pop(height)
            block_data['consensus'] = consensus_data
            
            # Calculate percentage
            if block_data['total_block_time_seconds'] > 0:
                percent = consensus_data['total_consensus_ms'] / (block_data['total_block_time_seconds'] * 1000) * 100
                block_data['consensus']['percent_of_block_time'] = round(percent, 1)
        
        # 5. Parse ABCI timing from logs
        abci_timings = self.log_parser.parse_abci_timing()
        for timing in abci_timings:
            if timing['height'] == height:
                block_data['abci'] = {
                    'finalize_block_ms': timing.get('finalize_block_ms', 0)
                }
                break
        
        # 6. Analysis
        block_data['analysis'] = {
            'has_tips': len(tx_analysis.get('tips', [])) > 0,
            'tip_count': len(tx_analysis.get('tips', [])),
        }
        
        return block_data
    
    def monitor_consensus_state(self):
        """Continuously monitor consensus state."""
        poll_interval = 0.05  # 50ms - faster polling for better consensus tracking
        
        while True:
            try:
                consensus_state = self.fetch_json(f"{self.rpc_url}/consensus_state", timeout=2)
                if consensus_state:
                    completed_block = self.consensus_tracker.parse_consensus_state(consensus_state)
                    if completed_block:
                        # Buffer this consensus data for the block
                        height = completed_block['height']
                        self.block_data_buffer[height] = completed_block
            except Exception as e:
                # Don't spam errors - consensus tracking is best effort
                pass
            
            time.sleep(poll_interval)
    
    def monitor_blocks(self, duration: Optional[int] = None):
        """Monitor new blocks and collect timing data."""
        print(f"Starting block timing collection...")
        print(f"Output file: {self.output_file}")
        
        # Start consensus monitoring in background
        consensus_thread = threading.Thread(target=self.monitor_consensus_state, daemon=True)
        consensus_thread.start()
        
        start_time = time.time()
        blocks_collected = 0
        
        try:
            while True:
                # Check duration limit
                if duration and (time.time() - start_time) > duration:
                    print(f"\nDuration limit reached ({duration}s)")
                    break
                
                # Get current block height
                status = self.fetch_json(f"{self.rpc_url}/status")
                if not status:
                    time.sleep(1)
                    continue
                
                current_height = int(status['result']['sync_info']['latest_block_height'])
                
                # Check for new block
                if self.last_height is None:
                    self.last_height = current_height
                    print(f"Starting from block {current_height}")
                elif current_height > self.last_height:
                    # Collect data for each new block
                    for height in range(self.last_height + 1, current_height + 1):
                        block_data = self.collect_block_data(height)
                        
                        if block_data:
                            # Write to file
                            with open(self.output_file, 'a') as f:
                                f.write(json.dumps(block_data) + '\n')
                            
                            blocks_collected += 1
                            self._print_block_summary(block_data)
                    
                    self.last_height = current_height
                
                time.sleep(0.5)
        
        except KeyboardInterrupt:
            print("\n\nCollection interrupted by user")
        
        print(f"\nTotal blocks collected: {blocks_collected}")
        print(f"Output saved to: {self.output_file}")
    
    def _print_block_summary(self, block_data: Dict):
        """Print a summary of collected block data."""
        height = block_data.get('height', '?')
        block_time = block_data.get('total_block_time_seconds', 0)
        num_txs = block_data.get('transactions', {}).get('count', 0)
        has_tips = block_data.get('analysis', {}).get('has_tips', False)
        
        execution = block_data.get('execution', {})
        exec_time = execution.get('total_execution_ms', 0)
        
        consensus = block_data.get('consensus', {})
        consensus_time = consensus.get('total_consensus_ms', 0)
        
        # Show which module is slowest
        slowest_module = ""
        if execution.get('end_block_modules'):
            modules = {k: v for k, v in execution['end_block_modules'].items() if k != 'total'}
            if modules:
                slowest = max(modules.items(), key=lambda x: x[1])
                slowest_module = f" | Slowest: {slowest[0]}({slowest[1]:.1f}ms)"
        
        print(f"Block {height}: {block_time:.3f}s | "
              f"Txs: {num_txs} | "
              f"Exec: {exec_time:.1f}ms | "
              f"Consensus: {consensus_time:.1f}ms{slowest_module} | "
              f"Tips: {'YES' if has_tips else 'no'}")


def load_config(config_file: str) -> Dict:
    """Load configuration from YAML file."""
    config_path = Path(config_file)
    if not config_path.exists():
        print(f"Error: Config file not found: {config_file}")
        sys.exit(1)
    
    with open(config_path, 'r') as f:
        return yaml.safe_load(f)


def main():
    parser = argparse.ArgumentParser(
        description='Collect per-block timing data for Cosmos SDK chain'
    )
    parser.add_argument(
        '--config',
        default='scripts/config/block_timing_config.yaml',
        help='Path to configuration file'
    )
    parser.add_argument(
        '--duration',
        type=int,
        help='Duration to monitor in seconds (default: indefinite)'
    )
    
    args = parser.parse_args()
    
    config = load_config(args.config)
    collector = BlockTimingCollector(config)
    collector.monitor_blocks(duration=args.duration)


if __name__ == '__main__':
    main()

