#!/usr/bin/bash
set -euo pipefail

for cmd in \
    bootc podman skopeo nmcli nmtui firewall-cmd sshd sudo just \
    tailscale netbird curl jq openssl tar gzip rsync ping dig traceroute nc tcpdump lsof; do
    command -v "${cmd}" >/dev/null
 done

rpm -q \
    NetworkManager \
    NetworkManager-tui \
    firewalld \
    openssh-server \
    sudo \
    podman \
    skopeo \
    just \
    container-selinux \
    policycoreutils-python-utils \
    selinux-policy-extra >/dev/null

semodule -l >/dev/null

test "$(systemctl is-enabled NetworkManager.service)" = "enabled"
test "$(systemctl is-enabled firewalld.service)" = "enabled"
test "$(systemctl is-enabled sshd.service)" = "enabled"
[[ "$(systemctl is-enabled tailscaled.service 2>/dev/null || true)" != "enabled" ]]
[[ "$(systemctl is-enabled netbird.service 2>/dev/null || true)" != "enabled" ]]

for forbidden in cockpit-system cockpit-files cockpit-podman cockpit-storaged cockpit-machines libvirt-daemon-kvm qemu-kvm virt-install; do
    if rpm -q "${forbidden}" >/dev/null 2>&1; then
        echo "ERROR: unrelated package present in JustVoxel core: ${forbidden}" >&2
        exit 1
    fi
done

test -f /etc/profile.d/zz-justvoxel-prompt.sh
grep -Fq '38;5;82' /etc/profile.d/zz-justvoxel-prompt.sh

test -f /usr/lib/justvoxel/variant
test -f /usr/lib/tmpfiles.d/justvoxel.conf

# Step 1 intentionally contains neither the Minecraft runtime nor mjust yet.
test ! -e /etc/containers/systemd/minecraft.container
test ! -e /usr/bin/mjust

if dnf repolist --enabled | grep -Eiq 'epel|tailscale|netbird'; then
    echo "ERROR: external package repository enabled in completed image" >&2
    exit 1
fi
