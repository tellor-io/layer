#!/bin/bash
set -euo pipefail

NODE="tcp://localhost:26657"
RPC_URL="http://localhost:26657"
PER_PAGE=100

usage() {
    cat <<EOF
Usage: $(basename "$0") <tellor_address>

Query the jail/unjail history for a validator on the Layer chain.
Uses binary search over historical block state to find the exact heights
where the validator's jailed status changed.

Arguments:
  tellor_address    A tellor account (tellor1...) or validator operator
                    (tellorvaloper1...) address.

Options:
  -h, --help        Show this help message and exit.

Requirements:
  - layerd binary in PATH
  - jq installed
  - curl installed
  - A running Layer node at localhost:26657
EOF
    exit 0
}

# ── argument handling ────────────────────────────────────────────────
[[ $# -lt 1 || "$1" == "-h" || "$1" == "--help" ]] && usage
INPUT_ADDR="$1"

# ── dependency check ─────────────────────────────────────────────────
for cmd in layerd jq curl; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "Error: '$cmd' is required but not found in PATH." >&2
        exit 1
    fi
done

# ── address resolution ───────────────────────────────────────────────
resolve_addresses() {
    local addr="$1"

    if [[ "$addr" == tellorvaloper1* ]]; then
        VALOPER="$addr"
        ACCOUNT=$(layerd debug addr "$addr" 2>/dev/null | grep "Bech32 Acc:" | awk '{print $3}')
    elif [[ "$addr" == tellor1* ]]; then
        ACCOUNT="$addr"
        VALOPER=$(layerd debug addr "$addr" 2>/dev/null | grep "Bech32 Val:" | awk '{print $3}')
    else
        echo "Error: address must start with 'tellor1' or 'tellorvaloper1'." >&2
        echo "       Got: $addr" >&2
        echo "" >&2
        echo "Hint: run 'layerd debug addr <your_address>' to inspect the format." >&2
        exit 1
    fi

    if [[ -z "$VALOPER" || -z "$ACCOUNT" ]]; then
        echo "Error: could not derive addresses from '$addr'." >&2
        echo "       Output of 'layerd debug addr':" >&2
        layerd debug addr "$addr" >&2
        exit 1
    fi
}

resolve_addresses "$INPUT_ADDR"

# ── query validator info (current) ───────────────────────────────────
VALIDATOR_JSON=$(layerd query staking validator "$VALOPER" --node "$NODE" -o json 2>/dev/null) || {
    echo "Error: could not query validator '$VALOPER'." >&2
    echo "       Is the node running at $NODE?" >&2
    exit 1
}

MONIKER=$(echo "$VALIDATOR_JSON" | jq -r '.validator.description.moniker // "unknown"')
JAILED=$(echo "$VALIDATOR_JSON" | jq -r 'if .validator.jailed then "true" else "false" end')
STATUS_RAW=$(echo "$VALIDATOR_JSON" | jq -r '.validator.status // "unknown"')
TOKENS=$(echo "$VALIDATOR_JSON" | jq -r '.validator.tokens // "unknown"')

status_label() {
    case "$1" in
        1|BOND_STATUS_UNBONDED)   echo "UNBONDED" ;;
        2|BOND_STATUS_UNBONDING)  echo "UNBONDING" ;;
        3|BOND_STATUS_BONDED)     echo "BONDED" ;;
        *)                        echo "$1" ;;
    esac
}
STATUS=$(status_label "$STATUS_RAW")

# Handle both Protobuf JSON ("@type"/"key") and Amino JSON ("type"/"value")
CONSPUB_TYPE=$(echo "$VALIDATOR_JSON" | jq -r '.validator.consensus_pubkey."@type" // empty')
CONSPUB_KEY=$(echo "$VALIDATOR_JSON" | jq -r '.validator.consensus_pubkey.key // empty')
AMINO_TYPE=$(echo "$VALIDATOR_JSON" | jq -r '.validator.consensus_pubkey.type // empty')
AMINO_VALUE=$(echo "$VALIDATOR_JSON" | jq -r '.validator.consensus_pubkey.value // empty')

if [[ -n "$CONSPUB_TYPE" && -n "$CONSPUB_KEY" ]]; then
    CONSPUB_JSON="{\"@type\":\"${CONSPUB_TYPE}\",\"key\":\"${CONSPUB_KEY}\"}"
