#!/usr/bin/bash

JV_MIGRATION_TRANSPORT_ROOT=''
JV_MIGRATION_OWNED_MOUNT=''
JV_MIGRATION_MEDIA_MOUNT=''
JV_MIGRATION_SMB_CREDENTIALS=''
JV_MIGRATION_SOURCE=''
JV_MIGRATION_DESTINATION=''
JV_MIGRATION_MEDIA_ADMIN_MOUNT=no
JV_MIGRATION_MEDIA_REMOVABLE=no

jv_migration_transport_begin() {
    [[ -n ${JV_MIGRATION_TRANSPORT_ROOT} ]] && return 0
    install -d -m0700 -o root -g root "${JV_MIGRATION_RUN_ROOT}"
    JV_MIGRATION_TRANSPORT_ROOT="$(mktemp -d "${JV_MIGRATION_RUN_ROOT}/transport.XXXXXX")"
    chmod 0700 "${JV_MIGRATION_TRANSPORT_ROOT}"
}

jv_migration_transport_cleanup() {
    local rc=0 mount_still_active=no
    if [[ -n ${JV_MIGRATION_OWNED_MOUNT:-} ]] && mountpoint -q -- "${JV_MIGRATION_OWNED_MOUNT}"; then
        if ! umount -- "${JV_MIGRATION_OWNED_MOUNT}"; then
            echo "ERROR: temporary migration mount could not be unmounted: ${JV_MIGRATION_OWNED_MOUNT}" >&2
            echo 'Temporary mount state was retained; JustVoxel will not delete through a still-mounted filesystem.' >&2
            rc=1
            mount_still_active=yes
        fi
    fi
    if [[ -n ${JV_MIGRATION_SMB_CREDENTIALS:-} ]]; then
        rm -f -- "${JV_MIGRATION_SMB_CREDENTIALS}" || rc=1
    fi
    if [[ ${mount_still_active} == no && -n ${JV_MIGRATION_TRANSPORT_ROOT:-} && -d ${JV_MIGRATION_TRANSPORT_ROOT} ]]; then
        rm -rf -- "${JV_MIGRATION_TRANSPORT_ROOT}" || rc=1
    fi
    if [[ ${mount_still_active} == no ]]; then
        JV_MIGRATION_OWNED_MOUNT=''
        JV_MIGRATION_MEDIA_MOUNT=''
        JV_MIGRATION_TRANSPORT_ROOT=''
    fi
    JV_MIGRATION_SMB_CREDENTIALS=''
    return "${rc}"
}

jv_migration_supported_media_fs() {
    case "${1,,}" in
        ext4|xfs|btrfs|vfat|exfat) return 0 ;;
        ntfs|ntfs3)
            echo 'ERROR: NTFS removable media is not supported by JustVoxel migration v1.' >&2
            echo 'Use ext4, XFS, Btrfs, exFAT, FAT32/vfat, SMB/NFS, or copy the archive locally.' >&2
            return 1
            ;;
        *)
            echo "ERROR: unsupported removable-media filesystem: ${1:-unknown}" >&2
            echo 'Allowed migration media filesystems: ext4, XFS, Btrfs, exFAT, FAT32/vfat.' >&2
            return 1
            ;;
    esac
}

jv_migration_show_block_devices() {
    lsblk -e7 -o NAME,PATH,TYPE,SIZE,FSTYPE,LABEL,UUID,MOUNTPOINTS,MODEL,TRAN,HOTPLUG,RM,RO
}

jv_migration_parent_disk() {
    lsblk -s -npo NAME,TYPE "$1" 2>/dev/null | awk '$2 == "disk" {print $1; exit}'
}

jv_migration_validate_media_device() {
    local device="$1" removable_only="${2:-no}" type fstype parent tran rm hotplug
    [[ -b ${device} ]] || { echo "ERROR: not a block device: ${device}" >&2; return 1; }
    type="$(lsblk -dnro TYPE "${device}" 2>/dev/null || true)"
    [[ ${type} == part || ${type} == disk ]] || { echo 'ERROR: select a disk or filesystem partition.' >&2; return 1; }
    fstype="$(blkid -s TYPE -o value "${device}" 2>/dev/null || true)"
    [[ -n ${fstype} ]] || { echo 'ERROR: selected device has no recognized filesystem. Migration never formats media.' >&2; return 1; }
    jv_migration_supported_media_fs "${fstype}" || return 1

    parent="$(jv_migration_parent_disk "${device}")"
    [[ -n ${parent} ]] || { echo 'ERROR: could not identify the parent disk safely.' >&2; return 1; }
    if storage_disk_is_system "${parent}"; then
        echo "ERROR: refusing temporary migration mount from the JustVoxel system disk: ${parent}" >&2
        echo 'Use Local file / already-mounted filesystem for paths that intentionally live on system storage.' >&2
        return 1
    fi

    if [[ ${removable_only} == yes ]]; then
        tran="$(lsblk -dnro TRAN "${parent}" 2>/dev/null | xargs || true)"
        rm="$(lsblk -dnro RM "${parent}" 2>/dev/null || true)"
        hotplug="$(lsblk -dnro HOTPLUG "${parent}" 2>/dev/null || true)"
        if [[ ${tran} != usb && ${rm} != 1 && ${hotplug} != 1 ]]; then
            echo 'ERROR: selected device is not identified as USB/removable/hot-pluggable.' >&2
            return 1
        fi
    fi
}

