# HP ZGX Nano G1n (NVIDIA Spark) — piKVM Onboarding Guide

## Hardware Overview

The **HP ZGX Nano G1n** running NVIDIA Spark (EVE-OS) has the following network interfaces:

| Interface | Driver     | Type          | Connector | Notes                          |
|-----------|------------|---------------|-----------|--------------------------------|
| eth0      | igc        | 1GbE copper   | RJ45      | Standard LAN cable goes here   |
| eth1      | igc        | 1GbE copper   | RJ45      | Secondary LAN                  |
| eth2      | mlx5_core  | 25GbE fiber   | SFP28     | DAC or fiber only, NOT RJ45    |
| eth3      | mlx5_core  | 25GbE fiber   | SFP28     | DAC or fiber only, NOT RJ45    |
| usb0      | cdc_ether  | USB gadget    | piKVM USB | 172.30.30.x subnet             |

**IMPORTANT:** eth2 and eth3 are SFP28 fiber ports. Plugging an RJ45 (copper ethernet) cable into them will NOT work. Always use eth0 or eth1 for standard 1GbE LAN connections.

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

### eth0/eth1 show NO-CARRIER

- Verify cable is in an **RJ45 port** (eth0 or eth1), not the SFP28 cages (eth2/eth3)
- Verify the switch port on the other end shows a link LED
- Try a different cable

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