elif [[ -n "$AMINO_TYPE" && -n "$AMINO_VALUE" ]]; then
    case "$AMINO_TYPE" in
        tendermint/PubKeyEd25519)   PROTO_TYPE="/cosmos.crypto.ed25519.PubKey" ;;
        tendermint/PubKeySecp256k1) PROTO_TYPE="/cosmos.crypto.secp256k1.PubKey" ;;
        *)                          PROTO_TYPE="$AMINO_TYPE" ;;
    esac
    CONSPUB_JSON="{\"@type\":\"${PROTO_TYPE}\",\"key\":\"${AMINO_VALUE}\"}"
else
    echo "Error: could not extract consensus pubkey from validator info." >&2
    echo "       Raw consensus_pubkey field:" >&2
    echo "$VALIDATOR_JSON" | jq '.validator.consensus_pubkey' >&2
    exit 1
fi

# ── query signing info ───────────────────────────────────────────────
SIGNING_JSON=$(layerd query slashing signing-info "$CONSPUB_JSON" --node "$NODE" -o json 2>/dev/null) || {
    echo "Warning: could not query signing info. Continuing without it." >&2
    SIGNING_JSON=""
}

VALCONS=""
JAILED_UNTIL=""
TOMBSTONED=""
MISSED_BLOCKS=""
START_HEIGHT=""

if [[ -n "$SIGNING_JSON" ]]; then
    VALCONS=$(echo "$SIGNING_JSON" | jq -r '.val_signing_info.address // empty')
    JAILED_UNTIL=$(echo "$SIGNING_JSON" | jq -r '.val_signing_info.jailed_until // empty')
    TOMBSTONED=$(echo "$SIGNING_JSON" | jq -r '.val_signing_info.tombstoned // empty')
    MISSED_BLOCKS=$(echo "$SIGNING_JSON" | jq -r '.val_signing_info.missed_blocks_counter // empty')
    START_HEIGHT=$(echo "$SIGNING_JSON" | jq -r '.val_signing_info.start_height // empty')
fi

# ── get node block range ─────────────────────────────────────────────
NODE_STATUS=$(curl -s "${RPC_URL}/status" 2>/dev/null) || {
    echo "Error: could not reach CometBFT RPC at $RPC_URL" >&2
    exit 1
}
EARLIEST_HEIGHT=$(echo "$NODE_STATUS" | jq -r '.result.sync_info.earliest_block_height')
LATEST_HEIGHT=$(echo "$NODE_STATUS" | jq -r '.result.sync_info.latest_block_height')

if [[ -z "$EARLIEST_HEIGHT" || -z "$LATEST_HEIGHT" ]]; then
    echo "Error: could not determine block range from node status." >&2
    exit 1
fi

# ── print header ─────────────────────────────────────────────────────
SEP="============================================================"
echo ""
echo "$SEP"
echo "  Validator Jail History"
echo "$SEP"
echo ""
printf "  %-14s %s\n" "Moniker:" "$MONIKER"
printf "  %-14s %s\n" "Account:" "$ACCOUNT"
printf "  %-14s %s\n" "Operator:" "$VALOPER"
[[ -n "$VALCONS" ]] && printf "  %-14s %s\n" "Consensus:" "$VALCONS"
echo ""

echo "--- Current Status ---"
printf "  %-18s %s\n" "Jailed:" "$JAILED"
printf "  %-18s %s\n" "Status:" "$STATUS"
printf "  %-18s %s\n" "Tokens:" "$TOKENS"
[[ -n "$TOMBSTONED" ]]  && printf "  %-18s %s\n" "Tombstoned:" "$TOMBSTONED"
[[ -n "$MISSED_BLOCKS" ]] && printf "  %-18s %s\n" "Missed Blocks:" "$MISSED_BLOCKS"
[[ -n "$START_HEIGHT" ]] && printf "  %-18s %s\n" "Start Height:" "$START_HEIGHT"

if [[ -n "$JAILED_UNTIL" && "$JAILED_UNTIL" != "1970-01-01"* ]]; then
    printf "  %-18s %s\n" "Jailed Until:" "$JAILED_UNTIL"
fi
echo ""
printf "  %-18s %s\n" "Node range:" "${EARLIEST_HEIGHT} .. ${LATEST_HEIGHT}"
echo ""

# ── helpers ──────────────────────────────────────────────────────────

# Query the validator's jailed status at a given height. Returns "true" or "false".
query_jailed_at() {
    local h="$1"
    local result
    result=$(layerd query staking validator "$VALOPER" --node "$NODE" --height "$h" -o json 2>/dev/null) || {
        echo "error"
        return
    }
    echo "$result" | jq -r 'if .validator.jailed then "true" else "false" end'
}

