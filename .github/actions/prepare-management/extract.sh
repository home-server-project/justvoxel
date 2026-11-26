#!/usr/bin/env bash
set -euo pipefail
package_ref="${MANAGEMENT_PACKAGE%:*}@${DIGEST}"
sudo podman pull --authfile "${HOME}/.docker/config.json" "${package_ref}"
container="$(sudo podman create "${package_ref}" /unused)"
stage="$(mktemp -d)"
trap 'sudo podman rm "${container}" >/dev/null; sudo rm -rf -- "${stage}"' EXIT
mkdir -p build_artifacts/management
sudo podman cp "${container}:/artifacts/management-agent" "${stage}/management-agent"
test -f "${stage}/management-agent"
test -x "${stage}/management-agent"
sudo install -m0755 "${stage}/management-agent" build_artifacts/management/management-agent
test -x build_artifacts/management/management-agent
if [[ "${MANAGEMENT_BUILD_MODE:-production}" == both ]]; then
    sudo podman cp "${container}:/artifacts/management-agent-development" "${stage}/management-agent-development"
    test -f "${stage}/management-agent-development"
    test -x "${stage}/management-agent-development"
    sudo install -m0755 "${stage}/management-agent-development" build_artifacts/management/management-agent-development
fi
