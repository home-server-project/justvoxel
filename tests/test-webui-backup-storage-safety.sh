#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
helper_main="${repo_root}/mjust/libexec/admin-backup-storage-json"
helper_common="${repo_root}/mjust/libexec/admin-backup-storage-common.sh"
helper_apply="${repo_root}/mjust/libexec/admin-backup-storage-apply.sh"
agent="${repo_root}/management/cmd/justvoxel-management-agent/admin_backup_storage.go"

for file in "${helper_main}" "${helper_common}" "${helper_apply}" "${agent}"; do
    [[ -f ${file} ]] || { echo "missing A3.1 file: ${file}" >&2; exit 1; }
done

if grep -Eq '(^|[^[:alnum:]_])(mkfs(\.[[:alnum:]]+)?|wipefs|parted)([^[:alnum:]_]|$)' "${helper_main}" "${helper_common}" "${helper_apply}"; then
    echo 'A3.1 backup storage helpers must not format, erase, or partition disks.' >&2
    exit 1
fi

grep -q 'admin-backup-storage-json' "${agent}"
grep -q 'GET /v1/admin/backup-storage' "${agent}"
grep -q 'POST /v1/admin/backup-storage/plan' "${agent}"
grep -q 'POST /v1/admin/backup-storage/apply' "${agent}"
grep -q 'chmod 0600 "${A31_SMB_CREDENTIALS}"' "${helper_apply}"
grep -q "password is required when applying a new SMB mount" "${helper_apply}"

echo 'WebUI backup storage A3.1 safety checks passed.'