declare -A HEIGHT_TIME_CACHE

get_block_time() {
    local height="$1"
    if [[ -n "${HEIGHT_TIME_CACHE[$height]+x}" ]]; then
        echo "${HEIGHT_TIME_CACHE[$height]}"
        return
    fi
    local t
    t=$(curl -s "${RPC_URL}/block?height=${height}" 2>/dev/null \
        | jq -r '.result.block.header.time // empty' 2>/dev/null)
    HEIGHT_TIME_CACHE[$height]="$t"
    echo "$t"
}

# Cache for jailed-status lookups to avoid redundant queries
declare -A JAILED_CACHE

query_jailed_cached() {
    local h="$1"
    if [[ -n "${JAILED_CACHE[$h]+x}" ]]; then
        echo "${JAILED_CACHE[$h]}"
        return
    fi
    local val
    val=$(query_jailed_at "$h")
    JAILED_CACHE[$h]="$val"
    echo "$val"
}

# Binary search for the exact height where jailed status flips from
# $low_jailed (at $low) to something different (at $high).
# Precondition: status at $low == $low_jailed, status at $high != $low_jailed.
bisect() {
    local low=$1
    local high=$2
    local low_jailed="$3"

    while (( high - low > 1 )); do
        local mid=$(( (low + high) / 2 ))
        local mid_jailed
        mid_jailed=$(query_jailed_cached "$mid")
        if [[ "$mid_jailed" == "error" ]]; then
            echo "-1"; return
        fi
        if [[ "$mid_jailed" == "$low_jailed" ]]; then
            low=$mid
        else
            high=$mid
        fi
    done
    echo "$high"
}

# Recursively scan an interval for ALL jailed-status transitions, including
# round-trips (e.g., false→true→false) that look the same at both endpoints.
# Results are appended to $EVENTS_FILE.
# $4 = recursion depth limit (prevents runaway queries)
scan_interval() {
    local low=$1
    local high=$2
    local low_jailed="$3"
    local depth=${4:-12}

    if (( high - low <= 1 )) || (( depth <= 0 )); then
        return
    fi

    local high_jailed
    high_jailed=$(query_jailed_cached "$high")
    if [[ "$high_jailed" == "error" ]]; then return; fi

    if [[ "$low_jailed" != "$high_jailed" ]]; then
        # Endpoints differ: exactly one transition boundary in this interval.
        local transition
        transition=$(bisect "$low" "$high" "$low_jailed")
        if [[ "$transition" != "-1" ]]; then
            local new_jailed
            new_jailed=$(query_jailed_cached "$transition")
            local event_type="UNJAILED"
            [[ "$new_jailed" == "true" ]] && event_type="JAILED"
            echo "  Found: ${event_type} at height ${transition}"
            echo "${transition}||${event_type}|binary search" >> "$EVENTS_FILE"
            # There could be MORE transitions between transition and high
            scan_interval "$transition" "$high" "$new_jailed" $((depth - 1))
        fi
    else
        # Endpoints are the same — probe the midpoint to detect hidden
        # round-trip transitions (e.g., not-jailed → jailed → not-jailed).
        local mid=$(( (low + high) / 2 ))
        local mid_jailed
        mid_jailed=$(query_jailed_cached "$mid")
        if [[ "$mid_jailed" == "error" ]]; then return; fi

        if [[ "$mid_jailed" != "$low_jailed" ]]; then
            # Status differs at midpoint — transitions in BOTH halves
            scan_interval "$low" "$mid" "$low_jailed" $((depth - 1))
            scan_interval "$mid" "$high" "$mid_jailed" $((depth - 1))
        fi
        # If midpoint matches endpoints we can't rule out very narrow
        # windows, but we stop here to keep query count bounded.
    fi
}

# ── find all jailed status transitions ───────────────────────────────
EVENTS_FILE=$(mktemp)
trap 'rm -f "$EVENTS_FILE"' EXIT

# Collect anchor heights from on-chain state to seed the scan.
# These ensure we probe known-interesting regions even if they'd
# otherwise be hidden inside a same-status interval.
UNBONDING_HEIGHT=$(echo "$VALIDATOR_JSON" | jq -r '.validator.unbonding_height // "0"')
ANCHORS=()
[[ -n "$START_HEIGHT" && "$START_HEIGHT" != "0" ]] && ANCHORS+=("$START_HEIGHT")
[[ "$UNBONDING_HEIGHT" != "0" ]] && ANCHORS+=("$UNBONDING_HEIGHT")

