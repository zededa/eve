#!/bin/sh

# Copyright (c) 2026 Zededa, Inc.
# SPDX-License-Identifier: Apache-2.0

# Stage 1 scaffold for NVIDIA DGX Spark (GB10 Grace Blackwell).
# Spark uses NVIDIA's datacenter driver stack on UEFI arm64, not L4T/Tegra.
# GPU userspace bundling and module probing will be added once the driver
# extraction path is implemented.

VENDOR="/opt/vendor/nvidia"

export PATH="$PATH:/hostfs/bin"
export LD_LIBRARY_PATH="/hostfs/lib"

modprobe nvidia 2>/dev/null || true
modprobe nvidia_modeset 2>/dev/null || true
modprobe nvidia_uvm 2>/dev/null || true
modprobe nvidia_drm 2>/dev/null || true

mkdir -p /dev/dri/by-path

if [ -d "${VENDOR}/etc/udev/rules.d" ] && [ -n "$(ls "${VENDOR}"/etc/udev/rules.d/ 2>/dev/null)" ]; then
    mkdir -p /run/udev/rules.d/
    cp "${VENDOR}"/etc/udev/rules.d/* /run/udev/rules.d/
    udevadm control --reload
fi

echo "add" > /sys/module/nvidia/uevent 2>/dev/null || true
echo "add" > /sys/module/nvidia_modeset/uevent 2>/dev/null || true
echo "add" > /sys/module/nvidia_uvm/uevent 2>/dev/null || true
