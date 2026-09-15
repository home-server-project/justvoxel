#!/usr/bin/bash
set -ouex pipefail

: "${IMAGE_REPOSITORY:?IMAGE_REPOSITORY must be set}"
: "${IMAGE_PRETTY_NAME:?IMAGE_PRETTY_NAME must be set}"
: "${IMAGE_VARIANT:?IMAGE_VARIANT must be set}"
: "${IMAGE_VARIANT_ID:?IMAGE_VARIANT_ID must be set}"

/ctx/build_files/install-image-trust.sh "${IMAGE_REPOSITORY}"

OS_RELEASE_USR=/usr/lib/os-release
OS_RELEASE_ETC=/etc/os-release
[[ -r "${OS_RELEASE_USR}" ]]
# shellcheck disable=SC1090
source "${OS_RELEASE_USR}"
BASE_ID="${ID:-}"
BASE_PRETTY_NAME="${PRETTY_NAME:-}"
BASE_VERSION_ID="${VERSION_ID:-}"
BASE_PLATFORM_ID="${PLATFORM_ID:-}"
BASE_CPE_NAME="${CPE_NAME:-}"

[[ "${BASE_ID}" == "almalinux" ]]
[[ "${BASE_VERSION_ID%%.*}" == "10" ]]
[[ "${BASE_PLATFORM_ID}" == "platform:el10" ]]

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

osr_unset() {
    local key="$1" file
    for file in "${OS_RELEASE_FILES[@]}"; do
        sed -i "/^${key}=/d" "${file}"
    done
}

osr_set NAME "JustVoxel"
osr_set PRETTY_NAME "${IMAGE_PRETTY_NAME}"
osr_set ID "justvoxel"
osr_set ID_LIKE "almalinux rhel centos fedora"
osr_set VERSION "${BASE_VERSION_ID}"
osr_set VARIANT "${IMAGE_VARIANT}"
osr_set VARIANT_ID "${IMAGE_VARIANT_ID}"
osr_set IMAGE_ID "${IMAGE_VARIANT_ID}"
osr_set IMAGE_VERSION "10"
osr_set HOME_URL "https://github.com/home-server-project/justvoxel"
osr_set DOCUMENTATION_URL "https://github.com/home-server-project/justvoxel/tree/main/docs"
osr_set SUPPORT_URL "https://github.com/home-server-project/justvoxel/issues"
osr_set BUG_REPORT_URL "https://github.com/home-server-project/justvoxel/issues"
osr_set VENDOR_NAME "Home Server Project"
osr_set VENDOR_URL "https://github.com/home-server-project"
osr_set CPE_NAME "cpe:/o:home-server-project:justvoxel:10"
osr_set JUSTVOXEL_BASE_ID "${BASE_ID}"
osr_set JUSTVOXEL_BASE_PRETTY_NAME "${BASE_PRETTY_NAME}"
osr_set JUSTVOXEL_BASE_VERSION_ID "${BASE_VERSION_ID}"
osr_set JUSTVOXEL_BASE_PLATFORM_ID "${BASE_PLATFORM_ID}"
osr_set JUSTVOXEL_BASE_CPE_NAME "${BASE_CPE_NAME}"
osr_set JUSTVOXEL_BASE_PROFILE "almalinux-10-minimal-plus"

for key in ALMALINUX_MANTISBT_PROJECT ALMALINUX_MANTISBT_PROJECT_VERSION REDHAT_SUPPORT_PRODUCT REDHAT_SUPPORT_PRODUCT_VERSION SUPPORT_END LOGO; do
    osr_unset "${key}"
done
chmod 0644 "${OS_RELEASE_FILES[@]}"

install -d -m0755 /usr/lib/justvoxel
printf '%s\n' "${IMAGE_VARIANT_ID}" > /usr/lib/justvoxel/variant
chmod 0644 /usr/lib/justvoxel/variant

# External repositories are composition inputs only. Immutable hosts must not
# accidentally drift through ad-hoc package installation.
for repo_file in \
    /etc/yum.repos.d/epel*.repo \
    /etc/yum.repos.d/tailscale.repo \
    /etc/yum.repos.d/netbird.repo; do
    [[ -e "${repo_file}" ]] || continue
    sed -Ei 's/^[[:space:]]*enabled[[:space:]]*=[[:space:]]*1[[:space:]]*$/enabled=0/' "${repo_file}"
done

if dnf repolist --enabled | grep -Eiq 'epel|tailscale|netbird'; then
    echo "ERROR: an external package repository remains enabled in the final image."
    dnf repolist --enabled
    exit 1
fi

# /var is persistent machine state in bootc and image-provided /var content is
# only populated on the initial deployment. Required runtime directories must
# therefore be recreated declaratively (tmpfiles.d/StateDirectory), not relied
# upon as package payload in the image. Remove obvious build-only state first,
# then run an informational lint over the remaining package/runtime state before
# cleanup. The shipped image is still gated later by fatal bootc lint.
dnf clean all
rm -rf /var/cache/* /var/log/* /var/tmp/* /var/lib/dnf /var/lib/rpm-state
bootc container lint

# Keep only the minimal bootc /var skeleton in the immutable image. Runtime
# state is recreated by the declarative rules validated above.
rm -rf /var
install -d -m0755 /var
install -d -m1777 /var/tmp

test "$(stat -c '%a %U %G' /var/tmp)" = "1777 root root"
jq empty /etc/containers/policy.json
test -f /usr/lib/pki/containers/home-server-project.pub
test -f /etc/containers/registries.d/ghcr.io-home-server-project.yaml
grep -Fq "${IMAGE_REPOSITORY}:" /etc/containers/registries.d/ghcr.io-home-server-project.yaml
grep -Fq "use-sigstore-attachments: true" /etc/containers/registries.d/ghcr.io-home-server-project.yaml
