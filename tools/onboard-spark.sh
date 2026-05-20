#!/bin/bash
# onboard-spark.sh — Full onboarding automation for HP ZGX Nano G1n (NVIDIA Spark / EVE-OS)
#                    using piKVM v4 MINI as the out-of-band management interface.
#
# Usage:
#   ./tools/onboard-spark.sh [OPTIONS]
#
# Options:
#   -k  piKVM IP address        (default: 10.208.13.31)
#   -c  ZEDEDA controller URL   (default: https://zedcloud.alpha.zededa.net)
#   -p  ZEDEDA project name     (required)
#   -n  Device name             (default: spark-node-01)
#   -t  API token file          (default: ~/.zededa/token)
#   -e  Ethernet interface      (default: eth0)
#   -h  Help
#
# What this script does:
#   1. Verifies piKVM connectivity
#   2. Verifies EVE is reachable over USB CDC-ECM (172.30.30.x)
#   3. Verifies EVE has reached ZEDEDA controller (onboarding state)
#   4. Pushes corrected DPC (usb0 + eth0 as management)
#   5. Waits for device to come online in controller
#   6. Prints final status
#
# Prerequisites:
#   curl, jq, ssh (with piKVM root key or password)
#
# HARDWARE NOTE for HP ZGX Nano G1n:
#   eth0/eth1 — 1GbE RJ45 copper (standard LAN cable goes here)
#   eth2/eth3 — 25GbE SFP28 fiber (CANNOT use RJ45 cable here!)
#   usb0      — piKVM USB CDC-ECM gadget (172.30.30.x subnet)
#
# piKVM USB CDC-ECM:
#   piKVM side: 172.30.30.1
#   EVE side:   172.30.30.x (assigned via DHCP from piKVM dnsmasq)
#   DNS fix:    piKVM dnsmasq must advertise DNS with --dhcp-option=6,8.8.8.8,8.8.4.4
#               See: pkg/pillar/docs/zededa-spark-pikvm-setup.md

set -euo pipefail

PIKVM_IP="10.208.13.31"
PIKVM_USER="root"
PIKVM_PASSWORD="root"
CONTROLLER_URL="https://zedcloud.alpha.zededa.net"
PROJECT=""
DEVICE_NAME="spark-node-01"
TOKEN_FILE="${HOME}/.zededa/token"
ETH_IF="eth0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
  grep '^#' "$0" | grep -v '#!/' | sed 's/^# \?//'
  exit 0
}

while getopts "k:c:p:n:t:e:h" opt; do
  case $opt in
    k) PIKVM_IP="$OPTARG" ;;
    c) CONTROLLER_URL="$OPTARG" ;;
    p) PROJECT="$OPTARG" ;;
    n) DEVICE_NAME="$OPTARG" ;;
    t) TOKEN_FILE="$OPTARG" ;;
    e) ETH_IF="$OPTARG" ;;
    h) usage ;;
    *) usage ;;
  esac
done

# Color helpers
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail() { echo -e "${RED}[FAIL]${NC} $*"; exit 1; }
info() { echo -e "[INFO] $*"; }

echo "=========================================="
echo "  ZEDEDA Spark Onboarding"
echo "  Device: ${DEVICE_NAME}"
echo "  Controller: ${CONTROLLER_URL}"
echo "=========================================="

# --- Step 1: Verify piKVM ---
info "Step 1: Checking piKVM at ${PIKVM_IP}..."
if ! curl -sk --max-time 5 "https://${PIKVM_IP}/api/info" >/dev/null 2>&1; then
  fail "Cannot reach piKVM at ${PIKVM_IP}. Check network connectivity."
fi
ok "piKVM is reachable"

# --- Step 2: Check USB CDC-ECM link ---
info "Step 2: Checking USB CDC-ECM link on piKVM..."
CDC_STATUS=$(sshpass -p "${PIKVM_PASSWORD}" ssh -o StrictHostKeyChecking=no \
  "${PIKVM_USER}@${PIKVM_IP}" \
  "cat /sys/class/net/usb0/operstate 2>/dev/null || echo unknown" 2>/dev/null || echo "unknown")

