#!/usr/bin/bash

jv_normalize_arch() {
    case "${1:-}" in
        x86_64|amd64) printf 'amd64' ;;
        aarch64|arm64) printf 'arm64' ;;
        armv7l|armv7|arm) printf 'arm' ;;
        ppc64le) printf 'ppc64le' ;;
        s390x) printf 's390x' ;;
        *) return 1 ;;
    esac
}

jv_manifest_platform_digest() {
    local raw="$1" os="$2" arch="$3"
    jq -r --arg os "${os}" --arg arch "${arch}" '
        .manifests[]?
        | select(.platform.os == $os and .platform.architecture == $arch)
        | .digest
    ' <<< "${raw}" | head -n1
}

jv_remote_platform_digest() {
    local image="$1" raw digest arch
    raw="$(skopeo inspect --raw "docker://${image}" 2>/dev/null)" || return 1

    if jq -e '.manifests and (.manifests | type == "array")' <<< "${raw}" >/dev/null 2>&1; then
        arch="$(jv_normalize_arch "$(uname -m)")" || return 1
        digest="$(jv_manifest_platform_digest "${raw}" linux "${arch}")"
    else
        digest="$(skopeo inspect --format '{{.Digest}}' "docker://${image}" 2>/dev/null)" || return 1
    fi

    [[ ${digest} =~ ^sha256:[0-9a-f]{64}$ ]] || return 1
    printf '%s' "${digest}"
}

jv_game_version_state() {
    local mode="$1" current="$2" latest="$3"
    if [[ ${mode} != pinned ]]; then
        printf 'moving'
    elif [[ -z ${latest} ]]; then
        printf 'unknown'
    elif [[ ${current} == "${latest}" ]]; then
        printf 'current'
    else
        printf 'update_available'
    fi
}
