#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${repo_root}/mjust/libexec/common.sh"
# shellcheck disable=SC1091
source "${repo_root}/mjust/libexec/ui.sh"

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

for id in setup setup-advanced status players configure service backups storage update validate logs advanced exit; do
    grep -Fq "${id})" "${repo_root}/mjust/libexec/menu" || fail "menu preview/dispatch id missing: ${id}"
done

grep -Fq 'mjust status --details' "${repo_root}/mjust/libexec/menu" || fail 'detailed status is not exposed through the advanced menu'
grep -Fq -- '--details' "${repo_root}/mjust/bin/mjust" || fail 'mjust wrapper does not accept status --details'
grep -Fq 'ERASE /dev/...' "${repo_root}/mjust/libexec/menu" || fail 'storage preview does not preserve typed-confirmation guidance'

echo 'mjust UI regression tests passed.'
