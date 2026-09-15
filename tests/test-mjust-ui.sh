#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${repo_root}/mjust/libexec/common.sh"
# shellcheck disable=SC1091
source "${repo_root}/mjust/libexec/ui.sh"
# shellcheck disable=SC1091
source "${repo_root}/mjust/libexec/player-guidance.sh"

fail() {
    echo "FAIL: $*" >&2
    exit 1
}

visible="$(mktemp)"
trap 'rm -f "${visible}"' EXIT

result="$(choose 'Choose a value' 'one' 'two' 2>"${visible}" <<< '2')"
[[ ${result} == 2 ]] || fail "choose stdout was '${result}', expected exactly '2'"
grep -Fq 'Choose a value' "${visible}" || fail 'choose prompt was not visible'
grep -Fq '1. one' "${visible}" || fail 'choose option 1 was not visible'
grep -Fq '2. two' "${visible}" || fail 'choose option 2 was not visible'

: > "${visible}"
result="$(choose 'No implicit default' 'safe' 'dangerous' 2>"${visible}" <<< $'\n1')"
[[ ${result} == 1 ]] || fail "choose accepted an empty selection as a default"
grep -Fq 'Please enter a valid number.' "${visible}" || fail 'choose did not reject empty input'

: > "${visible}"
result="$(choose_default 'Safe default menu' 1 'recommended' 'other' 2>"${visible}" <<< '')"
[[ ${result} == 1 ]] || fail "choose_default stdout was '${result}', expected exactly '1'"
grep -Fq 'Safe default menu' "${visible}" || fail 'choose_default prompt was not visible'
grep -Fq '1. recommended' "${visible}" || fail 'choose_default option text was not visible'

[[ $(validate_nonroot_id 1000; echo $?) == 0 ]] || fail 'valid non-root UID was rejected'
if validate_nonroot_id 0; then
    fail 'UID/GID 0 must be rejected'
fi

for value in 1 5 10 20 100 999999; do
    validate_positive_int "${value}" || fail "positive player limit rejected: ${value}"
done
for value in 0 -1 abc 10.5; do
    if validate_positive_int "${value}"; then
        fail "invalid player limit accepted: ${value}"
    fi
done
[[ $(player_limit_bucket 5) == five ]] || fail '5-player guidance bucket wrong'
[[ $(player_limit_bucket 10) == ten ]] || fail '10-player guidance bucket wrong'
[[ $(player_limit_bucket 20) == twenty ]] || fail '20-player guidance bucket wrong'
[[ $(player_limit_bucket 100) == high ]] || fail 'high player guidance bucket wrong'

[[ $(normalize_daily_backup_time '04:30') == '04:30' ]] || fail '04:30 should stay 04:30'
[[ $(normalize_daily_backup_time '4:30') == '04:30' ]] || fail '4:30 should normalize to 04:30'
[[ $(normalize_daily_backup_time '23:59') == '23:59' ]] || fail '23:59 should be accepted'
if normalize_daily_backup_time '24:00' >/dev/null 2>&1; then
    fail '24:00 must be rejected'
fi
if normalize_daily_backup_time '12:60' >/dev/null 2>&1; then
    fail '12:60 must be rejected'
fi
if normalize_daily_backup_time '*-*-* 04:30:00' >/dev/null 2>&1; then
    fail 'normal daily-time helper must reject raw systemd calendar syntax'
fi
[[ $(daily_backup_schedule_from_time '4:30') == '*-*-* 04:30:00' ]] || fail 'daily schedule rendering is incorrect'

TERM=dumb
export TERM
[[ $(jui_backend) == none ]] || fail 'TERM=dumb must disable the interactive selector backend'

menu="${repo_root}/mjust/libexec/menu"
storage="${repo_root}/mjust/libexec/storage-provision"
justfile="${repo_root}/mjust/justfile"
start_over="${repo_root}/mjust/libexec/start-over"

for id in setup setup-advanced status players configure service whitelist backups storage update validate logs advanced exit; do
    grep -Fq "${id})" "${menu}" || fail "menu preview/dispatch id missing: ${id}"
done

grep -Fq '[[ -e /etc/containers/systemd/minecraft.container ]]' "${menu}" || fail 'menu must use the non-secret rendered Quadlet as its configured marker'
grep -Fq 'Configure Minecraft' "${menu}" || fail 'configure submenu missing'
grep -Fq 'Maximum players' "${menu}" || fail 'maximum players menu entry missing'
grep -Fq 'Whitelist' "${menu}" || fail 'whitelist menu entry missing'
grep -Fq 'BedrockBuddy' "${menu}" || fail 'generic Bedrock example missing'
grep -Fq 'PlayerOne' "${menu}" || fail 'generic Java example missing'
grep -Fq 'mjust status --details' "${menu}" || fail 'detailed status is not exposed through the advanced menu'
grep -Fq -- '--details' "${repo_root}/mjust/bin/mjust" || fail 'mjust wrapper does not accept status --details'
grep -Fq 'ERASE /dev/...' "${menu}" || fail 'storage preview does not preserve typed-confirmation guidance'
grep -Fq "'Back'" "${repo_root}/mjust/libexec/storage-ui.sh" || fail 'storage UI has no Back option'

# Every useful direct recipe must be discoverable from the interactive interface.
for command in \
    'mjust setup' 'mjust setup-advanced' 'mjust configure' 'mjust configure-max-players' \
    'mjust status' 'mjust start' 'mjust stop' 'mjust restart' 'mjust players' \
    'mjust whitelist-list' 'mjust whitelist-add-java' 'mjust whitelist-remove-java' \
    'mjust whitelist-add-bedrock' 'mjust whitelist-remove-bedrock' 'mjust backup' \
    'mjust update-minecraft' 'mjust logs' 'mjust validate' \
    'mjust storage-disk' 'mjust storage-partition' 'mjust storage-free-space' \
    'mjust storage-network' 'mjust storage-system' 'mjust storage-migrate' 'mjust storage-plan' \
    'mjust welcome' 'mjust welcome-off' 'mjust welcome-on' 'mjust start-over'; do
    grep -Fq "${command}" "${menu}" || fail "direct command is not discoverable in mjust UI: ${command}"
done

grep -Fq 'All mjust commands' "${menu}" || fail 'advanced all-commands entry missing'
grep -Fq '/usr/bin/mjust --list' "${menu}" || fail 'all-commands entry must use authoritative mjust --list output'
grep -Fq 'storage-system:' "${justfile}" || fail 'storage-system direct recipe missing'
grep -Fq 'start-over:' "${justfile}" || fail 'start-over direct recipe missing'
grep -Fq 'system)' "${storage}" || fail 'storage-provision system action missing'

test -f "${start_over}" || fail 'start-over implementation missing'
grep -Fq 'START OVER' "${start_over}" || fail 'start-over exact typed confirmation missing'
grep -Fq 'config-backups' "${start_over}" || fail 'start-over configuration backup path missing'
grep -Fq 'Minecraft world/data will NOT be deleted.' "${start_over}" || fail 'start-over data-preservation statement missing'
grep -Fq 'podman rename minecraft' "${start_over}" || fail 'start-over preserved-container path missing'
grep -Fq 'podman rm minecraft' "${start_over}" || fail 'start-over remove-container path missing'

echo 'mjust UI regression tests passed.'
