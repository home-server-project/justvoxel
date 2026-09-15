#!/usr/bin/bash
jv_migration_select_import_source() {
    local have_config="$1" choice path
    local -a options=('Local file / directory or already-mounted filesystem' 'Attached disk / partition' 'USB / removable storage')
    [[ ${have_config} == yes ]] && options+=('Configured JustVoxel backup storage')
    options+=('Temporary NFS share' 'Temporary SMB/CIFS share' 'Back')
    choice="$(jui_choose 'Import source' "${options[@]}")" || return 2
    case "${choice}" in
        'Local file / directory or already-mounted filesystem')
            path="$(jui_input 'Migration archive or server directory path')" || return 2
            [[ -n ${path} ]] || return 2
            JV_MIGRATION_SOURCE="$(realpath -m -- "${path}")"
            [[ -e ${JV_MIGRATION_SOURCE} ]] || { echo "ERROR: source does not exist: ${JV_MIGRATION_SOURCE}" >&2; return 1; }
            ;;
        'Attached disk / partition') jv_migration_choose_block_source no ;;
        'USB / removable storage') jv_migration_choose_block_source yes ;;
        'Configured JustVoxel backup storage')
            jv_backup_require_read_target || return 1
            jv_migration_choose_source_under "${minecraft_backup_path}" 'Select from configured backup storage'
            ;;
        'Temporary NFS share')
            jv_migration_mount_nfs import || return $?
            jv_migration_choose_source_under "${JV_MIGRATION_MEDIA_MOUNT}" 'Select from temporary NFS share'
            ;;
        'Temporary SMB/CIFS share')
            jv_migration_mount_smb import || return $?
            jv_migration_choose_source_under "${JV_MIGRATION_MEDIA_MOUNT}" 'Select from temporary SMB share'
            ;;
        'Back') return 2 ;;
        *) return 2 ;;
    esac
}

jv_migration_ensure_export_dir() {
    local dir="$1"
    if [[ -e ${dir} && ! -d ${dir} ]]; then
        echo "ERROR: export destination parent is not a directory: ${dir}" >&2
        return 1
    fi
    if [[ ! -d ${dir} ]]; then
        install -d -m0700 -o root -g root "${dir}" || return 1
    fi
    [[ -w ${dir} ]] || {
        echo "ERROR: export destination is not writable: ${dir}" >&2
        return 1
    }
}

jv_migration_resolve_export_path() {
    local base="$1" filename="$2"
    if [[ ${base} == *.tar.gz ]]; then
        JV_MIGRATION_DESTINATION="$(realpath -m -- "${base}")"
    else
        JV_MIGRATION_DESTINATION="$(realpath -m -- "${base}/${filename}")"
    fi
}

jv_migration_prepare_export_destination() {
    local have_config="$1" filename="$2" choice path device fstype
    local -a options=('Local directory / already-mounted filesystem' 'Attached disk / partition' 'USB / removable storage')
    [[ ${have_config} == yes ]] && options+=('Configured JustVoxel backup storage')
    options+=('Temporary NFS share' 'Temporary SMB/CIFS share' 'Back')
    choice="$(jui_choose 'Export destination' "${options[@]}")" || return 2
    case "${choice}" in
        'Local directory / already-mounted filesystem')
            path="$(jui_input 'Destination directory or .tar.gz filename' '/var/lib/justvoxel/exports')" || return 2
            [[ -n ${path} ]] || return 2
            if [[ ${path} == *.tar.gz ]]; then
                jv_migration_ensure_export_dir "$(dirname -- "${path}")" || return 1
            else
                jv_migration_ensure_export_dir "${path}" || return 1
            fi
            jv_migration_resolve_export_path "${path}" "${filename}"
            ;;
        'Attached disk / partition'|'USB / removable storage')
            echo 'Detected block storage:'
            jv_migration_show_block_devices
            device="$(jui_input 'Device or partition to mount temporarily')" || return 2
            [[ -n ${device} ]] || return 2
            if [[ ${choice} == 'USB / removable storage' ]]; then
                jv_migration_mount_device "${device}" export yes || return 1
            else
                jv_migration_mount_device "${device}" export no || return 1
            fi
            fstype="$(findmnt -n -o FSTYPE --target "${JV_MIGRATION_MEDIA_MOUNT}" 2>/dev/null || true)"
            JV_MIGRATION_EXPORT_FSTYPE="${fstype}"
            jv_migration_resolve_export_path "${JV_MIGRATION_MEDIA_MOUNT}" "${filename}"
            ;;
        'Configured JustVoxel backup storage')
            jv_backup_prepare_write_target || return 1
            jv_migration_resolve_export_path "${minecraft_backup_path}" "${filename}"
            ;;
        'Temporary NFS share')
            jv_migration_mount_nfs export || return $?
            jv_migration_resolve_export_path "${JV_MIGRATION_MEDIA_MOUNT}" "${filename}"
            ;;
        'Temporary SMB/CIFS share')
            jv_migration_mount_smb export || return $?
            jv_migration_resolve_export_path "${JV_MIGRATION_MEDIA_MOUNT}" "${filename}"
            ;;
        'Back') return 2 ;;
        *) return 2 ;;
    esac
    [[ ! -e ${JV_MIGRATION_DESTINATION} && ! -e ${JV_MIGRATION_DESTINATION}.partial ]] || {
        echo "ERROR: export destination already exists: ${JV_MIGRATION_DESTINATION}" >&2
        return 1
    }
    [[ -w $(dirname -- "${JV_MIGRATION_DESTINATION}") ]] || {
        echo "ERROR: export destination is not writable: $(dirname -- "${JV_MIGRATION_DESTINATION}")" >&2
        return 1
    }
}
