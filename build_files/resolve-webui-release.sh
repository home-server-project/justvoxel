#!/usr/bin/bash
set -euo pipefail

readonly repo='home-server-project/justvoxel-webui'
readonly api='https://api.github.com'
readonly required_api='v1'

usage() {
    echo 'Usage:' >&2
    echo '  resolve-webui-release.sh select <stable|testing>' >&2
    echo '  resolve-webui-release.sh fetch <semantic-version> <output-directory>' >&2
    exit 2
}

api_get() {
    local url="$1"
    local args=(--fail --silent --show-error --location --retry 3 --header 'Accept: application/vnd.github+json')
    if [[ -n ${GITHUB_TOKEN:-} ]]; then
        args+=(--header "Authorization: Bearer ${GITHUB_TOKEN}")
    fi
    curl "${args[@]}" "${url}"
}

all_releases() {
    local page page_json count combined='[]'
    for page in $(seq 1 20); do
        page_json="$(api_get "${api}/repos/${repo}/releases?per_page=100&page=${page}")"
        count="$(jq 'length' <<<"${page_json}")"
        [[ ${count} =~ ^[0-9]+$ ]] || {
            echo 'ERROR: GitHub Releases API returned unexpected data.' >&2
            exit 1
        }
        combined="$(jq -cn --argjson a "${combined}" --argjson b "${page_json}" '$a + $b')"
        (( count < 100 )) && break
        if (( page == 20 )); then
            echo 'ERROR: WebUI release history exceeds resolver pagination safety limit.' >&2
            exit 1
        fi
    done
    printf '%s' "${combined}"
}

select_release() {
    local channel="$1" releases releases_file
    [[ ${channel} == stable || ${channel} == testing ]] || usage
    if [[ -n ${JV_WEBUI_RELEASES_FILE:-} ]]; then
        releases="$(cat "${JV_WEBUI_RELEASES_FILE}")"
    else
        releases="$(all_releases)"
    fi
    releases_file="$(mktemp)"
    printf '%s' "${releases}" > "${releases_file}"
    python3 - "${channel}" "${releases_file}" <<'PY'
import json
import re
import sys

channel = sys.argv[1]
with open(sys.argv[2], encoding='utf-8') as handle:
    releases = json.load(handle)

pattern = re.compile(r'^v(\d+)\.(\d+)\.(\d+)(?:-(beta|rc)\.(\d+))?$')
candidates = []
for release in releases:
    if release.get('draft'):
        continue
    tag = release.get('tag_name', '')
    match = pattern.fullmatch(tag)
    if not match:
        continue
    major, minor, patch = map(int, match.group(1, 2, 3))
    stage = match.group(4)
    serial = int(match.group(5) or 0)
    if channel == 'stable' and (stage is not None or release.get('prerelease')):
        continue
    rank = 2 if stage is None else (1 if stage == 'rc' else 0)
    candidates.append(((major, minor, patch, rank, serial), tag[1:]))

if not candidates:
    print(f'ERROR: no usable {channel} JustVoxel WebUI release found', file=sys.stderr)
    sys.exit(1)

candidates.sort(key=lambda item: item[0])
print(candidates[-1][1])
PY
    rm -f "${releases_file}"
}

fetch_release() {
    local version="$1" out="$2" tag release archive asset_url expected_sha actual_sha tmp version_output
    [[ ${version} =~ ^[0-9]+\.[0-9]+\.[0-9]+(-(beta|rc)\.[0-9]+)?$ ]] || {
        echo "ERROR: invalid WebUI semantic version: ${version}" >&2
        exit 1
    }
    tag="v${version}"
    release="$(api_get "${api}/repos/${repo}/releases/tags/${tag}")"
    if [[ $(jq -r '.draft' <<<"${release}") != false ]]; then
        echo "ERROR: WebUI release ${tag} is a draft." >&2
        exit 1
    fi
    if [[ $(jq -r '.tag_name' <<<"${release}") != "${tag}" ]]; then
        echo 'ERROR: WebUI release tag mismatch.' >&2
        exit 1
    fi

    archive="justvoxel-webui-${tag}-linux-amd64.tar.gz"
    rm -rf "${out}"
    mkdir -p "${out}"
    for asset in "${archive}" SHA256SUMS SHA256SUMS.sig; do
        asset_url="$(jq -r --arg name "${asset}" '.assets[]? | select(.name == $name) | .browser_download_url' <<<"${release}" | head -n1)"
        if [[ -z ${asset_url} || ${asset_url} == null ]]; then
            echo "ERROR: required WebUI release asset missing: ${asset}" >&2
            exit 1
        fi
        api_get "${asset_url}" > "${out}/${asset}"
    done

    cosign verify-blob \
        --new-bundle-format=false \
        --key cosign.pub \
        --signature "${out}/SHA256SUMS.sig" \
        "${out}/SHA256SUMS" >/dev/null

    expected_sha="$(awk -v file="${archive}" '$2 == file {print $1}' "${out}/SHA256SUMS")"
    [[ ${expected_sha} =~ ^[0-9a-f]{64}$ ]] || {
        echo 'ERROR: signed SHA256SUMS does not contain the expected WebUI archive.' >&2
        exit 1
    }
    actual_sha="$(sha256sum "${out}/${archive}" | awk '{print $1}')"
    [[ ${actual_sha} == "${expected_sha}" ]] || {
        echo 'ERROR: WebUI artifact checksum mismatch.' >&2
        exit 1
    }

    tmp="$(mktemp -d)"
    mapfile -t members < <(tar -tzf "${out}/${archive}" | sed 's#^\./##' | sort)
    if [[ ${#members[@]} -ne 2 || ${members[0]} != justvoxel-webui || ${members[1]} != release.json ]]; then
        echo 'ERROR: WebUI release archive contains unexpected files.' >&2
        rm -rf "${tmp}"
        exit 1
    fi
    tar -xzf "${out}/${archive}" -C "${tmp}"
    chmod 0755 "${tmp}/justvoxel-webui"

    jq -e --arg version "${version}" --arg api "${required_api}" \
        '.version == $version and .management_api == $api and (.source_commit | test("^[0-9a-f]{40}$")) and (.build_date | type == "string")' \
        "${tmp}/release.json" >/dev/null

    version_output="$("${tmp}/justvoxel-webui" -version)"
    grep -Fqx "JustVoxel WebUI ${version}" <<<"${version_output}"
    grep -Fqx "source=$(jq -r '.source_commit' "${tmp}/release.json")" <<<"${version_output}"
    grep -Fqx "management-api=${required_api}" <<<"${version_output}"

    install -m0755 "${tmp}/justvoxel-webui" "${out}/justvoxel-webui"
    jq --arg sha256 "${actual_sha}" --arg tag "${tag}" \
        '. + {artifact_sha256:$sha256, release_tag:$tag}' \
        "${tmp}/release.json" > "${out}/webui-release.json"
    rm -rf "${tmp}"
    rm -f "${out}/${archive}" "${out}/SHA256SUMS" "${out}/SHA256SUMS.sig"
    printf 'Verified JustVoxel WebUI %s (%s)\n' "${version}" "${actual_sha}"
}

case "${1:-}" in
    select)
        [[ $# -eq 2 ]] || usage
        select_release "$2"
        ;;
    fetch)
        [[ $# -eq 3 ]] || usage
        fetch_release "$2" "$3"
        ;;
    *)
        usage
        ;;
esac
