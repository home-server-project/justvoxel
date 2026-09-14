#!/usr/bin/bash
set -ouex pipefail

source /ctx/build_files/packages.env
: "${JUSTVOXEL_COMMON_PACKAGES:?JUSTVOXEL_COMMON_PACKAGES must be set}"
: "${TAILSCALE_PACKAGE:?TAILSCALE_PACKAGE must be set}"
: "${NETBIRD_PACKAGE:?NETBIRD_PACKAGE must be set}"

cp -avf /ctx/system_files/. /

if ! dnf repolist --enabled | grep -Eiq '(^|[[:space:]])crb([[:space:]]|$)'; then
    echo "ERROR: AlmaLinux CRB repository is not enabled."
    exit 1
fi

# EPEL provides upstream just on EL10. External repositories are build inputs;
# finalize-image.sh disables them in the deployed immutable appliance.
dnf install -y epel-release curl
read -r -a common_packages <<< "${JUSTVOXEL_COMMON_PACKAGES}"
dnf install -y "${common_packages[@]}"

# Remote-access clients are intentionally present because adding host RPMs after
# installation is not the JustVoxel management model. Both remain disabled and
# unconfigured until an administrator explicitly enrolls one.
curl -fsSL \
    https://pkgs.tailscale.com/stable/rhel/10/tailscale.repo \
    -o /etc/yum.repos.d/tailscale.repo
sed -ri 's/^enabled=1/enabled=0/' /etc/yum.repos.d/tailscale.repo || true
dnf --enablerepo=tailscale-stable install -y "${TAILSCALE_PACKAGE}"
systemctl disable tailscaled.service 2>/dev/null || true

cat > /etc/yum.repos.d/netbird.repo <<'REPO'
[netbird]
name=NetBird
baseurl=https://pkgs.netbird.io/yum/
enabled=0
gpgcheck=1
gpgkey=https://pkgs.netbird.io/yum/repodata/repomd.xml.key
repo_gpgcheck=1
REPO

dnf --setopt=tsflags=noscripts --enablerepo=netbird install -y "${NETBIRD_PACKAGE}"
systemctl disable netbird.service 2>/dev/null || true

systemctl enable NetworkManager.service 2>/dev/null || true
systemctl enable systemd-resolved.service
systemctl enable firewalld.service 2>/dev/null || true
systemctl enable sshd.service 2>/dev/null || true

install -d -m0755 /usr/share/doc/justvoxel
cp -avf /ctx/docs/. /usr/share/doc/justvoxel/

install -d -m0755 /usr/share/justvoxel/templates
cp -avf /ctx/templates/. /usr/share/justvoxel/templates/

install -d -m0755 /usr/libexec/justvoxel
install -m0755 /ctx/runtime/minecraft-backup /usr/libexec/justvoxel/minecraft-backup

install -d -m0755 /usr/libexec/justvoxel/health
install -m0755 /ctx/build_files/validate/common.sh /usr/libexec/justvoxel/health/common
install -m0755 /ctx/build_files/validate/vm.sh /usr/libexec/justvoxel/health/vm
install -m0755 /ctx/build_files/validate/baremetal.sh /usr/libexec/justvoxel/health/baremetal

for cmd in \
    bootc podman skopeo nmcli nmtui resolvectl firewall-cmd sshd sudo just \
    tailscale netbird curl jq findmnt mountpoint flock mkfs.xfs mount.nfs mount.cifs \
    qemu-ga vmtoolsd iperf3; do
    command -v "${cmd}"
done

bash -n /usr/libexec/justvoxel/minecraft-backup
semodule -l >/dev/null
