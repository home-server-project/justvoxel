#!/usr/bin/bash
set -ouex pipefail

: "${JUSTVOXEL_BASE_REPOSITORY:?JUSTVOXEL_BASE_REPOSITORY must be set}"
: "${JUSTVOXEL_VM_REPOSITORY:?JUSTVOXEL_VM_REPOSITORY must be set}"
: "${JUSTVOXEL_HWE_REPOSITORY:?JUSTVOXEL_HWE_REPOSITORY must be set}"

OS_RELEASE_USR=/usr/lib/os-release
OS_RELEASE_ETC=/etc/os-release
[[ -r "${OS_RELEASE_USR}" ]]
# shellcheck disable=SC1090
source "${OS_RELEASE_USR}"

[[ "${ID:-}" == "justvoxel" ]] || {
    echo "ERROR: expected JustVoxel Base parent ID, got '${ID:-}'." >&2
    exit 1
}
[[ "${VARIANT_ID:-}" == "justvoxel-vm" ]] || {
    echo "ERROR: expected VM-ready JustVoxel Base parent, got VARIANT_ID='${VARIANT_ID:-}'." >&2
    exit 1
}
[[ -r /usr/lib/justvoxel/variant ]]
[[ "$(cat /usr/lib/justvoxel/variant)" == "justvoxel-vm" ]] || {
    echo "ERROR: JustVoxel Base runtime variant marker is not justvoxel-vm." >&2
    exit 1
}
[[ "${VERSION_ID%%.*}" == "10" ]] || {
    echo "ERROR: expected JustVoxel major version 10, got '${VERSION_ID:-}'." >&2
    exit 1
}

# Preserve the Base and VM trust inherited from the parent and add HWE trust.
# install-image-trust.sh rewrites the sigstore registry mapping from its inputs,
# so pass all three repositories explicitly.
 /ctx/build_files/install-image-trust.sh     "${JUSTVOXEL_BASE_REPOSITORY}"     "${JUSTVOXEL_VM_REPOSITORY}"     "${JUSTVOXEL_HWE_REPOSITORY}"

OS_RELEASE_FILES=("${OS_RELEASE_USR}")
if [[ -e "${OS_RELEASE_ETC}" ]] && ! [[ "${OS_RELEASE_ETC}" -ef "${OS_RELEASE_USR}" ]]; then
    OS_RELEASE_FILES+=("${OS_RELEASE_ETC}")
fi

osr_set() {
    local key="$1" value="$2" file
    for file in "${OS_RELEASE_FILES[@]}"; do
        sed -i "/^${key}=/d" "${file}"
        printf '%s="%s"\n' "${key}" "${value}" >> "${file}"
    done
}

osr_set PRETTY_NAME "JustVoxel HWE 10"
osr_set VARIANT "JustVoxel HWE"
osr_set VARIANT_ID "justvoxel-hwe"
osr_set IMAGE_ID "justvoxel-hwe"

printf '%s\n' "justvoxel-hwe" > /usr/lib/justvoxel/variant
chmod 0644 /usr/lib/justvoxel/variant

# External repositories are build inputs only. Keep the immutable product from
# drifting through ad-hoc package installation after composition.
for repo_file in     /etc/yum.repos.d/epel*.repo     /etc/yum.repos.d/tailscale.repo     /etc/yum.repos.d/netbird.repo; do
    [[ -e "${repo_file}" ]] || continue
    sed -Ei 's/^[[:space:]]*enabled[[:space:]]*=[[:space:]]*1[[:space:]]*$/enabled=0/' "${repo_file}"
done

if dnf repolist --enabled | grep -Eiq 'epel|tailscale|netbird'; then
    echo "ERROR: an external package repository remains enabled in the HWE image." >&2
    dnf repolist --enabled
    exit 1
fi

dnf clean all
rm -rf /var/cache/* /var/log/* /var/tmp/* /var/lib/dnf /var/lib/rpm-state
bootc container lint

rm -rf /var
install -d -m0755 /var
install -d -m1777 /var/tmp

test "$(stat -c '%a %U %G' /var/tmp)" = "1777 root root"
jq empty /etc/containers/policy.json
test -f /usr/lib/pki/containers/home-server-project.pub
test -f /etc/containers/registries.d/ghcr.io-home-server-project.yaml
for trust_repository in     "${JUSTVOXEL_BASE_REPOSITORY}"     "${JUSTVOXEL_VM_REPOSITORY}"     "${JUSTVOXEL_HWE_REPOSITORY}"; do
    grep -Fq "${trust_repository}:" /etc/containers/registries.d/ghcr.io-home-server-project.yaml
done
grep -Fq "use-sigstore-attachments: true" /etc/containers/registries.d/ghcr.io-home-server-project.yaml
