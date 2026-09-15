#!/usr/bin/bash

storage_prepare_backup_interactive() {
    local allow_back="${1:-no}" choice
    local -a options

    if storage_is_vm; then
        echo 'VM storage recommendation: use a second virtual disk for backups, or a network share.'
        options=(
            'Provision a dedicated second virtual disk (recommended)'
            'Use an existing filesystem/partition'
            'NFS network share'
            'SMB/CIFS network share'
            'Directory on the system filesystem'
        )
    else
        options=(
            'Provision a dedicated whole disk or USB drive'
            'Use an existing filesystem/partition'
            'Create a partition in already-unallocated disk space'
            'NFS network share'
            'SMB/CIFS network share'
            'Directory on the system filesystem'
        )
    fi
    [[ ${allow_back} == yes ]] && options+=('Back')

    choice="$(jui_choose 'Backup storage' "${options[@]}")" || return 2

    if storage_is_vm; then
        case "${choice}" in
            'Provision a dedicated second virtual disk (recommended)') storage_prepare_whole_disk backup ;;
            'Use an existing filesystem/partition') storage_prepare_existing_partition backup ;;
            'NFS network share') storage_prepare_nfs ;;
            'SMB/CIFS network share') storage_prepare_smb ;;
            'Directory on the system filesystem')
                STORAGE_TYPE=system
                STORAGE_MOUNT_POINT=''
                STORAGE_EXPECTED_UUID=''
                STORAGE_EXPECTED_SOURCE=''
                STORAGE_PATH="$(prompt_default 'Backup directory' '/var/lib/justvoxel/backups')"
                STORAGE_PATH="$(realpath -m -- "${STORAGE_PATH}")"
                validate_storage_path "${STORAGE_PATH}" || return 1
                ;;
            'Back') return 2 ;;
            *) return 2 ;;
        esac
    else
        case "${choice}" in
            'Provision a dedicated whole disk or USB drive') storage_prepare_whole_disk backup ;;
            'Use an existing filesystem/partition') storage_prepare_existing_partition backup ;;
            'Create a partition in already-unallocated disk space') storage_prepare_free_partition backup ;;
            'NFS network share') storage_prepare_nfs ;;
            'SMB/CIFS network share') storage_prepare_smb ;;
            'Directory on the system filesystem')
                STORAGE_TYPE=system
                STORAGE_MOUNT_POINT=''
                STORAGE_EXPECTED_UUID=''
                STORAGE_EXPECTED_SOURCE=''
                STORAGE_PATH="$(prompt_default 'Backup directory' '/var/lib/justvoxel/backups')"
                STORAGE_PATH="$(realpath -m -- "${STORAGE_PATH}")"
                validate_storage_path "${STORAGE_PATH}" || return 1
                ;;
            'Back') return 2 ;;
            *) return 2 ;;
        esac
    fi

    storage_apply_backup_globals
}

storage_prepare_data_target_interactive() {
    local choice
    choice="$(jui_choose 'Minecraft data target' \
        'Provision a dedicated whole disk' \
        'Use an existing filesystem/partition' \
        'Create a partition in already-unallocated disk space' \
        'Back')" || return 2
    case "${choice}" in
        'Provision a dedicated whole disk') storage_prepare_whole_disk data ;;
        'Use an existing filesystem/partition') storage_prepare_existing_partition data ;;
        'Create a partition in already-unallocated disk space') storage_prepare_free_partition data ;;
        'Back') return 2 ;;
        *) return 2 ;;
    esac
}
