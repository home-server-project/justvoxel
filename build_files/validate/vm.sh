#!/usr/bin/bash
set -euo pipefail

test "$(cat /usr/lib/justvoxel/variant)" = "justvoxel-vm"

# VM keeps network-share clients and guarded partition tooling because a second
# virtual disk or NFS/SMB backup target is part of the supported storage model.
rpm -q nfs-utils cifs-utils parted xfsprogs util-linux >/dev/null

# pciutils is intentionally allowed in the VM image because open-vm-tools
# depends on it. The remaining packages below are physical-hardware
# administration tools that should stay out of the VM variant.
for package in \
    nut nut-client btrfs-progs smartmontools smartmontools-selinux nvme-cli \
    lm_sensors ethtool usbutils dmidecode fwupd fwupd-efi \
    NetworkManager-wifi microcode_ctl amd-ucode-firmware atheros-firmware \
    brcmfmac-firmware iwlwifi-dvm-firmware iwlwifi-mvm-firmware realtek-firmware \
    mt7xxx-firmware hdparm powertop; do
    if rpm -q "${package}" >/dev/null 2>&1; then
        echo "ERROR: physical-hardware package present in VM image: ${package}" >&2
        exit 1
    fi
done
