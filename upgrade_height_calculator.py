#!/usr/bin/env python3
"""
Blockchain Upgrade Block Height Calculator

This script calculates the target block height for a blockchain upgrade
based on current block height, average block time, and target upgrade time.
"""

from datetime import datetime, timezone
import zoneinfo
import sys


def calculate_target_block_height(current_height, avg_block_time_seconds, target_datetime):
    """
    Calculate the target block height for an upgrade.
    
    Args:
        current_height (int): Current block height
        avg_block_time_seconds (float): Average time between blocks in seconds
        target_datetime (datetime): Target datetime for the upgrade
    
    Returns:
        int: Target block height
    """
    current_time = datetime.now(zoneinfo.ZoneInfo("America/New_York"))
    
    if target_datetime <= current_time:
        raise ValueError("Target time must be in the future")
    
    time_diff_seconds = (target_datetime - current_time).total_seconds()
    blocks_to_add = int(time_diff_seconds / avg_block_time_seconds)
    
    return current_height + blocks_to_add


def parse_datetime_input(date_str, time_str):
    """Parse date and time strings into a datetime object."""
    try:
        datetime_str = f"{date_str} {time_str}"
        return datetime.strptime(datetime_str, "%Y-%m-%d %H:%M").replace(tzinfo=zoneinfo.ZoneInfo("America/New_York"))
    except ValueError as e:
        raise ValueError(f"Invalid date/time format. Use YYYY-MM-DD for date and HH:MM for time. Error: {e}")


def get_user_input():
    """Get input from user interactively."""
    try:
        current_height = int(input("Enter current block height: "))
        if current_height < 0:
            raise ValueError("Block height must be positive")
        
        avg_block_time = float(input("Enter average block time in seconds: "))
        if avg_block_time <= 0:
            raise ValueError("Block time must be positive")
        
        target_date = input("Enter target date (YYYY-MM-DD): ")
        target_time = input("Enter target time (HH:MM, 24-hour format, Eastern Time): ")
        
        target_datetime = parse_datetime_input(target_date, target_time)
        
        return current_height, avg_block_time, target_datetime
    
    except ValueError as e:
        print(f"Error: {e}")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nOperation cancelled by user")
        sys.exit(1)


def main():
    """Main function to run the calculator."""
    print("=== Blockchain Upgrade Block Height Calculator ===\n")
    
    try:
        current_height, avg_block_time, target_datetime = get_user_input()
        
        target_height = calculate_target_block_height(current_height, avg_block_time, target_datetime)
        
        print(f"\n=== Results ===")
        print(f"Current block height: {current_height:,}")
        print(f"Average block time: {avg_block_time} seconds")
        print(f"Target upgrade time: {target_datetime.strftime('%Y-%m-%d %H:%M ET')}")
        print(f"Target block height: {target_height:,}")
        print(f"Blocks to wait: {target_height - current_height:,}")
        
        current_time = datetime.now(zoneinfo.ZoneInfo("America/New_York"))
        time_until_upgrade = target_datetime - current_time
        days = time_until_upgrade.days
        hours, remainder = divmod(time_until_upgrade.seconds, 3600)
        minutes, _ = divmod(remainder, 60)
        
        print(f"Time until upgrade: {days} days, {hours} hours, {minutes} minutes")
        
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()


