#!/usr/bin/bash
set -ouex pipefail

source /ctx/build_files/packages.env
: "${JUSTVOXEL_HWE_PACKAGES:?JUSTVOXEL_HWE_PACKAGES must be set}"

read -r -a hwe_packages <<< "${JUSTVOXEL_HWE_PACKAGES}"

# justvoxel-base intentionally ships external repositories disabled.
# Enable EPEL only for composition; finalize-hwe.sh disables it again.
dnf --enablerepo=epel install -y "${hwe_packages[@]}"

# EL10 bootc stores vendor groups under /usr/lib/group. Keep the package-declared
# NUT service account memberships usable for common USB/serial UPS hardware.
if rpm -q nut >/dev/null 2>&1 && getent passwd nut >/dev/null 2>&1; then
    for group_name in tty dialout; do
        grep -q "^${group_name}:" /usr/lib/group
    done
    awk -F: -v OFS=: '
    $1 == "tty" || $1 == "dialout" {
        count = split($4, members, ",")
        found = 0
        for (i = 1; i <= count; i++) {
            if (members[i] == "nut")
                found = 1
        }
        if (!found)
            $4 = ($4 == "" ? "nut" : $4 ",nut")
    }
    { print }
    ' /usr/lib/group > /tmp/justvoxel-group
    install -o root -g root -m0644 /tmp/justvoxel-group /usr/lib/group
    rm -f /tmp/justvoxel-group
fi

for nut_file in /etc/ups/upsd.conf /etc/ups/upsd.users; do
    if [[ -f "${nut_file}" ]]; then
        chown root:nut "${nut_file}"
        chmod 0640 "${nut_file}"
    fi
done

for unit in nut-server.service nut-monitor.service nut-driver@.service; do
    systemctl disable "${unit}" 2>/dev/null || true
done

install -d -m0755 /usr/libexec/justvoxel/health
install -m0755 /ctx/build_files/validate/hwe.sh /usr/libexec/justvoxel/health/hwe
