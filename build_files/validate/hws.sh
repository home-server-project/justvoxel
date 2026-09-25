#!/usr/bin/bash
set -euo pipefail

test "$(cat /usr/lib/justvoxel/variant)" = "justvoxel-hws"

# HWS explicitly installs these 22 physical-hardware packages.
rpm -q \
    nut nut-client libusb1-devel smartmontools smartmontools-selinux nvme-cli \
    lm_sensors ethtool usbutils dmidecode fwupd-efi udisks2 \
    NetworkManager-wifi amd-ucode-firmware atheros-firmware \
    brcmfmac-firmware iwlwifi-dvm-firmware iwlwifi-mvm-firmware \
    realtek-firmware mt7xxx-firmware hdparm powertop >/dev/null

# HWS requires these capabilities but inherits them from its parent stack.
rpm -q pciutils fwupd microcode_ctl >/dev/null

for cmd in upsc nut-scanner smartctl nvme sensors ethtool lsusb lspci dmidecode fwupdmgr udisksctl hdparm powertop; do
    command -v "${cmd}" >/dev/null
done

test -e /usr/lib64/libusb-1.0.so

for unit in nut-server.service nut-monitor.service; do
    [[ "$(systemctl is-enabled "${unit}" 2>/dev/null || true)" != "enabled" ]]
done

# Final HWS filesystem and image-trust state.
test "$(stat -c '%a %U %G' /var/tmp)" = "1777 root root"

POLICY=/etc/containers/policy.json
REGISTRY_CONFIG=/etc/containers/registries.d/ghcr.io-home-server-project.yaml
JUSTVOXEL_BASE_REPOSITORY=ghcr.io/home-server-project/justvoxel-base
JUSTVOXEL_HWS_REPOSITORY=ghcr.io/home-server-project/justvoxel-hws
JUSTVOXEL_REPOSITORY_PREFIX=ghcr.io/home-server-project/justvoxel-

jq empty "${POLICY}"
test -f /usr/lib/pki/containers/home-server-project.pub
test -f "${REGISTRY_CONFIG}"

for trust_repository in \
    "${JUSTVOXEL_BASE_REPOSITORY}" \
    "${JUSTVOXEL_HWS_REPOSITORY}"; do
    grep -Fq "${trust_repository}:" "${REGISTRY_CONFIG}"
done

expected_trust="$(printf '%s\n' "${JUSTVOXEL_BASE_REPOSITORY}" "${JUSTVOXEL_HWS_REPOSITORY}" | sort)"
actual_trust="$(jq -r \
    --arg prefix "${JUSTVOXEL_REPOSITORY_PREFIX}" \
    '(.transports.docker // {}) | keys[] | select(startswith($prefix))' \
    "${POLICY}" | sort)"
[[ "${actual_trust}" == "${expected_trust}" ]] || {
    echo "ERROR: HWS image trust is not scoped to the exact Base + HWS repository set." >&2
    printf 'Expected:\n%s\nActual:\n%s\n' "${expected_trust}" "${actual_trust}" >&2
    exit 1
}

grep -Fq "use-sigstore-attachments: true" "${REGISTRY_CONFIG}"
