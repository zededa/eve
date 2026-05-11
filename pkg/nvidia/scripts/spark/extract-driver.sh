#!/bin/sh

# Copyright (c) 2026 Zededa, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Extract the NVIDIA datacenter driver + CUDA runtime for arm64/SBSA into
# $DEST_ROOTFS for DGX Spark image builds. Replaces the L4T tarball
# extraction used for Jetson platforms.
#
# **DRAFT** — NVIDIA's redistribution license for the arm64 SBSA driver/CUDA
# packages must be confirmed before this script can be run in CI. For local
# builds, ensure you have accepted the EULA at
# https://developer.nvidia.com/cuda-downloads.
#
# Inputs:
#   $1 — destination directory (the "rootfs" the CDI processing step will
#        read files from). Must exist.
#
# What it does:
#   1. Fetches the NVIDIA CUDA arm64 SBSA apt repo metadata for Ubuntu 24.04.
#   2. Downloads the nvidia-driver-* and cuda-runtime-* .debs.
#   3. Extracts them into the destination.

set -e

DEST=${1:?"usage: $0 <dest-rootfs>"}
mkdir -p "$DEST"

# NVIDIA datacenter arm64 SBSA repo (Ubuntu 24.04 base — matches DGX OS 7).
REPO_URL="https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/sbsa"
DRIVER_BRANCH=${NVIDIA_DRIVER_BRANCH:-580}

# Packages we need for a minimal runtime. The exact set must be validated
# against `dpkg -l | grep nvidia` on a working DGX OS install.
PACKAGES="
  libnvidia-compute-${DRIVER_BRANCH}
  libnvidia-decode-${DRIVER_BRANCH}
  libnvidia-encode-${DRIVER_BRANCH}
  libnvidia-gl-${DRIVER_BRANCH}
  nvidia-utils-${DRIVER_BRANCH}
  cuda-cudart-12-6
  cuda-nvrtc-12-6
"

WORK=$(mktemp -d)
cd "$WORK"

# Fetch the package index
curl -fsSL "${REPO_URL}/Packages" -o Packages || {
    echo "WARN: cannot fetch NVIDIA arm64 SBSA package index — skipping."
    echo "      The Spark image will be built without GPU userspace."
    exit 0
}

# Resolve and download each package
for pkg in $PACKAGES; do
    deb=$(awk -v p="$pkg" '/^Package: /{name=$2} /^Filename: /{if(name==p){print $2; exit}}' Packages)
    if [ -z "$deb" ]; then
        echo "WARN: package $pkg not found in repo — skipping."
        continue
    fi
    curl -fsSL "${REPO_URL}/${deb}" -o "$(basename "$deb")"
done

# Extract every .deb into the destination
for d in *.deb; do
    [ -f "$d" ] || continue
    dpkg -x "$d" "$DEST"
done

cd /
rm -rf "$WORK"

echo "Spark driver/CUDA extraction complete → $DEST"
