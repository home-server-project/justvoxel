#!/usr/bin/bash

# Call with the existing restore-exclusion flock held until all writes finish.
# The journal also fences unattended shell work between Agent exit and recovery.
jv_switch_matrix_reservation_clear() (
    set -o pipefail
    local directory=/var/lib/justvoxel/management/logs/operations
    local journal
    [[ ! -L ${directory} ]] || return 1
    [[ -e ${directory} ]] || return 0
    [[ -d ${directory} && -r ${directory} && -x ${directory} ]] || return 1
    shopt -s nullglob
    local -a journals=("${directory}"/*.json)
    (( ${#journals[@]} )) || return 0
    for journal in "${journals[@]}"; do
        [[ -f ${journal} && ! -L ${journal} ]] || return 1
    done
    # Read in one jq process, without putting an unbounded journal list on argv.
    for journal in "${journals[@]}"; do
        cat -- "${journal}" || return 1
        printf '\n'
    done | jq -en 'all(inputs; type == "object" and (.operation_type | type == "string") and
        (.operation_type != "switch_matrix" or
            (.state == "succeeded" or .state == "resolved" or .state == "rolled_back")))' >/dev/null
)
