#!/bin/bash
# Entrypoint: prepares files and launches layerd under Cosmovisor for automated,
# governance-driven binary upgrades. Non-"start" subcommands run layerd directly.
set -e

# Derive the node home from --home (Cosmovisor needs DAEMON_HOME as an env var;
# it does not read the --home flag). Falls back to LAYER_HOME.
HOME_DIR="${LAYER_HOME}"
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
    if [[ "${args[$i]}" == "--home" ]]; then
        HOME_DIR="${args[$((i + 1))]}"
    elif [[ "${args[$i]}" == --home=* ]]; then
        HOME_DIR="${args[$i]#--home=}"
    fi
done

export DAEMON_NAME="${DAEMON_NAME:-layerd}"
export DAEMON_HOME="${HOME_DIR}"
export DAEMON_RESTART_AFTER_UPGRADE="${DAEMON_RESTART_AFTER_UPGRADE:-true}"
export DAEMON_ALLOW_DOWNLOAD_BINARIES="${DAEMON_ALLOW_DOWNLOAD_BINARIES:-false}"

# only create the priv_validator_state.json if it doesn't exist and the command is start
if [[ $1 == "start" && ! -f ${HOME_DIR}/data/priv_validator_state.json ]]; then
    mkdir -p ${HOME_DIR}/data
    cat <<EOF > ${HOME_DIR}/data/priv_validator_state.json
{
  "height": "0",
  "round": 0,
  "step": 0
}
EOF
fi

if [[ $1 == "start" ]]; then
    # Seed the Cosmovisor layout once: cosmovisor/genesis/bin/layerd + current symlink.
    if [[ ! -d "${DAEMON_HOME}/cosmovisor/genesis/bin" ]]; then
        echo "Initializing cosmovisor at ${DAEMON_HOME}/cosmovisor"
        cosmovisor init /bin/layerd
    fi
    echo "Starting layerd via cosmovisor with command:"
    echo "cosmovisor run $@"
    echo ""
    exec cosmovisor run "$@"
fi

# Non-start subcommands (keys, genesis, version, ...) run layerd directly.
echo "Running layerd directly: /bin/layerd $@"
exec /bin/layerd "$@"
