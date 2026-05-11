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

## Local build notes

The Stage 2 driver extractor
([pkg/nvidia/scripts/spark/extract-driver.sh](../pkg/nvidia/scripts/spark/extract-driver.sh))
fetches the NVIDIA arm64 SBSA repo at RUN time. On **Linux CI hosts** this
works out of the box. On **Docker Desktop for macOS/Windows** the buildkit
RUN sandbox typically lacks DNS, so the fetch falls back to a no-GPU-userspace
build — verified working in this repo when bypassed with:

```sh
docker buildx build --network=host --platform linux/arm64 \
  --build-arg PLATFORM=nvidia-spark --target build pkg/nvidia
```

(Tested 2026-05: produced a 206 MB `/rootfs-dist/` containing
`libcuda.so.580.65.06`, `libnvidia-ml.so.580.65.06`, `nvidia-smi`, etc.)

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

- **Stage 1 — Platform scaffold.** `nvidia-spark` PLATFORM wired into the
  Makefile, linuxkit build templates, modifier yq, runme.sh, and CI matrices.
  L4T tarball extraction gated off for spark. ✅
- **Stage 2 — GPU userspace (draft).** Draft CDI spec at
  [pkg/nvidia/cdi/spark/dgx-spark.yaml](../pkg/nvidia/cdi/spark/dgx-spark.yaml)
  modelled on the NVIDIA datacenter arm64 driver layout. Driver extractor at
  [pkg/nvidia/scripts/spark/extract-driver.sh](../pkg/nvidia/scripts/spark/extract-driver.sh)
  fetches the arm64 SBSA CUDA repo. `process-cdi.sh` accepts a pre-populated
  rootfs so spark skips the L4T `Linux_for_Tegra/` extraction. ✅ (needs
  hardware validation — see below)
- **Stage 3 — Kernel (gated).** [kernel-commits.mk](../kernel-commits.mk)
  carries a placeholder slot for `eve-kernel-arm64-v6.14-nvidia-spark`.
  Builds use the generic arm64 kernel by default; set `NVIDIA_SPARK_KERNEL=1`
  to opt into the dedicated branch once it exists. ✅ scaffolding,
  ⚠ blocked on actual eve-kernel branch
- **Stage 4 — Virtualization.** No pillar code changes required: PCIe
  passthrough goes through generic vfio-pci binding in
  [pkg/pillar/hypervisor/hypervisor.go](../pkg/pillar/hypervisor/hypervisor.go).
  ✅ confirmed platform-agnostic. ⚠ NVIDIA forums still report KVM GPU
  passthrough is broken on Spark as of 2026-Q2; recommended initial mode is
  `HV=k` (bare-metal Kubernetes).

## Hardware validation checklist

Items that need a real Spark to confirm:

1. **CDI spec** — run `nvidia-ctk cdi generate --output /tmp/spark.yaml` on
   DGX OS and diff against
   [pkg/nvidia/cdi/spark/dgx-spark.yaml](../pkg/nvidia/cdi/spark/dgx-spark.yaml).
   Replace versioned suffixes (`580.00`, `12.6`) with what `nvidia-smi` and
   `nvcc --version` report.
2. **udev rules** — run `udevadm info /dev/nvidia0` and confirm the rules in
   [pkg/nvidia/udev/spark/rules.d/](../pkg/nvidia/udev/spark/rules.d/) match.
3. **Boot test** — `make ZARCH=arm64 HV=kvm PLATFORM=nvidia-spark live-raw`,
   flash, boot, capture serial output.
4. **GPU bringup** — once booted, `modprobe nvidia && nvidia-smi` from the
   debug shell to confirm the bundled userspace works.

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
