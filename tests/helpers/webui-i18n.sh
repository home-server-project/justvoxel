#!/usr/bin/bash
# Test-only catalog lookup; optional surrounding source preserves structural contracts.
catalog_text_used() {
    local file=$1 text=$2 prefix=${3-\"} suffix=${4-\"}
    local catalog keys key source_text
    catalog="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/webui/internal/server/locales/en.json"
    keys=$(jq -er --arg text "$text" '[to_entries[] | select(.value == $text) | .key] | select(length > 0) | .[]' "$catalog") || return 1
    source_text=$(cat "$file") || return 1
    while IFS= read -r key; do
        [[ $source_text == *"${prefix}${key}${suffix}"* ]] && return 0
    done <<< "$keys"
    return 1
}
