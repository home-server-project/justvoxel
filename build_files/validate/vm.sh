#!/usr/bin/bash
set -euo pipefail

test "$(cat /usr/lib/justvoxel/variant)" = "justvoxel-vm"

for package in \
    nut nut-client btrfs-progs smartmontools smartmontools-selinux nvme-cli \
    lm_sensors ethtool usbutils pciutils dmidecode fwupd fwupd-efi; do
    if rpm -q "${package}" >/dev/null 2>&1; then
        echo "ERROR: physical-hardware package present in VM image: ${package}" >&2
        exit 1
    fi
done
