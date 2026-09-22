#!/usr/bin/bash
set -euo pipefail

test "$(cat /usr/lib/justvoxel/variant)" = "justvoxel-hws"

# HWS intentionally inherits the complete VM-ready JustVoxel Base, including
# guest tooling, and adds the physical-machine administration delta below.
rpm -q \
    nut nut-client smartmontools smartmontools-selinux nvme-cli \
    lm_sensors ethtool usbutils pciutils dmidecode fwupd fwupd-efi udisks2 \
    NetworkManager-wifi microcode_ctl amd-ucode-firmware atheros-firmware \
    brcmfmac-firmware iwlwifi-dvm-firmware iwlwifi-mvm-firmware realtek-firmware \
    mt7xxx-firmware hdparm powertop >/dev/null

for cmd in upsc smartctl nvme sensors ethtool lsusb lspci dmidecode fwupdmgr udisksctl hdparm powertop; do
    command -v "${cmd}" >/dev/null
done

for unit in nut-server.service nut-monitor.service; do
    [[ "$(systemctl is-enabled "${unit}" 2>/dev/null || true)" != "enabled" ]]
done

# HWS owns its update trust. The parent Base deliberately does not.
grep -Fq 'ghcr.io/home-server-project/justvoxel-hws:' /etc/containers/registries.d/ghcr.io-home-server-project.yaml
