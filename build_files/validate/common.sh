#!/usr/bin/bash
set -euo pipefail

for cmd in \
    bootc podman skopeo nmcli nmtui resolvectl firewall-cmd sshd sudo just mjust \
    tailscale netbird curl jq openssl tar gzip rsync ping dig traceroute nc tcpdump lsof \
    findmnt mountpoint flock mkfs.xfs mount.nfs mount.cifs qemu-ga vmtoolsd iperf3; do
    command -v "${cmd}" >/dev/null
done

rpm -q \
    NetworkManager NetworkManager-tui systemd-resolved firewalld openssh-server sudo \
    podman skopeo just container-selinux policycoreutils-python-utils selinux-policy-extra \
    util-linux xfsprogs iperf3 nfs-utils cifs-utils qemu-guest-agent open-vm-tools \
    hyperv-daemons >/dev/null

semodule -l >/dev/null

test "$(systemctl is-enabled NetworkManager.service)" = "enabled"
test "$(systemctl is-enabled systemd-resolved.service)" = "enabled"
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

test -f /etc/NetworkManager/conf.d/90-systemd-resolved.conf
grep -Fqx 'dns=systemd-resolved' /etc/NetworkManager/conf.d/90-systemd-resolved.conf
test -f /usr/lib/tmpfiles.d/justvoxel-resolved.conf
grep -Fq '/run/systemd/resolve/stub-resolv.conf' /usr/lib/tmpfiles.d/justvoxel-resolved.conf

test -f /etc/profile.d/zz-justvoxel-prompt.sh
grep -Fq '38;5;82' /etc/profile.d/zz-justvoxel-prompt.sh

test -f /usr/lib/justvoxel/variant
test -f /usr/lib/tmpfiles.d/justvoxel.conf

test -x /usr/libexec/justvoxel/minecraft-backup
bash -n /usr/libexec/justvoxel/minecraft-backup
for template in \
    /usr/share/justvoxel/templates/quadlets/minecraft.container.in \
    /usr/share/justvoxel/templates/config/minecraft.env.in \
    /usr/share/justvoxel/templates/config/minecraft-backup.env.in \
    /usr/share/justvoxel/templates/systemd/minecraft-backup.service.in \
    /usr/share/justvoxel/templates/systemd/minecraft-backup.timer.in; do
    test -f "${template}"
done

test -x /usr/bin/mjust
test -f /usr/share/justvoxel/mjust/justfile
for script in /usr/libexec/justvoxel/mjust/*; do
    [[ -f "${script}" ]] || continue
    bash -n "${script}"
done
/usr/bin/mjust --list >/dev/null

# The bootc image ships only immutable source templates and management logic.
# Active, administrator-owned runtime files are created later by `mjust setup`.
test ! -e /etc/containers/systemd/minecraft.container
test ! -e /etc/justvoxel/minecraft.env
test ! -e /etc/justvoxel/justvoxel.conf

if dnf repolist --enabled | grep -Eiq 'epel|tailscale|netbird'; then
    echo "ERROR: external package repository enabled in completed image" >&2
    exit 1
fi
