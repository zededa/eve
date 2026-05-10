# NVIDIA DGX Spark (GB10 Grace Blackwell) — EVE-OS port

> **Status: experimental scaffolding (Stage 1).** The `nvidia-spark` PLATFORM
> is wired into the build system, but GPU userspace, kernel, and KVM/passthrough
> support are not yet implemented. See [Roadmap](#roadmap).

NVIDIA DGX Spark is a GB10-based desktop AI computer (20 Arm cores —
Cortex-X925/A725 — plus a Blackwell GPU, 128 GB unified LPDDR5X). Unlike
the Jetson family it ships with **DGX OS 7** (Ubuntu 24.04 derivative) and the
**datacenter NVIDIA driver/CUDA stack on arm64**, not Jetpack/L4T. It boots
via standard UEFI rather than the Tegra `flash.sh` / QSPI / cboot flow used by
Xavier/Orin.

## Differences vs. Jetson

| | Jetson (Orin / Xavier NX) | DGX Spark |
|---|---|---|
| BSP | Jetpack / L4T tarball | None (DGX OS via apt) |
| Boot | Tegra UEFI in QSPI/eMMC | Standard arm64 UEFI |
| GPU stack | L4T iGPU driver | Datacenter NVIDIA driver |
| Container runtime | nvidia-container-toolkit + CDI | Same |
| KVM GPU passthrough | Supported on Jetpack 5/6/7 | Not officially supported as of 2026-Q2 |
| MIG / vGPU | No | No |

## Building

> **Stage 1 only:** This produces a bootable EVE arm64 UEFI image with the
> `nvidia-spark` platform marker but **no GPU support**. Use it to validate
> that EVE comes up on Spark hardware before adding GPU bringup.

```sh
make ZARCH=arm64 HV=kvm PLATFORM=nvidia-spark live-raw
make ZARCH=arm64 HV=kvm PLATFORM=nvidia-spark installer-raw
```

The resulting `dist/arm64/current/live.raw` (or `installer.raw`) can be flashed
to a USB stick. Spark's UEFI should pick it up from the boot menu.

> **Tip:** if the device does not boot from USB automatically, press `ESC`
> during init to enter the UEFI Boot Manager and pick the USB stick manually.

## Roadmap

- **Stage 1 (this commit)** — `nvidia-spark` PLATFORM scaffold. Empty
  `pkg/nvidia/cdi/spark/` and `udev/spark/rules.d/`. L4T tarball extraction is
  skipped. The build produces an EVE image with the platform marker but no
  GPU userspace.
- **Stage 2 — GPU userspace.** Extract the NVIDIA datacenter driver and CUDA
  runtime from a working DGX OS install, generate a CDI spec on real hardware
  with `nvidia-ctk cdi generate`, and bundle the libs under
  `/opt/vendor/nvidia/`. Replaces the empty `cdi/spark/` and `udev/spark/`.
- **Stage 3 — Kernel.** Decide between forking Ubuntu HWE 6.14 with the NVIDIA
  arm64 patches into [eve-kernel](https://github.com/lf-edge/eve-kernel)
  (`eve-kernel-arm64-v6.14-nvidia-spark`) or tracking DGX OS's kernel directly.
- **Stage 4 — Virtualization.** Determine whether GPU passthrough is achievable
  or whether the port must be Kubernetes-only (`HV=k`) until NVIDIA enables
  it.

## Open questions / blockers

These need either NVIDIA confirmation or hands-on Spark access:

1. Redistribution license for the datacenter NVIDIA driver/CUDA arm64 packages
   in an EVE image (Jetpack tarballs were the workable redistributable path
   for Jetson).
2. Whether the NVIDIA arm64 kernel patches are public (kernel.org / Ubuntu
   git) or only available as binary modules against Ubuntu HWE.
3. Whether Spark exposes the GPU via ACPI (likely, given UEFI server-class
   boot) or DT, and how it presents to the guest under KVM.
4. Whether Secure Boot is enforced and signed kernel images are required.

## Hardware references

- [DGX Spark User Guide](https://docs.nvidia.com/dgx/dgx-spark/dgx-spark.pdf)
- [DGX OS 7 user guide](https://docs.nvidia.com/dgx/dgx-os-7-user-guide/)
- [DGX Spark Porting Guide](https://docs.nvidia.com/dgx/dgx-spark-porting-guide/)
