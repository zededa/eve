#!/bin/bash
# push-dpc-spark.sh — Push a correct DevicePortConfig to a ZEDEDA-managed NVIDIA Spark device.
#
# Usage:
#   ./tools/push-dpc-spark.sh [OPTIONS]
#
# Options:
#   -u  ZEDEDA controller URL  (default: https://zedcloud.alpha.zededa.net)
#   -p  Project name           (default: ruslan-gb10-spark)
#   -d  Device name            (default: spark-node-01)
#   -t  API token file path    (default: ~/.zededa/token)
#   -e  Ethernet interface     (default: eth0, the RJ45 1GbE port on HP ZGX Nano G1n)
#   -h  Help
#
# Prerequisites:
#   curl, jq
#
# The HP ZGX Nano G1n has:
#   eth0/eth1 — 1GbE RJ45 copper ports  (use these for normal LAN)
#   eth2/eth3 — 25GbE SFP28 fiber ports (need DAC cable or fiber, NOT RJ45)
#   usb0      — piKVM USB CDC-ECM gadget (172.30.30.x, management-only)
#
# This script pushes a DPC that marks usb0 + eth0 as management interfaces,
# bypassing the ZEDEDA UI validation that requires a "Network" object.

set -euo pipefail

CONTROLLER_URL="https://zedcloud.alpha.zededa.net"
PROJECT=""
DEVICE_NAME="spark-node-01"
TOKEN_FILE="${HOME}/.zededa/token"
ETH_IF="eth0"

usage() {
  grep '^#' "$0" | grep -v '#!/' | sed 's/^# \?//'
  exit 0
}

while getopts "u:p:d:t:e:h" opt; do
  case $opt in
    u) CONTROLLER_URL="$OPTARG" ;;
    p) PROJECT="$OPTARG" ;;
    d) DEVICE_NAME="$OPTARG" ;;
    t) TOKEN_FILE="$OPTARG" ;;
    e) ETH_IF="$OPTARG" ;;
    h) usage ;;
    *) usage ;;
  esac
done

# --- Validate dependencies ---
for cmd in curl jq; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: $cmd not found. Install it first."; exit 1; }
done

# --- Load token ---
if [[ ! -f "$TOKEN_FILE" ]]; then
  echo "ERROR: Token file not found: $TOKEN_FILE"
  echo "  Create it with: echo 'YOUR_API_TOKEN' > $TOKEN_FILE && chmod 600 $TOKEN_FILE"
  exit 1
fi
TOKEN=$(cat "$TOKEN_FILE" | tr -d '[:space:]')

BASE_HEADERS=(
  -H "Content-Type: application/json"
  -H "X-API-TOKEN: ${TOKEN}"
)

# --- Resolve device UUID ---
echo "==> Resolving device UUID for '${DEVICE_NAME}'..."
DEVICE_RESP=$(curl -sf "${BASE_HEADERS[@]}" \
  "${CONTROLLER_URL}/api/v1/devices?name=${DEVICE_NAME}" 2>&1) || {
  echo "ERROR: Failed to query devices. Check controller URL and token."
  exit 1
}

DEVICE_UUID=$(echo "$DEVICE_RESP" | jq -r '.list[0].id // empty')
if [[ -z "$DEVICE_UUID" ]]; then
  echo "ERROR: Device '${DEVICE_NAME}' not found. Response:"
  echo "$DEVICE_RESP" | jq . 2>/dev/null || echo "$DEVICE_RESP"
  exit 1
fi
echo "    Device UUID: ${DEVICE_UUID}"

# --- Fetch current device config to get revision ---
echo "==> Fetching current device config..."
DEV_CFG=$(curl -sf "${BASE_HEADERS[@]}" \
  "${CONTROLLER_URL}/api/v1/devices/${DEVICE_UUID}" 2>&1) || {
  echo "ERROR: Failed to fetch device config."
  exit 1
}

REVISION=$(echo "$DEV_CFG" | jq -r '.revision // "1"')
echo "    Current revision: ${REVISION}"

# --- Build the DPC payload ---
# TimePriority must be newer than existing DPCs. Use current UTC time.
TIME_NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

DPC_PAYLOAD=$(jq -n \
  --arg ts "$TIME_NOW" \
  --arg eth "$ETH_IF" \
  '{
    "dpc": {
      "key": "zedagent",
      "timePriority": $ts,
      "ports": [
        {
          "ifname": "usb0",
          "name": "USB-CDC-ECM",
          "isMgmt": true,
          "isL3Port": true,
          "dhcpType": "DHCP_TYPE_CLIENT"
        },
        {
          "ifname": $eth,
          "name": "Management1",
          "isMgmt": true,
          "isL3Port": true,
          "dhcpType": "DHCP_TYPE_CLIENT"
        }
      ]
    }
  }')

echo "==> DPC payload to push:"
echo "$DPC_PAYLOAD" | jq .

# --- Push DPC to controller ---
echo ""
echo "==> Pushing DPC to device ${DEVICE_UUID}..."
PUSH_RESP=$(curl -sf -X PUT "${BASE_HEADERS[@]}" \
  -d "$DPC_PAYLOAD" \
  "${CONTROLLER_URL}/api/v1/devices/${DEVICE_UUID}/config" 2>&1) || {
  echo "ERROR: PUT request failed. Response:"
  echo "$PUSH_RESP"
  exit 1
}

echo "    Response:"
echo "$PUSH_RESP" | jq . 2>/dev/null || echo "$PUSH_RESP"

echo ""
echo "==> Done. EVE device will apply the new DPC within ~60 seconds."
echo "    Monitor nim logs via piKVM console:"
echo "    ssh root@10.208.13.31"
echo "    Then open: https://10.208.13.31/kvm/"
echo ""
echo "    Expected nim log:"
echo "    INFO: Testing priority 0 DevicePortConfig key zedagent"
echo "    INFO: Port usb0: link: up use: mgmt ..."
echo "    INFO: Port ${ETH_IF}: link: up use: mgmt ..."
