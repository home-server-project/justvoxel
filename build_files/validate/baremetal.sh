#!/usr/bin/bash
set -euo pipefail

test "$(cat /usr/lib/justvoxel/variant)" = "justvoxel-baremetal"

rpm -q \
    nut nut-client btrfs-progs smartmontools smartmontools-selinux nvme-cli \
    lm_sensors ethtool usbutils pciutils dmidecode fwupd fwupd-efi \
    NetworkManager-wifi microcode_ctl amd-ucode-firmware atheros-firmware \
    brcmfmac-firmware iwlwifi-dvm-firmware iwlwifi-mvm-firmware realtek-firmware \
    mt7xxx-firmware hdparm powertop >/dev/null

for cmd in upsc btrfs smartctl nvme sensors ethtool lsusb lspci dmidecode fwupdmgr hdparm powertop; do
    command -v "${cmd}" >/dev/null
done

for unit in nut-server.service nut-monitor.service; do
    [[ "$(systemctl is-enabled "${unit}" 2>/dev/null || true)" != "enabled" ]]
done