# Build sorted, unique list of probe points within the node's range
PROBES=("$EARLIEST_HEIGHT")
for a in "${ANCHORS[@]}"; do
    if (( a > EARLIEST_HEIGHT && a < LATEST_HEIGHT )); then
        PROBES+=("$a")
    fi
done
PROBES+=("$LATEST_HEIGHT")
# Sort and deduplicate
IFS=$'\n' PROBES=($(printf '%s\n' "${PROBES[@]}" | sort -n -u)); unset IFS

echo "Scanning for jail status transitions (binary search)..."
echo "  Node range:    ${EARLIEST_HEIGHT} .. ${LATEST_HEIGHT}"
echo "  Anchor points: ${PROBES[*]}"
echo ""

# Query status at each probe point
declare -a PROBE_STATUS=()
for p in "${PROBES[@]}"; do
    s=$(query_jailed_cached "$p")
    PROBE_STATUS+=("$s")
done

echo "  Status at height ${PROBES[0]}: $(
    [[ "${PROBE_STATUS[0]}" == "true" ]] && echo "jailed" || echo "not jailed"
)"

# Scan each interval between consecutive probe points
for (( i=0; i < ${#PROBES[@]} - 1; i++ )); do
    lo="${PROBES[$i]}"
    hi="${PROBES[$((i+1))]}"
    lo_s="${PROBE_STATUS[$i]}"
    scan_interval "$lo" "$hi" "$lo_s" 14
done

if [[ ! -s "$EVENTS_FILE" ]]; then
    echo "  No status transitions found in available range."
fi

# ── enrich transitions with tx details ───────────────────────────────
echo ""
echo "Searching for related transactions..."

# Search for MsgUnjail txs to attach tx hashes to UNJAILED events
UNJAIL_QUERY="message.action='/cosmos.slashing.v1beta1.MsgUnjail'"
UNJAIL_RESULT=$(curl -s "${RPC_URL}/tx_search?query=\"${UNJAIL_QUERY}\"&per_page=${PER_PAGE}&order_by=\"asc\"" 2>/dev/null) || UNJAIL_RESULT=""

declare -A UNJAIL_TX_MAP
if echo "$UNJAIL_RESULT" | jq -e '.result.txs' &>/dev/null; then
    while IFS='|' read -r txh txhash; do
        UNJAIL_TX_MAP[$txh]="$txhash"
    done < <(echo "$UNJAIL_RESULT" | jq -r --arg acct "$ACCOUNT" --arg valop "$VALOPER" '
        [.result.txs[]? | select(
            .tx_result.events[]? |
            select(.type == "message") |
            .attributes[]? |
            select(.key == "sender") |
            (.value // "") |
            (. == $acct or . == $valop)
        )] | .[] | "\(.height)|\(.hash)"
    ' 2>/dev/null)
    unjail_count="${#UNJAIL_TX_MAP[@]}"
    echo "  MsgUnjail txs found: $unjail_count"
fi

# Search for bridge attestation slash txs
ATT_QUERY="attestation_slashed.operator_address='${VALOPER}'"
ATT_RESULT=$(curl -s "${RPC_URL}/tx_search?query=\"${ATT_QUERY}\"&per_page=${PER_PAGE}&order_by=\"asc\"" 2>/dev/null) || ATT_RESULT=""

declare -A ATT_TX_MAP=()
if echo "$ATT_RESULT" | jq -e '.result.txs' &>/dev/null; then
    while IFS='|' read -r txh txhash; do
        [[ -n "$txh" ]] && ATT_TX_MAP[$txh]="$txhash"
    done < <(echo "$ATT_RESULT" | jq -r '.result.txs[]? | "\(.height)|\(.hash)"' 2>/dev/null)
    att_count="${#ATT_TX_MAP[@]}"
    [[ $att_count -gt 0 ]] && echo "  Attestation slash txs found: $att_count"
fi

# Search for bridge valset signature slash txs
VAL_QUERY="valset_signature_slashed.operator_address='${VALOPER}'"
VAL_RESULT=$(curl -s "${RPC_URL}/tx_search?query=\"${VAL_QUERY}\"&per_page=${PER_PAGE}&order_by=\"asc\"" 2>/dev/null) || VAL_RESULT=""

declare -A VALSIG_TX_MAP=()
if echo "$VAL_RESULT" | jq -e '.result.txs' &>/dev/null; then
    while IFS='|' read -r txh txhash; do
        [[ -n "$txh" ]] && VALSIG_TX_MAP[$txh]="$txhash"
    done < <(echo "$VAL_RESULT" | jq -r '.result.txs[]? | "\(.height)|\(.hash)"' 2>/dev/null)
    valsig_count="${#VALSIG_TX_MAP[@]}"
    [[ $valsig_count -gt 0 ]] && echo "  Valset sig slash txs found: $valsig_count"
fi

# ── resolve timestamps and merge tx details ──────────────────────────
RESOLVED_FILE=$(mktemp)
trap 'rm -f "$EVENTS_FILE" "$RESOLVED_FILE"' EXIT

while IFS='|' read -r height _time event source; do
    time=$(get_block_time "$height")

    # Enrich the source with tx details where available
    if [[ "$event" == "UNJAILED" ]] && [[ ${#UNJAIL_TX_MAP[@]} -gt 0 ]] && [[ -n "${UNJAIL_TX_MAP[$height]+x}" ]]; then
        hash="${UNJAIL_TX_MAP[$height]}"
        source="MsgUnjail (tx: ${hash:0:12}...)"
    elif [[ "$event" == "JAILED" ]] && [[ ${#ATT_TX_MAP[@]} -gt 0 ]] && [[ -n "${ATT_TX_MAP[$height]+x}" ]]; then
        hash="${ATT_TX_MAP[$height]}"
        source="attestation slash (tx: ${hash:0:12}...)"
    elif [[ "$event" == "JAILED" ]] && [[ ${#VALSIG_TX_MAP[@]} -gt 0 ]] && [[ -n "${VALSIG_TX_MAP[$height]+x}" ]]; then
        hash="${VALSIG_TX_MAP[$height]}"
        source="valset sig slash (tx: ${hash:0:12}...)"
    elif [[ "$event" == "JAILED" ]]; then
        source="downtime or other (no matching tx)"
    fi

    echo "${height}|${time}|${event}|${source}" >> "$RESOLVED_FILE"
done < "$EVENTS_FILE"

# Also add any tx-only events that the binary search couldn't find
# (e.g., jail events at pruned heights inferred from tx index)
if [[ ${#ATT_TX_MAP[@]} -gt 0 ]]; then
    for txh in "${!ATT_TX_MAP[@]}"; do
        if ! grep -q "^${txh}|" "$RESOLVED_FILE" 2>/dev/null; then
            time=$(get_block_time "$txh")
            hash="${ATT_TX_MAP[$txh]}"
            echo "${txh}|${time}|JAILED|attestation slash (tx: ${hash:0:12}...)" >> "$RESOLVED_FILE"
        fi
    done
fi
if [[ ${#VALSIG_TX_MAP[@]} -gt 0 ]]; then
    for txh in "${!VALSIG_TX_MAP[@]}"; do
        if ! grep -q "^${txh}|" "$RESOLVED_FILE" 2>/dev/null; then
            time=$(get_block_time "$txh")
            hash="${VALSIG_TX_MAP[$txh]}"
            echo "${txh}|${time}|JAILED|valset sig slash (tx: ${hash:0:12}...)" >> "$RESOLVED_FILE"
        fi
    done
fi

# ── sort and print timeline ──────────────────────────────────────────
echo ""
echo "--- Jail / Unjail Event Timeline ---"

if [[ ! -s "$RESOLVED_FILE" ]]; then
    echo ""
    echo "  No jail/unjail events found in the node's available block range."
    echo ""
else
    SORTED_FILE=$(mktemp)
    trap 'rm -f "$EVENTS_FILE" "$RESOLVED_FILE" "$SORTED_FILE"' EXIT
    sort -t'|' -k1 -n "$RESOLVED_FILE" > "$SORTED_FILE"

    echo ""
    printf "  %-4s %-10s %-12s %-26s %s\n" "#" "Event" "Height" "Time" "Source"
    printf "  %-4s %-10s %-12s %-26s %s\n" "---" "--------" "----------" "------------------------" "------------------------------"

    COUNT=0
    while IFS='|' read -r height time event source; do
        COUNT=$((COUNT + 1))
        short_time="${time%%.*}Z"
        [[ "$short_time" == "Z" ]] && short_time="(unknown)"
        printf "  %-4d %-10s %-12s %-26s %s\n" "$COUNT" "$event" "$height" "$short_time" "$source"
    done < "$SORTED_FILE"
    echo ""
fi

echo "$SEP"
echo ""
