#!/usr/bin/env python3
"""
Cosmos SDK Advanced Block Performance Monitor
Comprehensive monitoring including consensus timing, app profiling, and log analysis.
"""

import requests
import json
import time
import re
import subprocess
from datetime import datetime, timedelta
from collections import deque
import statistics
import threading
from pathlib import Path

# Configuration
RPC_URL = "http://localhost:26657"
LCD_URL = "http://localhost:1317"
METRICS_URL = "http://localhost:26660/metrics"  # CometBFT Prometheus metrics
PPROF_URL = "http://localhost:6060"  # Application pprof
MONITOR_DURATION = 300  # Monitor for 5 minutes
SAMPLE_INTERVAL = 2
COMETBFT_LOG_PATH = "~/.cometbft/config/config.toml"  # Adjust as needed

class CosmosAdvancedMonitor:
    def __init__(self, rpc_url=RPC_URL, metrics_url=METRICS_URL, pprof_url=PPROF_URL):
        self.rpc_url = rpc_url
        self.metrics_url = metrics_url
        self.pprof_url = pprof_url
        
        # Block metrics
        self.block_times = deque(maxlen=100)
        self.tx_counts = deque(maxlen=100)
        self.block_sizes = deque(maxlen=100)
        self.gas_used = deque(maxlen=100)
        
        # Consensus metrics
        self.proposal_times = deque(maxlen=100)
        self.prevote_times = deque(maxlen=100)
        self.precommit_times = deque(maxlen=100)
        self.commit_times = deque(maxlen=100)
        self.consensus_rounds = deque(maxlen=100)
        
        # Application metrics
        self.state_machine_times = deque(maxlen=100)
        self.mempool_sizes = deque(maxlen=100)
        
        self.last_block_height = None
        self.last_block_time = None
        self.profile_data = {}
        
    # ==================== Block Metrics ====================
    
    def get_current_block(self):
        """Fetch the latest block from RPC"""
        try:
            response = requests.get(f"{self.rpc_url}/status", timeout=5)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"Error fetching status: {e}")
            return None
    
    def get_block_details(self, height):
        """Fetch detailed block information"""
        try:
            response = requests.get(f"{self.rpc_url}/block?height={height}", timeout=5)
            if response.status_code != 200:
                return None
            return response.json()
        except Exception as e:
            return None
    
    def get_block_results(self, height):
        """Fetch block results including gas and validation info"""
        try:
            response = requests.get(f"{self.rpc_url}/block_results?height={height}", timeout=5)
            if response.status_code != 200:
                return None
            return response.json()
        except Exception as e:
            return None
    
    def calculate_block_metrics(self, block_data, block_results):
        """Extract and calculate block performance metrics"""
        metrics = {}
        
        if block_data and 'result' in block_data:
            result = block_data['result']
            if 'block' in result:
                block = result['block']
                header = block.get('header', {})
                data = block.get('data', {})
                
                metrics['height'] = header.get('height')
                metrics['timestamp'] = header.get('time')
                metrics['num_transactions'] = len(data.get('txs', []))
                metrics['proposer'] = header.get('proposer_address')
                
                # Calculate block size (approximate)
                metrics['block_size'] = len(json.dumps(block).encode('utf-8'))
        
        if block_results and 'result' in block_results:
            result = block_results['result']
            # Handle gas_used which might be string or int
            gas_used = result.get('gas_used')
            if gas_used:
                try:
                    metrics['gas_used'] = int(gas_used) if isinstance(gas_used, str) else gas_used
                except:
                    metrics['gas_used'] = None
            
            gas_wanted = result.get('gas_wanted')
            if gas_wanted:
                try:
                    metrics['gas_wanted'] = int(gas_wanted) if isinstance(gas_wanted, str) else gas_wanted
                except:
                    metrics['gas_wanted'] = None
            
            metrics['num_events'] = len(result.get('finalize_block_events', []))
            metrics['tx_results'] = len(result.get('txs_results', []))
        
        return metrics
    
    # ==================== CometBFT Metrics ====================
    
    def fetch_prometheus_metrics(self):
        """Fetch CometBFT Prometheus metrics"""
        try:
            response = requests.get(self.metrics_url, timeout=5)
            if response.status_code == 200:
                return response.text
            else:
                print(f"  Metrics endpoint returned status {response.status_code}")
            return None
        except requests.exceptions.ConnectionError as e:
            print(f"  Connection error to metrics: {e}")
            return None
        except Exception as e:
            print(f"  Error fetching metrics: {e}")
            return None
    
    def parse_prometheus_metric(self, metrics_text, metric_name):
        """Extract a specific metric value from Prometheus output"""
        if not metrics_text:
            return None
        
        pattern = rf'{metric_name}\s+(\d+(?:\.\d+)?)'
        match = re.search(pattern, metrics_text)
        return float(match.group(1)) if match else None
    
    def extract_consensus_metrics(self, metrics_text):
        """Extract consensus-related metrics from Prometheus"""
        if not metrics_text:
            return {}
        
        metrics = {}
        
        # Mempool size - try different possible metric names
        mempool_match = re.search(r'(?:tendermint_mempool_size|mempool_size)\s+(\d+)', metrics_text)
        if mempool_match:
            metrics['mempool_size'] = int(mempool_match.group(1))
        
        # Num peers
        peers_match = re.search(r'(?:tendermint_p2p_peers|p2p_peers)\s+(\d+)', metrics_text)
        if peers_match:
            metrics['num_peers'] = int(peers_match.group(1))
        
        # Block height
        height_match = re.search(r'(?:tendermint_consensus_height|consensus_height)\s+(\d+)', metrics_text)
        if height_match:
            metrics['consensus_height'] = int(height_match.group(1))
        
        # Rounds
        rounds_match = re.search(r'(?:tendermint_consensus_round|consensus_round)\s+(\d+)', metrics_text)
        if rounds_match:
            metrics['consensus_round'] = int(rounds_match.group(1))
        
        # Extract begin_blocker and end_blocker timing by module
        begin_blocker_modules = {}
        end_blocker_modules = {}
        
        # Look for begin_blocker_sum metrics - more flexible regex
        for match in re.finditer(r'begin_blocker_sum\{[^}]*module="([^"]+)"[^}]*\}\s+([\d.]+)', metrics_text):
            module, time_ms = match.groups()
            begin_blocker_modules[module] = float(time_ms)
        
        # Look for end_blocker_sum metrics
        for match in re.finditer(r'end_blocker_sum\{[^}]*module="([^"]+)"[^}]*\}\s+([\d.]+)', metrics_text):
            module, time_ms = match.groups()
            end_blocker_modules[module] = float(time_ms)
        
        if begin_blocker_modules:
            metrics['begin_blocker_modules'] = begin_blocker_modules
        if end_blocker_modules:
            metrics['end_blocker_modules'] = end_blocker_modules
        
        return metrics
    
    # ==================== Application Profiling ====================
    
    def get_pprof_profile(self, profile_type='profile', seconds=10):
        """Fetch a CPU or heap profile from pprof"""
        try:
            url = f"{self.pprof_url}/debug/pprof/{profile_type}?seconds={seconds}"
            response = requests.get(url, timeout=seconds + 5)
            response.raise_for_status()
            return response.content
        except Exception as e:
            print(f"Warning: Could not fetch pprof {profile_type}: {e}")
            return None
    
    def analyze_pprof_profile(self, profile_data, profile_type='profile'):
        """Analyze pprof profile data using go tool pprof"""
        if not profile_data:
            return None
        
        try:
            # Save profile to temp file
            temp_file = f"/tmp/cosmos_pprof_{profile_type}_{int(time.time())}"
            with open(temp_file, 'wb') as f:
                f.write(profile_data)
            
            # Use go tool pprof to get top functions
            result = subprocess.run(
                ['go', 'tool', 'pprof', '-top', '-nodefraction=0.1', temp_file],
                capture_output=True,
                text=True,
                timeout=10
            )
            
            if result.returncode == 0:
                return result.stdout
            else:
                print(f"Warning: go tool pprof failed: {result.stderr}")
                return None
        except Exception as e:
            print(f"Warning: Could not analyze pprof profile: {e}")
            return None
    
    def capture_cpu_profile(self, duration=5):
        """Capture a CPU profile"""
        print(f"\nCapturing CPU profile for {duration} seconds...")
        profile_data = self.get_pprof_profile('profile', duration)
        if profile_data:
            analysis = self.analyze_pprof_profile(profile_data, 'cpu')
            self.profile_data['cpu_profile'] = analysis
            return analysis
        return None
    
    def capture_heap_profile(self):
        """Capture a heap profile"""
        profile_data = self.get_pprof_profile('heap', 0)
        if profile_data:
            analysis = self.analyze_pprof_profile(profile_data, 'heap')
            self.profile_data['heap_profile'] = analysis
            return analysis
        return None
    
    # ==================== Log Analysis ====================
    
    def parse_cometbft_logs(self, log_file_path=None, lookback_lines=500):
        """Parse CometBFT logs to extract timing information"""
        try:
            if log_file_path is None:
                # Try common log locations
                possible_paths = [
                    Path.home() / '.cometbft' / 'logs' / 'cometbft.log',
                    Path.home() / '.cosmos' / 'logs' / 'cometbft.log',
                    Path('/var/log/cometbft.log'),
                ]
                for p in possible_paths:
                    if p.exists():
                        log_file_path = p
                        break
            
            if not log_file_path or not Path(log_file_path).exists():
                print(f"Warning: Could not find CometBFT log file")
                return {}
            
            with open(log_file_path, 'r') as f:
                lines = f.readlines()[-lookback_lines:]
            
            log_analysis = {
                'proposals': [],
                'prevotes': [],
                'precommits': [],
                'commits': [],
                'errors': []
            }
            
            for line in lines:
                # Parse proposal times
                if 'proposing' in line.lower() or 'propose' in line.lower():
                    log_analysis['proposals'].append(line.strip())
                
                # Parse prevote times
                if 'prevote' in line.lower():
                    log_analysis['prevotes'].append(line.strip())
                
                # Parse precommit times
                if 'precommit' in line.lower():
                    log_analysis['precommits'].append(line.strip())
                
                # Parse commit times
                if 'commit' in line.lower() and 'committed' in line.lower():
                    log_analysis['commits'].append(line.strip())
                
                # Track errors
                if 'error' in line.lower() or 'failed' in line.lower():
                    log_analysis['errors'].append(line.strip())
            
            return log_analysis
        except Exception as e:
            print(f"Warning: Error parsing logs: {e}")
            return {}
    
    # ==================== Monitoring Loop ====================
    
    def monitor_once(self):
        """Perform one monitoring cycle"""
        status = self.get_current_block()
        if not status:
            return None
        
        current_height = int(status['result']['sync_info']['latest_block_height'])
        current_time_str = status['result']['sync_info']['latest_block_time']
        current_time = datetime.fromisoformat(current_time_str.replace('Z', '+00:00'))
        
        # Fetch consensus metrics
        metrics_text = self.fetch_prometheus_metrics()
        consensus_metrics = self.extract_consensus_metrics(metrics_text)
        self.profile_data['latest_metrics'] = consensus_metrics
        
        if self.last_block_height is not None and current_height > self.last_block_height:
            # Calculate time between blocks using the block timestamps, not wall clock time
            block_time = (current_time - self.last_block_time).total_seconds()
            self.block_times.append(block_time)
            
            if 'mempool_size' in consensus_metrics:
                self.mempool_sizes.append(consensus_metrics['mempool_size'])
            
            if 'consensus_round' in consensus_metrics:
                self.consensus_rounds.append(consensus_metrics['consensus_round'])
            
            # Get detailed metrics for the new block
            block_data = self.get_block_details(current_height)
            block_results = self.get_block_results(current_height)
            metrics = self.calculate_block_metrics(block_data, block_results)
            
            if 'num_transactions' in metrics:
                self.tx_counts.append(metrics['num_transactions'])
            if 'block_size' in metrics:
                self.block_sizes.append(metrics['block_size'])
            if 'gas_used' in metrics and metrics['gas_used'] is not None:
                self.gas_used.append(metrics['gas_used'])
            
            return {
                'height': current_height,
                'block_time': block_time,
                'metrics': metrics,
                'consensus_metrics': consensus_metrics,
                'timestamp': datetime.now().isoformat()
            }
        
        # Always update these for the next cycle
        self.last_block_height = current_height
        self.last_block_time = current_time
        return None
    
    # ==================== Statistics & Reporting ====================
    
    def print_statistics(self):
        """Print comprehensive statistics"""
        if not self.block_times:
            print("\nNo block data collected yet.")
            return
        
        print("\n" + "="*70)
        print("COMPREHENSIVE BLOCK PERFORMANCE ANALYSIS")
        print("="*70)
        
        # Block timing statistics
        print(f"\n[BLOCK TIMING]")
        print(f"  Average:  {statistics.mean(self.block_times):.2f}s")
        print(f"  Min:      {min(self.block_times):.2f}s")
        print(f"  Max:      {max(self.block_times):.2f}s")
        print(f"  Median:   {statistics.median(self.block_times):.2f}s")
        if len(self.block_times) > 1:
            print(f"  Std Dev:  {statistics.stdev(self.block_times):.2f}s")
        
        # Transaction statistics
        if self.tx_counts:
            print(f"\n[TRANSACTION METRICS]")
            print(f"  Avg Transactions:  {statistics.mean(self.tx_counts):.1f}")
            print(f"  Min:               {min(self.tx_counts)}")
            print(f"  Max:               {max(self.tx_counts)}")
        
        # Block size statistics
        if self.block_sizes:
            print(f"\n[BLOCK SIZE]")
            print(f"  Average:  {statistics.mean(self.block_sizes):,.0f} bytes")
            print(f"  Min:      {min(self.block_sizes):,} bytes")
            print(f"  Max:      {max(self.block_sizes):,} bytes")
        
        # Gas usage statistics
        if self.gas_used:
            print(f"\n[GAS USAGE]")
            print(f"  Average:  {statistics.mean(self.gas_used):,.0f}")
            print(f"  Min:      {min(self.gas_used):,}")
            print(f"  Max:      {max(self.gas_used):,}")
        
        # Consensus metrics
        if self.mempool_sizes:
            print(f"\n[MEMPOOL]")
            print(f"  Average Size:  {statistics.mean(self.mempool_sizes):.0f}")
            print(f"  Min:           {min(self.mempool_sizes)}")
            print(f"  Max:           {max(self.mempool_sizes)}")
        
        if self.consensus_rounds:
            print(f"\n[CONSENSUS ROUNDS]")
            print(f"  Average:  {statistics.mean(self.consensus_rounds):.2f}")
            print(f"  Min:      {min(self.consensus_rounds)}")
            print(f"  Max:      {max(self.consensus_rounds)}")
        
        # Module timing breakdown from latest metrics
        if self.profile_data.get('latest_metrics'):
            metrics = self.profile_data['latest_metrics']
            if 'begin_blocker_modules' in metrics:
                print(f"\n[BEGIN_BLOCKER TIMING BY MODULE (cumulative ms)]")
                for module, time_ms in sorted(metrics['begin_blocker_modules'].items(), key=lambda x: x[1], reverse=True):
                    print(f"  {module:20s}: {time_ms:>12,.1f}ms")
            
            if 'end_blocker_modules' in metrics:
                print(f"\n[END_BLOCKER TIMING BY MODULE (cumulative ms)]")
                for module, time_ms in sorted(metrics['end_blocker_modules'].items(), key=lambda x: x[1], reverse=True):
                    print(f"  {module:20s}: {time_ms:>12,.1f}ms")
        
        # Profile analysis
        if 'cpu_profile' in self.profile_data:
            print(f"\n[CPU PROFILE - TOP FUNCTIONS]")
            if self.profile_data['cpu_profile']:
                lines = self.profile_data['cpu_profile'].split('\n')[:15]
                for line in lines:
                    print(f"  {line}")
        
        print("\n" + "="*70)
    
    def run(self, duration=MONITOR_DURATION, capture_profile_at=None):
        """Run the monitor for a specified duration"""
        print(f"Starting Cosmos Advanced Block Performance Monitor")
        print(f"RPC URL: {self.rpc_url}")
        print(f"Metrics URL: {self.metrics_url}")
        print(f"pprof URL: {self.pprof_url}")
        print(f"Monitoring duration: {duration} seconds")
        print(f"Sample interval: {SAMPLE_INTERVAL} seconds")
        print("="*70)
        
        # Test connectivity
        print("\n[TESTING CONNECTIVITY]")
        status = self.get_current_block()
        if not status:
            print("ERROR: Cannot connect to RPC endpoint")
            return
        print(f"✓ RPC endpoint reachable")
        
        metrics = self.fetch_prometheus_metrics()
        if metrics:
            print(f"✓ Metrics endpoint reachable")
        else:
            print(f"⚠ Metrics endpoint not reachable at {self.metrics_url}")
            print(f"  (This is optional - continuing without it)")
        
        print("="*70 + "\n")
        
        start_time = time.time()
        blocks_observed = 0
        
        try:
            while time.time() - start_time < duration:
                result = self.monitor_once()
                
                if result:
                    blocks_observed += 1
                    print(f"\n[{result['timestamp']}]")
                    print(f"Height: {result['height']}")
                    print(f"Block Time: {result['block_time']:.2f}s")
                    print(f"Transactions: {result['metrics'].get('num_transactions', 'N/A')}")
                    gas = result['metrics'].get('gas_used')
                    if gas is not None:
                        print(f"Gas Used: {gas:,}")
                    print(f"Block Size: {result['metrics'].get('block_size', 'N/A')} bytes")
                    if 'mempool_size' in result['consensus_metrics']:
                        print(f"Mempool Size: {result['consensus_metrics']['mempool_size']}")
                    if 'consensus_round' in result['consensus_metrics']:
                        print(f"Consensus Round: {result['consensus_metrics']['consensus_round']}")
                
                # Capture profile at specified time
                if capture_profile_at and blocks_observed == capture_profile_at:
                    self.capture_cpu_profile(5)
                
                time.sleep(SAMPLE_INTERVAL)
        
        except KeyboardInterrupt:
            print("\n\nMonitor interrupted by user")
        
        print(f"\n\nBlocks observed: {blocks_observed}")
        
        # Parse logs
        print("\n[ANALYZING COMETBFT LOGS]")
        log_analysis = self.parse_cometbft_logs()
        if log_analysis:
            print(f"Recent proposals: {len(log_analysis['proposals'])}")
            print(f"Recent prevotes: {len(log_analysis['prevotes'])}")
            print(f"Recent precommits: {len(log_analysis['precommits'])}")
            print(f"Recent commits: {len(log_analysis['commits'])}")
            print(f"Recent errors: {len(log_analysis['errors'])}")
            if log_analysis['errors']:
                print("\nLast 5 errors:")
                for error in log_analysis['errors'][-5:]:
                    print(f"  {error}")
        
        self.print_statistics()

def main():
    monitor = CosmosAdvancedMonitor()
    # Run for 5 minutes, optionally capture CPU profile after 10 blocks
    monitor.run(duration=MONITOR_DURATION, capture_profile_at=10)

if __name__ == "__main__":
    main()