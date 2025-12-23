# Setup Scripts

This directory contains scripts for setting up and maintaining Tellor Layer nodes.

---

## install_layer.sh

`install_layer.sh` is a comprehensive installation script that can help with setting up new nodes. 

### What it does:
1) Detect platform (linux or mac. Use WSL on Windows.) 
2) Download the latest layerd and cosmovisor binaries to `~/layer`.
3) Initialize and configure the layer node in `~/.layer` or `~/.layer_palmito`
4) Add cosmovisor variables to .bashrc (or .zshrc if mac).
5) Download and install the latest pre-built snapshot from layer-node.com.
6) (linux only) Export an example systemd service file with example commands for installation.

### Usage

1) Download or create the install_layer.sh script. Give the script permission to execute:

```bash
chmod +x install_layer.sh
```

2) Run the script:

```bash
./install_layer.sh [NETWORK] [NODE_MONIKER]
```

**Arguments:**
- `NETWORK` (required): `mainnet` or `palmito` depending on if you want a mainnet node or a testnet node.
- `NODE_MONIKER` (optional): If you provide the NODE_MONIKER, the script will initialize your node with this moniker. Additionally, an account with this name will be created or imported if you have a valid mnemonic. 

**Flags:**

`--snapshot` Allows for using a pre-built snapshot that you downloaded before running the script.

```bash
./install_layer.sh [NETWORK] --snapshot /home/user/path/to/layer_snapshot.tar
```

`--no-snapshot` If you want to skip downloading a snapshot with the script. (You will need to configure syncing manually.)

```bash
./install_layer.sh [NETWORK] --no-snapshot
```

---

## resync-prune-node.sh

`resync-prune-node.sh` is a maintenance script for resetting and resyncing an existing Tellor Layer node's chain data. This is useful when your node's database has grown too large or become corrupted, and you need to prune it by starting fresh with a recent snapshot.

**⚠️ Important:** This script is for Linux only and must be run with sudo privileges.

### What it does:
1) Validates your existing node installation and configuration
2) Sets up a temporary second node (`layer_snapshot`) with offset ports
3) Downloads the latest snapshot from layer-node.com (or uses a pre-downloaded one)
4) Extracts and syncs the snapshot data in the temporary node
5) Waits for the temporary node to fully sync with the network
6) Stops both the main node and temporary node
7) Backs up your old chain data to `~/tmp/layer_data/`
8) Replaces the main node's chain data with the synced snapshot data
9) Restores your validator state file (`priv_validator_state.json`)
10) Restarts the main node and verifies it syncs successfully
11) Sends Discord notifications at key steps (if webhook provided)

### Prerequisites

Before running this script, you must have:
- An existing Tellor Layer node installed and running
- The node configured as a systemd service at `/etc/systemd/system/layer.service`
- Layer home directory at `~/.layer`
- At least 200GB of free disk space
- The layerd binary installed at `~/layer/binaries/[VERSION]/layerd`

### Usage

**Basic usage (downloads snapshot automatically):**

```bash
sudo ./resync-prune-node.sh --network tellor-1
```

or for testnet:

```bash
sudo ./resync-prune-node.sh --network layertest-4
```

**With pre-downloaded snapshot:**

```bash
sudo ./resync-prune-node.sh --network tellor-1 --snapshot /path/to/snapshot.tar
```

**With Discord notifications:**

```bash
sudo ./resync-prune-node.sh --network tellor-1 --discord-webhook "https://discord.com/api/webhooks/YOUR_WEBHOOK_URL"
```

### Command-line Options

- `--network, -n` (required): Network to use. Must be either `tellor-1` (mainnet) or `layertest-4` (testnet)
- `--snapshot, -s` (optional): Path to a pre-downloaded snapshot file. If not provided, the script will download the latest snapshot automatically
- `--discord-webhook, -d` (optional): Discord webhook URL for receiving status notifications during the resync process
- `--help, -h`: Display help message

### Examples

**Mainnet resync with Discord alerts:**

```bash
sudo ./resync-prune-node.sh --network tellor-1 --discord-webhook "https://discord.com/api/webhooks/123456/abcdef"
```

**Testnet resync with pre-downloaded snapshot:**

```bash
sudo ./resync-prune-node.sh --network layertest-4 --snapshot ~/downloads/layertest-4-snapshot.tar
```

### What to Expect

The script will:
- Check that your node is fully synced before starting (won't run if catching up)
- Verify the network matches your node's chain ID
- Check for sufficient disk space (requires 200GB minimum)
- Create a temporary systemd service `layer_snapshot.service`
- Download and extract a 60-80GB snapshot (if not provided)
- Wait for the temporary node to sync (may take several hours depending on snapshot age)
- Pause for confirmation before replacing your main node's data
- Preserve your validator state to prevent double-signing
- Verify the main node starts and syncs successfully after data replacement

### Safety Features

- Validates node is not catching up before starting
- Backs up old chain data to `~/tmp/layer_data/` (can be deleted after successful resync)
- Preserves `priv_validator_state.json` to prevent validator double-signing
- Verifies each step completes successfully before proceeding
- Stops execution immediately if any critical step fails
- Provides detailed debug output for troubleshooting

### Disk Space Management

The script requires significant disk space:
- Snapshot download: ~60-80GB
- Snapshot extraction: ~60-80GB (temporary)
- Old chain data backup: ~40-80GB (in `~/tmp/layer_data/`)
- **Total required: ~200GB minimum**

After successful resync, you can safely delete:
- `~/tmp/layer_data/` (old chain data backup)
- `~/.layer_snapshot/` (temporary node directory)

### Troubleshooting

If the script fails, check:
- `sudo journalctl -u layer_snapshot -f` - View temporary node logs
- `sudo journalctl -u layer -f` - View main node logs
- Ensure you have enough disk space
- Verify your systemd service is properly configured
- Check that the layerd binary exists at the expected path

### Time Estimates

- Snapshot download: 30-60 minutes (depends on internet speed)
- Snapshot extraction: 10-20 minutes
- Temporary node sync: 1-6 hours (depends on how old the snapshot is)
- Data replacement: 5-10 minutes
- Main node resync: 5-15 minutes

**Total time: 2-8 hours** (mostly automated, requires minimal interaction)
