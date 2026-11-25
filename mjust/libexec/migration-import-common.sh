#!/usr/bin/bash
# Sourced by migration-import: selected-instance transaction helpers.
jv_migration_api_result() {
    local outcome="$1" phase="$2" recovery_transaction="$3" status="$4" result_file="${JV_MIGRATION_API_RESULT_FILE:-}" tmp
    [[ ${JV_MIGRATION_API_MODE:-0} == 1 && -n ${result_file} ]] || return 0
    case "${result_file}" in "${JV_MIGRATION_RUN_ROOT}/"*) ;; *) echo 'ERROR: invalid Management API Import result path.' >&2; return 1 ;; esac
    install -d -m0700 -o root -g root "${JV_MIGRATION_RUN_ROOT}"
    tmp="$(jv_migration_private_temp "${result_file}.tmp.XXXXXX")" || return 1
    jq -cn --arg outcome "${outcome}" --arg phase "${phase}" --arg transaction "${recovery_transaction}" --arg status "${status}"         '{outcome:$outcome,phase:$phase,transaction:$transaction,status:$status}' > "${tmp}" || return 1
    jv_migration_publish_private "${tmp}" "${result_file}"
}

# Validate the private rollback record and its historical destination snapshot.
jv_migration_validate_rolled_back_import_evidence() {
    local retained="$1" path
    [[ ${retained##*/} == .justvoxel-import-?* ]] || return 1
    [[ -d ${retained} && ! -L ${retained} && $(stat -c '%u:%g:%a' -- "${retained}") == 0:0:700 ]] || return 1
    for path in "${retained}/state" "${retained}/evidence.json"; do
        [[ -f ${path} && ! -L ${path} && $(stat -c '%u:%g:%a:%h' -- "${path}") == 0:0:600:1 ]] || return 1
    done
    [[ $(sed -n 's/^phase=//p' "${retained}/state") == rolled-back ]] || return 1
    [[ -d ${retained}/destination-state && ! -L ${retained}/destination-state &&
        $(stat -c '%u:%g:%a' -- "${retained}/destination-state") == 0:0:700 ]] || return 1
    jv_migration_validate_evidence "${retained}" || return 1
    [[ ! -e ${retained}/pre-import && ! -L ${retained}/pre-import ]]
}

# After a fresh rollback, verify that the live destination was restored.
jv_migration_check_fresh_rollback() {
    local retained="$1" path running state data_identity
    jv_migration_validate_rolled_back_import_evidence "${retained}" || return 1
    [[ -d ${DATA_PATH} && ! -L ${DATA_PATH} ]] || return 1
    jv_migration_validate_data || return 1
    data_identity="$(stat -Lc '%d:%i' -- "${DATA_PATH}")" || return 1
    [[ ${data_identity} == "$(jq -r '.data_identity' "${retained}/evidence.json")" ]] || return 1
    while IFS= read -r path; do
        [[ -f ${path} && ! -L ${path} ]] || return 1
        cmp -s -- "${path}" "${retained}/destination-state/${path##*/}" || return 1
    done < <(jv_migration_runtime_paths)
    running="$(jq -r '.was_running' "${retained}/evidence.json")" || return 1
    state="$(systemctl show --property=ActiveState --value "${JV_INSTANCE_UNIT}")" || return 1
    [[ ${running} == true && ${state} == active || ${running} == false && ${state} == inactive ]]
}

# Call only after verifying a fresh rollback or a retained terminal record,
# with maintenance held.
jv_migration_finalize_rolled_back_import() {
    local retained="$1"
    # Keep the recovery evidence intact while removing the staged and failed data.
    if rm -rf --one-file-system --preserve-root=all -- \
        "${retained}/staging-source" "${retained}/staged-server" "${retained}/failed-import" &&
        jv_migration_clear_recovery "${retained}" &&
        rm -rf --one-file-system --preserve-root=all -- "${retained}"; then
        return 0
    fi
    if [[ -d ${retained} ]] && ! jv_migration_register_recovery "${retained}"; then
        echo "ERROR: could not restore the recovery registry entry for ${retained}" >&2
    fi
    return 1
}

jv_migration_check_earlier_imports() {
    local parent="$1" retained
    # Open the maintenance lock only when there is retained state to inspect.
    for retained in "${parent}"/.justvoxel-import-*; do
        [[ -e ${retained} || -L ${retained} ]] || continue
        if ! jv_migration_finalize_earlier_import "${retained}" 9>"${JV_MAINTENANCE_LOCK}"; then
            echo 'ERROR: an earlier JustVoxel import transaction still exists.' >&2
            echo "Recovery state: ${retained}" >&2
            echo 'Use the guided recovery flow before starting another import:' >&2
            echo '  mjust migration-recover' >&2
            return 1
        fi
    done
}

# A durable terminal record proves Recovery already validated the rollback.
# The retained snapshot is authoritative only for the historical transaction.
jv_migration_validate_retained_rollback() {
    local retained="$1" path symlink
    jv_migration_validate_rolled_back_import_evidence "${retained}" || return 1
    # Require the recovery writer's fields in order, with an optional timestamp.
    awk -v id="${JV_INSTANCE_ID}" '
        NR == 1 && $0 == "instance_id=" id { next }
        NR == 2 && $0 == "phase=rolled-back" { next }
        NR == 3 && $0 == "source=recovery" { next }
        NR == 4 && $0 == "source_type=recovery" { next }
        NR == 5 &&
            /^updated_at=[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]Z$/ { next }
        { invalid = 1 }
        END { exit (invalid || NR < 4) }
    ' "${retained}/state" || return 1
    jq -se 'length == 1 and (.[0].data_identity | test("^[0-9]+:[0-9]+$"))' \
        "${retained}/evidence.json" >/dev/null || return 1
    for path in "${retained}/staging-source" "${retained}/staged-server" "${retained}/failed-import"; do
        [[ ! -e ${path} || -d ${path} ]] || return 1
    done
    # Staged server payloads can contain supported links. Only inspect the
    # transaction and snapshot entries, which must never be symlinks.
    symlink="$(find "${retained}" "${retained}/destination-state" -maxdepth 1 -type l -print -quit)" || return 1
    [[ -z ${symlink} ]]
}

jv_migration_finalize_earlier_import() {
    local retained="$1"
    flock -n 9 || return 1
    jv_migration_validate_retained_rollback "${retained}" || return 1
    jv_migration_finalize_rolled_back_import "${retained}"
}

rollback_import() {
    local cleanup_status='The selected Minecraft destination was restored and validated, but rollback transaction cleanup did not complete. Use Migration Recovery to finish cleanup.'
    rollback_attempted=yes
    echo 'Import did not validate. Restoring the selected destination instance.' >&2
    if jv_migration_write_state "${transaction}" rollback "${JV_MIGRATION_SOURCE:-unknown}" "${source_class}" &&
        jv_migration_recover_selected "${transaction}" && jv_migration_check_fresh_rollback "${transaction}"; then
        live_modified=no
        if ! jv_migration_finalize_rolled_back_import "${transaction}"; then
            echo "WARNING: ${cleanup_status}" >&2
            echo "Recovery state: ${transaction}" >&2
            jv_migration_api_result needs_attention rolled-back "${transaction}" "${cleanup_status}" ||
                echo 'ERROR: could not publish the rollback cleanup result.' >&2
            return 1
        fi
        transaction=''
        jv_migration_api_result rolled_back rolled-back '' 'Import failed; the selected destination instance was restored and validated.'
        return $?
    fi
    [[ ${jv_migration_handoff_pending} == no ]] || return 1
    jv_migration_write_state "${transaction}" critical-rollback "${JV_MIGRATION_SOURCE:-unknown}" "${source_class}" || true
    jv_migration_register_recovery "${transaction}" || true
    jv_migration_api_result needs_attention critical-rollback "${transaction}" 'Import rollback requires recovery of the selected destination instance.' || true
    return 1
}

on_exit() {
    local rc=$? preactivation_outcome=rolled_back
    local preactivation_status='Server migration Import stopped before live activation; no Minecraft data was changed.'
    trap - EXIT INT TERM
    [[ ${jv_migration_handoff_pending} == no ]] || exit 1
    if (( rc != 0 )); then
        if [[ ${live_modified} == yes && ${validated} != yes && -n ${transaction:-} && -d ${transaction} ]]; then
            rollback_import || true
        elif [[ ${minecraft_stopped_by_import} == yes && ${live_modified} != yes && ${minecraft_was_active} == yes ]]; then
            echo 'Import failed before live data changed; restarting the previously running Minecraft server.' >&2
            if jv_migration_start_with_maintenance_handoff >/dev/null 2>&1; then
                minecraft_stopped_by_import=no
            else
                [[ ${jv_migration_handoff_pending} == no ]] || exit 1
                jv_migration_register_recovery "${transaction}" || true
                preactivation_outcome=needs_attention
                preactivation_status='Server migration Import failed before activation, but the previously running Minecraft server could not be restarted automatically.'
            fi
        fi
    fi
    [[ ${jv_migration_handoff_pending} == no ]] || exit 1
    # Staging created before live activation is disposable. Preserve transaction
    # evidence only after a rollback attempt or a critical/live-state failure.
    if [[ ${preactivation_outcome} != needs_attention && ${live_modified} != yes && ${rollback_attempted} != yes && -n ${transaction:-} && -d ${transaction} ]]; then
        rm -rf -- "${transaction}" || true
        transaction=''
    fi
    if [[ ${transport_started} == yes ]]; then
        jv_migration_transport_cleanup || true
    fi
    if (( rc != 0 )) && [[ ${live_modified} != yes && ${rollback_attempted} != yes ]]; then
        jv_migration_api_result "${preactivation_outcome}" pre-activation '' "${preactivation_status}" || true
    fi
    exit "${rc}"
}
trap on_exit EXIT INT TERM

select_candidate() {
    local detection="$1" base="$2" count selection index candidate label
    local -a labels=() candidates=()
    count="$(jq -r '.candidates | length' <<< "${detection}")"
    (( count > 0 )) || { echo 'ERROR: no recognizable Minecraft server root was found.' >&2; return 1; }
    if [[ ${JV_MIGRATION_API_MODE:-0} == 1 ]]; then
        local requested="${JV_MIGRATION_API_SELECTED_ROOT_REL:-}"
        if (( count == 1 )) && [[ -z ${requested} ]]; then
            jq -c '.candidates[0]' <<< "${detection}"
            return 0
        fi
        [[ -n ${requested} ]] || { echo 'ERROR: multiple Minecraft roots were detected and no reviewed root was supplied.' >&2; return 1; }
        for (( index=0; index<count; index++ )); do
            candidate="$(jq -c ".candidates[${index}]" <<< "${detection}")"
            root="$(jq -r '.root' <<< "${candidate}")"
            if [[ "${root#${base}/}" == "${requested}" ]]; then
                printf '%s' "${candidate}"
                return 0
            fi
        done
        echo 'ERROR: the reviewed Minecraft root no longer exists in staged source data.' >&2
        return 1
    fi
    if (( count == 1 )); then
        jq -c '.candidates[0]' <<< "${detection}"
        return 0
    fi
    echo 'Multiple possible Minecraft server roots were found. JustVoxel will not guess.'
    for (( index=0; index<count; index++ )); do
        candidate="$(jq -c ".candidates[${index}]" <<< "${detection}")"
        root="$(jq -r '.root' <<< "${candidate}")"
        type="$(jq -r '.sourceType' <<< "${candidate}")"
        version="$(jq -r '.minecraftVersion // "unknown"' <<< "${candidate}")"
        label="${root#${base}/}   [${type}, Minecraft ${version}]"
        labels+=("${label}")
        candidates+=("${candidate}")
    done
    labels+=('Back')
    selection="$(jui_choose 'Select Minecraft server root' "${labels[@]}")" || return 2
    [[ ${selection} == Back ]] && return 2
    for index in "${!labels[@]}"; do
        if [[ ${labels[index]} == "${selection}" && ${index} -lt ${#candidates[@]} ]]; then
            printf '%s' "${candidates[index]}"
            return 0
        fi
    done
    return 1
}

prompt_source_online_mode() {
    local answer
    if [[ ${JV_MIGRATION_API_MODE:-0} == 1 ]]; then
        [[ ${JV_MIGRATION_API_ONLINE_MODE_CONFIRMED:-no} == yes ]]
        return
    fi
    echo
    echo 'JustVoxel could not determine the source Java online-mode setting.'
    echo 'Player UUID identity must not change during migration.'
    echo 'Only continue if you have verified that the source server used online-mode=true.'
    printf 'Type ONLINE to confirm source online-mode=true: ' >/dev/tty
    IFS= read -r answer </dev/tty
    [[ ${answer} == ONLINE ]]
}
