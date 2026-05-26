# HP ZGX Nano G1n (NVIDIA Spark) — piKVM Onboarding Guide

## Hardware Overview

The **HP ZGX Nano G1n** running NVIDIA Spark (EVE-OS) has the following network interfaces:

| Interface | Driver     | Type           | Connector | Notes                                          |
|-----------|------------|----------------|-----------|------------------------------------------------|
| eth0      | mlx5_core  | 200GbE         | QSFP28    | P4242-0000 NIC port 0 — NO RJ45, DAC/fiber only |
| eth1      | mlx5_core  | 200GbE         | QSFP28    | P4242-0000 NIC port 1 — NO RJ45, DAC/fiber only |
| eth2      | mlx5_core  | 200GbE         | QSFP28    | P4242-0000 NIC port 0 — NO RJ45, DAC/fiber only |
| eth3      | mlx5_core  | 200GbE         | QSFP28    | P4242-0000 NIC port 1 — NO RJ45, DAC/fiber only |
| usb0      | cdc_ether  | USB gadget     | piKVM USB | 172.30.30.x subnet                             |

**CRITICAL — NO RJ45 PORTS:** The HP ZGX Nano G1n uses **NVIDIA P4242-0000** NICs — 2-port QSFP28 up to 200G. This is **different from the consumer NVIDIA DGX Spark** which has a 10GbE RJ45. All 4 physical ethernet ports on this device require QSFP28 connectors. A standard RJ45 ethernet cable will NEVER produce a link.

**To use physical ethernet you need one of:**
- QSFP28 to QSFP28 DAC (Direct Attach Copper) twinax cable — plugs into QSFP cage on both ends
- QSFP28 → 10GBASE-T RJ45 transceiver module — gives you an RJ45 port from a QSFP cage
- QSFP28 fiber transceiver + fiber cable

**For initial onboarding, use usb0 (piKVM CDC-ECM) as the management path — it works without any special cables.**

## piKVM v4 MINI Setup

### Prerequisites

1. piKVM v4 MINI at a known IP (default in this setup: `10.208.13.31`)
2. piKVM USB OTG cable connected to the HP ZGX Nano G1n USB-C port
3. piKVM HDMI cable connected to the HP ZGX Nano G1n HDMI output

### Required piKVM Configuration

#### 1. Enable USB Ethernet Gadget (USB CDC-ECM)

Edit `/etc/kvmd/override.yaml` on piKVM:

```yaml
otg:
  devices:
    ethernet:
      enabled: true
      driver: cdc_ecm
      host_mac: 48:6f:73:74:50:43
      kvm_mac: 42:61:64:55:53:42
```

Restart kvmd: `systemctl restart kvmd`

#### 2. Fix dnsmasq DNS Advertisement (CRITICAL)

Without this fix, EVE's nim cannot resolve `zedcloud.alpha.zededa.net` via the USB CDC-ECM interface, and onboarding will fail.

Edit `/usr/lib/python3.13/site-packages/kvmd/apps/__init__.py` on piKVM.

Find the dnsmasq invocation (around line 692). Change:
```python
"--dhcp-option=6",
```
to:
```python
"--dhcp-option=6,8.8.8.8,8.8.4.4",
```

To make the edit (filesystem is read-only by default):
```bash
ssh root@10.208.13.31
rw   # remount filesystem read-write
sed -i 's/"--dhcp-option=6",/"--dhcp-option=6,8.8.8.8,8.8.4.4",/' \
  /usr/lib/python3.13/site-packages/kvmd/apps/__init__.py
systemctl restart kvmd
ro   # remount read-only
```

Verify: check that piKVM's dnsmasq process shows the DNS option:
```bash
ps aux | grep dnsmasq
# Should include: --dhcp-option=6,8.8.8.8,8.8.4.4
```

#### 3. Configure sudoers for webterm (optional but useful)

To allow the kvmd-webterm user to run diagnostic commands:
```bash
rw
echo 'kvmd-webterm ALL=(ALL) NOPASSWD: /sbin/ip, /sbin/arp, /bin/cat /sys/class/net/*' \
  > /etc/sudoers.d/kvmd-webterm
chmod 440 /etc/sudoers.d/kvmd-webterm
ro
```

### USB CDC-ECM Network Addressing

