#!/usr/bin/bash
set -euo pipefail

test "$(cat /usr/lib/justvoxel/variant)" = "justvoxel-baremetal"

rpm -q \
    nut nut-client btrfs-progs smartmontools smartmontools-selinux nvme-cli \
    lm_sensors ethtool usbutils pciutils dmidecode fwupd fwupd-efi >/dev/null

for cmd in upsc btrfs smartctl nvme sensors ethtool lsusb lspci dmidecode fwupdmgr; do
    command -v "${cmd}" >/dev/null
 done

for unit in nut-server.service nut-monitor.service; do
    [[ "$(systemctl is-enabled "${unit}" 2>/dev/null || true)" != "enabled" ]]
done
