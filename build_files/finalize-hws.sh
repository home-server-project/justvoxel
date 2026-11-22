#!/usr/bin/bash
set -ouex pipefail

: "${JUSTVOXEL_HWS_REPOSITORY:?JUSTVOXEL_HWS_REPOSITORY must be set}"

# External repositories are build inputs only. Keep the immutable product from
# drifting through ad-hoc package installation after composition.
for repo_file in \
    /etc/yum.repos.d/epel*.repo \
    /etc/yum.repos.d/tailscale.repo \
    /etc/yum.repos.d/netbird.repo; do
    [[ -e "${repo_file}" ]] || continue
    sed -Ei 's/^[[:space:]]*enabled[[:space:]]*=[[:space:]]*1[[:space:]]*$/enabled=0/' "${repo_file}"
done

# This is an execution guard for the repository transformation above, not a
# completed-image health validation. Run it before DNF state is removed.
if dnf repolist --enabled | grep -Eiq 'epel|tailscale|netbird'; then
    echo "ERROR: an external package repository remains enabled in the HWS image." >&2
    dnf repolist --enabled
    exit 1
fi

IMAGE_REPOSITORY="${JUSTVOXEL_HWS_REPOSITORY}" \
IMAGE_PRETTY_NAME="JustVoxel HWS 10" \
IMAGE_VARIANT="JustVoxel HWS" \
IMAGE_VARIANT_ID="justvoxel-hws" \
exec /ctx/build_files/finalize-image.sh
