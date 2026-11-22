#!/usr/bin/bash

jv_bootc_status_json() {
    bootc status --json --format-version=1
}

jv_bootc_validate_status_json() {
    jq -e '.apiVersion == "org.containers.bootc/v1" and (.status | type == "object")' >/dev/null <<< "$1"
}

jv_bootc_entry_ref() {
    local json="$1" slot="$2"
    jq -r --arg slot "${slot}" '.status[$slot].image.image.image // empty' <<< "${json}"
}

jv_bootc_entry_version() {
    local json="$1" slot="$2"
    jq -r --arg slot "${slot}" '.status[$slot].image.version // empty' <<< "${json}"
}

jv_bootc_entry_digest() {
    local json="$1" slot="$2"
    jq -r --arg slot "${slot}" '.status[$slot].image.imageDigest // empty' <<< "${json}"
}

jv_bootc_cached_ref() {
    jq -r '.status.booted.cachedUpdate.image.image // empty' <<< "$1"
}

jv_bootc_cached_version() {
    jq -r '.status.booted.cachedUpdate.version // empty' <<< "$1"
}

jv_bootc_cached_digest() {
    jq -r '.status.booted.cachedUpdate.imageDigest // empty' <<< "$1"
}

jv_bootc_read_only() {
    jq -r '.status.readOnly // false' <<< "$1"
}

jv_bootc_staged_exists() {
    [[ -n $(jv_bootc_entry_digest "$1" staged) ]]
}

jv_bootc_rollback_exists() {
    [[ -n $(jv_bootc_entry_digest "$1" rollback) ]]
}

jv_bootc_short_state() {
    local json="$1"
    if ! jv_bootc_validate_status_json "${json}"; then
        printf 'Unknown'
    elif jv_bootc_staged_exists "${json}"; then
        printf 'Update staged - reboot to use it'
    elif [[ -n $(jv_bootc_entry_digest "${json}" booted) ]]; then
        printf 'Running'
    else
        printf 'Unknown'
    fi
}

jv_bootc_print_entry() {
    local json="$1" slot="$2" title="$3" ref version digest
    ref="$(jv_bootc_entry_ref "${json}" "${slot}")"
    version="$(jv_bootc_entry_version "${json}" "${slot}")"
    digest="$(jv_bootc_entry_digest "${json}" "${slot}")"

    echo "${title}"
    if [[ -z ${digest} ]]; then
        echo '  None'
        return 0
    fi
    [[ -n ${ref} ]] && echo "  Image:   ${ref}"
    [[ -n ${version} ]] && echo "  Version: ${version}"
    echo "  Digest:  ${digest}"
}
