#!/usr/bin/env python3
"""
Debug script to check where gas data is located in block_results
"""

import requests
import json
import sys

def check_gas_location(height=None):
    """Check where gas data is in block_results."""
    rpc_url = "http://localhost:26657"
    
    # Get current height if not specified
    if not height:
        status = requests.get(f"{rpc_url}/status").json()
        height = int(status['result']['sync_info']['latest_block_height'])
    
    print(f"Checking block {height}...")
    print("="*60)
    
    # Get block results
    response = requests.get(f"{rpc_url}/block_results?height={height}")
    block_results = response.json()
    
    if 'result' not in block_results:
        print("ERROR: No result in block_results")
        return
    
    result = block_results['result']
    
    # Check top-level structure
    print("\n1. TOP-LEVEL KEYS:")
    print(f"   {list(result.keys())}")
    
    # Check for gas at top level
    print("\n2. TOP-LEVEL GAS FIELDS:")
    gas_used = result.get('gas_used')
    gas_wanted = result.get('gas_wanted')
    print(f"   gas_used: {gas_used} (type: {type(gas_used).__name__})")
    print(f"   gas_wanted: {gas_wanted} (type: {type(gas_wanted).__name__})")
    
    # Check transaction results
    if 'txs_results' in result:
        tx_results = result['txs_results']
        print(f"\n3. TRANSACTION RESULTS:")
        print(f"   Number of txs: {len(tx_results)}")
        
        if tx_results:
            print(f"\n4. FIRST TX STRUCTURE:")
            first_tx = tx_results[0]
            print(f"   Keys: {list(first_tx.keys())}")
            
            tx_gas_used = first_tx.get('gas_used')
            tx_gas_wanted = first_tx.get('gas_wanted')
            print(f"   gas_used: {tx_gas_used} (type: {type(tx_gas_used).__name__ if tx_gas_used else 'None'})")
            print(f"   gas_wanted: {tx_gas_wanted} (type: {type(tx_gas_wanted).__name__ if tx_gas_wanted else 'None'})")
            
            # Try summing all txs
            total_gas_used = 0
            total_gas_wanted = 0
            for tx in tx_results:
                try:
                    tx_gas = tx.get('gas_used', 0)
                    total_gas_used += int(tx_gas) if tx_gas else 0
                    tx_wanted = tx.get('gas_wanted', 0)
                    total_gas_wanted += int(tx_wanted) if tx_wanted else 0
                except:
                    pass
            
            print(f"\n5. SUMMED FROM ALL TXS:")
            print(f"   Total gas_used: {total_gas_used}")
            print(f"   Total gas_wanted: {total_gas_wanted}")
    else:
        print("\n3. NO txs_results FIELD")
    
    # Pretty print a sample
    print(f"\n6. RAW STRUCTURE (first 1000 chars):")
    print(json.dumps(result, indent=2)[:1000])
    
    print("\n" + "="*60)
    print("RECOMMENDATION:")
    if gas_used and gas_used != "0":
        print("  ✅ Use top-level result.gas_used and result.gas_wanted")
    elif total_gas_used > 0:
        print("  ✅ Sum from individual tx_results[].gas_used")
    else:
        print("  ⚠️  No gas data found - might need to check different location")

if __name__ == '__main__':
    height = None
    if len(sys.argv) > 1:
        height = int(sys.argv[1])
    check_gas_location(height)