jv_migration_existing_mount_for_device() {
    local real
    real="$(readlink -f -- "$1" 2>/dev/null || true)"
    [[ -n ${real} ]] || return 1
    findmnt -rn -S "${real}" -o TARGET 2>/dev/null | head -n1
}

jv_migration_mount_device() {
    local device="$1" mode="$2" removable_only="${3:-no}" fstype existing options actual
    JV_MIGRATION_MEDIA_REMOVABLE="${removable_only}"
    jv_migration_transport_begin
    jv_migration_validate_media_device "${device}" "${removable_only}" || return 1
    fstype="$(blkid -s TYPE -o value "${device}")"
    existing="$(jv_migration_existing_mount_for_device "${device}" || true)"
    if [[ -n ${existing} ]]; then
        if storage_mount_is_critical "${existing}"; then
            echo "ERROR: refusing critical existing mount: ${existing}" >&2
            return 1
        fi
        JV_MIGRATION_MEDIA_MOUNT="$(realpath -m -- "${existing}")"
        JV_MIGRATION_MEDIA_ADMIN_MOUNT=yes
        JV_MIGRATION_OWNED_MOUNT=''
        if [[ ${mode} == export && ! -w ${JV_MIGRATION_MEDIA_MOUNT} ]]; then
            echo "ERROR: existing administrator mount is not writable: ${JV_MIGRATION_MEDIA_MOUNT}" >&2
            return 1
        fi
        echo "Using existing administrator mount: ${JV_MIGRATION_MEDIA_MOUNT}"
        echo 'JustVoxel will not unmount it after migration.'
        return 0
    fi

    JV_MIGRATION_MEDIA_MOUNT="${JV_MIGRATION_TRANSPORT_ROOT}/media"
    install -d -m0700 -o root -g root "${JV_MIGRATION_MEDIA_MOUNT}"
    if [[ ${mode} == import ]]; then
        options='ro,nodev,nosuid,noexec'
    else
        options='rw,nodev,nosuid,noexec'
    fi
    [[ ${fstype} == xfs ]] && options="${options},nouuid"
    if ! mount -t "${fstype}" -o "${options}" "${device}" "${JV_MIGRATION_MEDIA_MOUNT}"; then
        echo "ERROR: ${fstype} was recognized but could not be mounted by this running JustVoxel image/kernel." >&2
        echo 'No filesystem changes were made.' >&2
        return 1
    fi
    actual="$(findmnt -n -o SOURCE --target "${JV_MIGRATION_MEDIA_MOUNT}" 2>/dev/null || true)"
    [[ -n ${actual} ]] || { umount "${JV_MIGRATION_MEDIA_MOUNT}" || true; echo 'ERROR: temporary mount identity could not be verified.' >&2; return 1; }
    JV_MIGRATION_OWNED_MOUNT="${JV_MIGRATION_MEDIA_MOUNT}"
    JV_MIGRATION_MEDIA_ADMIN_MOUNT=no
}

jv_migration_choose_source_under() {
    local base="$1" title="${2:-Select migration source}" selection index manual
    local -a paths=() labels=()
    while IFS= read -r path; do
        [[ -n ${path} ]] || continue
        paths+=("${path}")
        labels+=("${path#${base}/}")
        (( ${#paths[@]} >= 100 )) && break
    done < <(
        find "${base}" -maxdepth 5 \
            \( -type f \( -iname '*.tar.gz' -o -iname '*.tgz' -o -iname '*.tar' -o -iname '*.zip' \) \
               -o -type f -name 'server.properties' \) -print 2>/dev/null \
            | while IFS= read -r p; do
                if [[ $(basename -- "${p}") == server.properties ]]; then dirname -- "${p}"; else printf '%s\n' "${p}"; fi
              done | awk '!seen[$0]++' | sort
    )

    if (( ${#paths[@]} > 0 )); then
        labels+=('Enter path manually' 'Back')
        selection="$(jui_choose "${title}" "${labels[@]}")" || return 2
        [[ ${selection} == Back ]] && return 2
        if [[ ${selection} != 'Enter path manually' ]]; then
            for index in "${!labels[@]}"; do
                if [[ ${labels[index]} == "${selection}" && ${index} -lt ${#paths[@]} ]]; then
                    JV_MIGRATION_SOURCE="${paths[index]}"
                    return 0
                fi
            done
        fi
    fi

    manual="$(jui_input 'Path relative to mounted location, or absolute path')" || return 2
    [[ -n ${manual} ]] || return 2
    if [[ ${manual} == /* ]]; then
        JV_MIGRATION_SOURCE="$(realpath -m -- "${manual}")"
    else
        JV_MIGRATION_SOURCE="$(realpath -m -- "${base}/${manual}")"
    fi
    case "${JV_MIGRATION_SOURCE}/" in
        "$(realpath -m -- "${base}")/"*) ;;
        *) echo 'ERROR: selected path is outside the migration source mount.' >&2; return 1 ;;
    esac
    [[ -e ${JV_MIGRATION_SOURCE} ]] || { echo "ERROR: migration source does not exist: ${JV_MIGRATION_SOURCE}" >&2; return 1; }
}

jv_migration_choose_block_source() {
    local removable_only="$1" device
    echo 'Detected block storage:'
    jv_migration_show_block_devices
    echo
    device="$(jui_input 'Device or partition to mount temporarily (example /dev/sdb1)')" || return 2
    [[ -n ${device} ]] || return 2
    jv_migration_mount_device "${device}" import "${removable_only}" || return 1
    jv_migration_choose_source_under "${JV_MIGRATION_MEDIA_MOUNT}" 'Select migration archive or server directory'
}