When configured correctly:
- piKVM (host) side: `172.30.30.1/24`
- EVE device side: `172.30.30.x` (DHCP-assigned, typically `172.30.30.124`)

Check EVE's current IP from piKVM:
```bash
arp -n | grep '172\.30\.30\.'
```

## EVE Bootstrap DPC

EVE reads `/config/DevicePortConfig/bootstrap.json` on first boot, before any controller DPC arrives. This ensures usb0 and eth0 are always tried as management interfaces, even before the controller can push a proper DPC.

The bootstrap file is at `conf/DevicePortConfig/bootstrap-spark.json` in the EVE source tree.

To apply to an EVE installer image:
```bash
# Mount the EVE config partition and copy bootstrap DPC
cp conf/DevicePortConfig/bootstrap-spark.json \
   /mnt/eve-config/DevicePortConfig/bootstrap.json
```

## ZEDEDA Controller Configuration

### Project Setup

1. Create project: `ruslan-gb10-spark` (or client-specific name)
2. Device registration: use the onboard certificate from `conf/onboard.cert.pem`
3. Set device name: `spark-node-01` (or client convention)

### DPC Push via API

The ZEDEDA UI requires a "Network" object for management interfaces — this validation can block the first-time DPC configuration. Use the API script instead:

```bash
./tools/push-dpc-spark.sh \
  -u https://zedcloud.alpha.zededa.net \
  -d spark-node-01 \
  -t ~/.zededa/token \
  -e eth0
```

This pushes a DPC with usb0 (USB CDC-ECM) and eth0 as management interfaces.

## Automated Onboarding

Run the full onboarding automation:

```bash
./tools/onboard-spark.sh \
  -k 10.208.13.31 \
  -c https://zedcloud.alpha.zededa.net \
  -n spark-node-01 \
  -t ~/.zededa/token \
  -e eth0
```

## Troubleshooting

### "No management interfaces" in nim logs

The controller pushed a DPC with no management interfaces set. This happens when the ZEDEDA UI auto-generates the initial DPC.

**Fix:** Run `push-dpc-spark.sh` to push a correct DPC via API.

**Verify:** Check nim logs via piKVM KVM console (`https://10.208.13.31/kvm/`):
```
INFO: Testing priority 0 DevicePortConfig key zedagent
INFO: Port usb0: link: up use: mgmt 172.30.30.124
INFO: Port eth0: link: up use: mgmt 192.168.x.x
```

### eth0/eth1/eth2/eth3 show NO-CARRIER

The HP ZGX Nano G1n has **NO RJ45 ethernet ports**. All 4 NICs are NVIDIA P4242-0000 (2-port QSFP28, up to 200G). A standard RJ45 ethernet cable will never produce a link on any port.

To use physical ethernet you need:
- **QSFP28 DAC cable** — twinax cable with QSFP28 connectors on both ends (cheapest, for short runs)
- **QSFP28 → 10GBASE-T transceiver** — lets you use a regular RJ45 cable into a QSFP28 cage
- **QSFP28 SR transceiver + fiber** — for longer runs

For initial onboarding, use `usb0` via piKVM CDC-ECM (172.30.30.x). No special cables needed.

### EVE cannot resolve zedcloud.alpha.zededa.net

- piKVM dnsmasq DNS fix not applied (see section above)
- Check: `ps aux | grep dnsmasq` on piKVM should show `--dhcp-option=6,8.8.8.8,8.8.4.4`

### SSH to EVE fails

EVE uses key-only SSH authentication. Default piKVM image has no EVE SSH keys.

**Workaround:** Use piKVM KVM video interface at `https://10.208.13.31/kvm/`

**Permanent fix:** Embed operator SSH public key in EVE installer at `/config/authorized_keys`

### piKVM KVM keyboard issue (uppercase / special chars)

The piKVM KVM `type` tool converts `&&` → `77` and drops uppercase letters.

**Workaround:**
- Use semicolons `;` instead of `&&`
- Avoid commands with uppercase letters when typing via KVM interface
- SSH to piKVM and use `ssh root@172.30.30.124` from there (requires SSH key setup)

## ZEDEDA API Token Setup

```bash
mkdir -p ~/.zededa
# Obtain token from ZEDEDA controller UI → Profile → API Keys
echo "YOUR_API_TOKEN_HERE" > ~/.zededa/token
chmod 600 ~/.zededa/token
```
