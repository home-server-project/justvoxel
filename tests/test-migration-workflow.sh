#!/usr/bin/bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fail(){ echo "FAIL: $*" >&2; exit 1; }
# Static workflow invariants that require the full appliance at runtime.
import_files=(
    "${repo_root}/mjust/libexec/migration-import"
    "${repo_root}/mjust/libexec/migration-import-common.sh"
    "${repo_root}/mjust/libexec/migration-import-plan-source.sh"
    "${repo_root}/mjust/libexec/migration-import-plan-destination.sh"
    "${repo_root}/mjust/libexec/migration-import-activate.sh"
)
import_text="$(cat "${import_files[@]}")"
exporter="${repo_root}/mjust/libexec/migration-export"
transport_files=(
    "${repo_root}/mjust/libexec/migration-transport.sh"
    "${repo_root}/mjust/libexec/migration-transport-device.sh"
    "${repo_root}/mjust/libexec/migration-transport-network.sh"
    "${repo_root}/mjust/libexec/migration-transport-ui.sh"
)
transport_text="$(cat "${transport_files[@]}")"
common="${repo_root}/mjust/libexec/common.sh"
migration_common="${repo_root}/mjust/libexec/migration-common.sh"
template="${repo_root}/templates/config/minecraft.env.in"
menu="${repo_root}/mjust/libexec/menu"
justfile="${repo_root}/mjust/justfile"

# shellcheck disable=SC1090
source "${migration_common}"
available_bytes="$(jv_migration_available_bytes "${repo_root}")" || fail 'migration free-space helper failed on a local path'
[[ ${available_bytes} =~ ^[0-9]+$ ]] || fail 'migration free-space helper did not return a numeric byte count'
(( available_bytes > 0 )) || fail 'migration free-space helper returned no available space'

for text in \
    'Type IMPORT to continue:' \
    'The source eula.txt, if present, is NOT accepted' \
    'plugins are executable server code' \
    'jv_migration_snapshot_runtime' \
    'jv_migration_rollback_data' \
    'jv_migration_assert_fresh_selinux_path' \
    'jv_migration_remove_fresh_selinux_rule' \
    'restore-runtime-validate' \
    '/usr/libexec/justvoxel/mjust/validate' \
    'JUSTVOXEL_REGENERATE_RCON=1' \
    'online-mode=false' \
    'MINECRAFT_VERSION_MODE=pinned'; do
    grep -Fq "${text}" <<< "${import_text}" || fail "import workflow invariant missing: ${text}"
done
for text in '.partial' 'verify-native' 'flock -n' 'jv_player_check_before_interrupt' 'sync -f' 'mv -- "${partial}"'; do
    grep -Fq "${text}" "${exporter}" || fail "export workflow invariant missing: ${text}"
done
for fs in ext4 xfs btrfs vfat exfat; do
    grep -Fq "${fs}" <<< "${transport_text}" || fail "temporary media allowlist missing ${fs}"
done
grep -Fq 'NTFS removable media is not supported' <<< "${transport_text}" || fail 'NTFS refusal is missing'
if grep -Fq 'storage_write_network_fstab' <<< "${transport_text}"; then fail 'temporary migration transport must not write fstab'; fi
grep -Fq 'JV_MIGRATION_SMB_CREDENTIALS="${JV_MIGRATION_TRANSPORT_ROOT}/smb.credentials"' <<< "${transport_text}" || fail 'temporary SMB credentials are not under /run transport state'
grep -Fq 'chmod 0600 "${JV_MIGRATION_SMB_CREDENTIALS}"' <<< "${transport_text}" || fail 'temporary SMB credentials are not mode 0600'
for token in GAME_MODE DIFFICULTY WHITELIST_ENABLED ENFORCE_WHITELIST; do
    grep -Fq "@@${token}@@" "${template}" || fail "Minecraft template missing migration-safe token ${token}"
done
grep -Fq 'GAME_MODE="${GAME_MODE:-survival}"' "${common}" || fail 'backward-compatible game-mode default missing'
grep -Fq 'WHITELIST_ENABLED="${WHITELIST_ENABLED:-yes}"' "${common}" || fail 'backward-compatible whitelist default missing'
grep -Fq 'JUSTVOXEL_REGENERATE_RCON' "${common}" || fail 'RCON regeneration support missing'
grep -Fq 'Migration' "${menu}" || fail 'Migration TUI missing'
grep -Fq 'Import existing Minecraft server' "${menu}" || fail 'fresh-appliance import entry missing'
grep -Fq 'mjust export' "${menu}" || fail 'export command not discoverable in TUI'
grep -Fq 'mjust import' "${menu}" || fail 'import command not discoverable in TUI'
grep -Fq 'export destination=""' "${justfile}" || fail 'mjust export recipe missing'
grep -Fq 'import source=""' "${justfile}" || fail 'mjust import recipe missing'

echo 'migration workflow invariant tests passed.'
