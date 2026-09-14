#!/usr/bin/bash
set -euo pipefail

if [[ "$#" -lt 1 ]]; then
    echo "Usage: install-image-trust.sh ghcr.io/owner/image [ghcr.io/owner/image ...]"
    exit 1
fi

POLICY="/etc/containers/policy.json"
KEY="/usr/lib/pki/containers/home-server-project.pub"
SIGSTORE_REGISTRIES="/etc/containers/registries.d/ghcr.io-home-server-project.yaml"

[[ -f /ctx/cosign.pub ]] || { echo "ERROR: /ctx/cosign.pub is missing."; exit 1; }
[[ -f "${POLICY}" ]] || { echo "ERROR: ${POLICY} is missing."; exit 1; }
command -v jq >/dev/null 2>&1

install -Dm0644 /ctx/cosign.pub "${KEY}"

working_policy="$(mktemp)"
next_policy="$(mktemp)"
trap 'rm -f "${working_policy}" "${next_policy}"' EXIT
cp "${POLICY}" "${working_policy}"

for image_repo in "$@"; do
    jq \
        --arg repo "${image_repo}" \
        --arg key "${KEY}" \
        '.transports |= (. // {}) |
         .transports.docker |= (. // {}) |
         .transports.docker[$repo] = [
           {
             "type": "sigstoreSigned",
             "keyPath": $key,
             "signedIdentity": {"type": "matchRepository"}
           }
         ]' \
        "${working_policy}" > "${next_policy}"
    jq empty "${next_policy}"
    mv "${next_policy}" "${working_policy}"
    next_policy="$(mktemp)"
done

install -m0644 "${working_policy}" "${POLICY}"
install -d -m0755 "$(dirname "${SIGSTORE_REGISTRIES}")"
{
    echo "docker:"
    for image_repo in "$@"; do
        printf '    %s:\n' "${image_repo}"
        echo "        use-sigstore-attachments: true"
    done
} > "${SIGSTORE_REGISTRIES}"
chmod 0644 "${SIGSTORE_REGISTRIES}"
