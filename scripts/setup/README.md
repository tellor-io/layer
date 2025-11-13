# install_layer.sh

install_layer.sh is a comprehensive installation script that can help with setting up new nodes. 

Here is a breakdown of what this script does:
1) Detect platform (linux or mac. Use WSL on Windows.) 
2) Download the latest layerd and cosmovisor binaries to `~/layer`.
3) initialize and configure the layer node in `~/.layer` or `~/.layer_palmito`
4) Add cosmovisor variables to .bashrc (or .zshrc if mac).
5) Download and install the latest pre-built snapshot from layer-node.com.
6) (linux only) Export an example systemd service file with example commands for installation.

## Usage

1) Download or create the install_layer.sh script. Give the script permission to execute:

```bash
chmod +x install_layer.sh
```

2) Run the script:

```bash
./install_layer.sh [NETWORK] [NODE_MONIKER]
```
Arguments:
    NETWORK (required): `mainnet` or `palmito` depending on if you want a mainnet node or a testnet node.
    NODE_MONIKER (optional):  If you provide the NODE_MONIKER, the script will initialize your node with this moniker. Additionally, an account with this name will be created or imported if you have a valid mnemonic. 

Flags:

`--snapshot ` Allows for using a pre-built snapshot that you downloaded before running the script.

```bash
./install_layer.sh [NETWORK] --snapshot /home/user/path/to/layer_snapshot.tar
```

`--no-snapshot` If you want to skip downloading a snapshot with the script. (You will need to configure syncing manually.)

```bash
./install_layer.sh [NETWORK] --no-snapshot
```