if [[ "$CDC_STATUS" != "up" ]]; then
  warn "USB CDC-ECM interface usb0 is '${CDC_STATUS}' on piKVM."
  warn "EVE device may be powered off or not yet booted."
  warn "Continuing — will retry connectivity checks..."
else
  ok "USB CDC-ECM (usb0) is UP on piKVM"
fi

# --- Step 3: Find EVE IP on CDC-ECM subnet ---
info "Step 3: Finding EVE IP address on 172.30.30.x..."
EVE_IP=""
for attempt in 1 2 3; do
  EVE_IP=$(sshpass -p "${PIKVM_PASSWORD}" ssh -o StrictHostKeyChecking=no \
    "${PIKVM_USER}@${PIKVM_IP}" \
    "arp -n 2>/dev/null | grep '172\.30\.30\.' | awk '{print \$1}' | head -1" 2>/dev/null || echo "")
  if [[ -n "$EVE_IP" ]]; then
    break
  fi
  info "  Attempt ${attempt}/3: EVE not found yet, waiting 10s..."
  sleep 10
done

if [[ -z "$EVE_IP" ]]; then
  warn "EVE IP not found via ARP. This may mean:"
  warn "  - EVE has not booted yet (boot takes 2-5 minutes)"
  warn "  - USB CDC-ECM gadget not loaded on EVE"
  warn "  - Bootstrap DPC not applied"
  warn "Skipping EVE-side checks. You can run push-dpc-spark.sh separately once EVE is up."
else
  ok "EVE found at ${EVE_IP}"
fi

# --- Step 4: Push corrected DPC ---
info "Step 4: Pushing corrected DPC via ZEDEDA API..."
if [[ ! -f "$TOKEN_FILE" ]]; then
  warn "Token file not found: ${TOKEN_FILE}"
  warn "Skipping DPC push. Run manually:"
  warn "  ${SCRIPT_DIR}/push-dpc-spark.sh -d ${DEVICE_NAME} -e ${ETH_IF}"
else
  "${SCRIPT_DIR}/push-dpc-spark.sh" \
    -u "${CONTROLLER_URL}" \
    -d "${DEVICE_NAME}" \
    -t "${TOKEN_FILE}" \
    -e "${ETH_IF}" && ok "DPC pushed successfully" || warn "DPC push failed — device may not be onboarded yet"
fi

# --- Step 5: Wait for device online ---
info "Step 5: Waiting for device '${DEVICE_NAME}' to appear online in controller..."
TOKEN=$(cat "${TOKEN_FILE}" 2>/dev/null | tr -d '[:space:]' || echo "")
if [[ -z "$TOKEN" ]]; then
  warn "No token — skipping controller status check"
else
  ONLINE=false
  for attempt in $(seq 1 12); do
    DEV_STATE=$(curl -sf \
      -H "X-API-TOKEN: ${TOKEN}" \
      "${CONTROLLER_URL}/api/v1/devices?name=${DEVICE_NAME}" 2>/dev/null | \
      jq -r '.list[0].adminState // "unknown"' 2>/dev/null || echo "unknown")
    if [[ "$DEV_STATE" == "ACTIVE" ]] || [[ "$DEV_STATE" == "active" ]]; then
      ONLINE=true
      break
    fi
    info "  Attempt ${attempt}/12: device state=${DEV_STATE}, waiting 30s..."
    sleep 30
  done
  if $ONLINE; then
    ok "Device '${DEVICE_NAME}' is ACTIVE in controller"
  else
    warn "Device not yet ACTIVE after 6 minutes. Check controller UI or nim logs via piKVM console."
  fi
fi

echo ""
echo "=========================================="
echo "  Onboarding Summary"
echo "  Device name: ${DEVICE_NAME}"
echo "  piKVM:       ${PIKVM_IP}"
echo "  EVE IP:      ${EVE_IP:-not found}"
echo "  Controller:  ${CONTROLLER_URL}"
echo "=========================================="
echo ""
echo "Next steps:"
echo "  1. Verify ethernet cable is in eth0 or eth1 RJ45 port (NOT the SFP28 cages)"
echo "  2. Check nim logs:  ssh root@${PIKVM_IP}  then open https://${PIKVM_IP}/kvm/"
echo "  3. View device in controller UI: ${CONTROLLER_URL}"
echo ""
