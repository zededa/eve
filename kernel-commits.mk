KERNEL_COMMIT_amd64_next_generic = 3152b8fe36ea
KERNEL_COMMIT_amd64_v6.12.49_generic = b21691f37d8d
KERNEL_COMMIT_arm64_v5.10.192_nvidia-jp5 = be4828dfe4e1
KERNEL_COMMIT_arm64_v5.15.136_nvidia-jp6 = dbf1688d4cb1
KERNEL_COMMIT_arm64_v6.8.12_nvidia-jp7 = a9757a5802c2
# DGX Spark (GB10) — placeholder slot. Replace with the actual commit once
# a `eve-kernel-arm64-v6.14-nvidia-spark` branch exists in lf-edge/eve-kernel
# (Ubuntu HWE 6.14 base + NVIDIA arm64 datacenter driver patches).
# Until then, kernel-version.mk routes nvidia-spark to the generic arm64
# kernel; enable the dedicated branch by passing NVIDIA_SPARK_KERNEL=1 to
# make.
KERNEL_COMMIT_arm64_v6.14_nvidia-spark = 000000000000
KERNEL_COMMIT_arm64_v6.1.155_generic = 04d922430698
KERNEL_COMMIT_riscv64_v6.1.112_generic = a6bc06835103
