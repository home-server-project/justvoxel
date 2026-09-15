#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${repo_root}/mjust/libexec/status-lib.sh"

fail() {
    echo "FAIL: $*" >&2
    exit 1
}

[[ $(jv_status_format_duration 90) == '1m' ]] || fail 'duration formatter should use minutes below one hour'
[[ $(jv_status_format_duration 3660) == '1h 1m' ]] || fail 'duration formatter should include hours and minutes'
[[ $(jv_status_format_duration 90061) == '1d 1h 1m' ]] || fail 'duration formatter should include days, hours and minutes'

[[ $(jv_status_overall Running 0 running healthy healthy healthy) == 'Healthy' ]] || fail 'healthy appliance state should report Healthy'
[[ $(jv_status_overall Stopped 0 running healthy healthy healthy) == 'Healthy' ]] || fail 'deliberately stopped Minecraft must not make appliance unhealthy'
[[ $(jv_status_overall Failed 0 running healthy healthy healthy) == 'Attention needed' ]] || fail 'failed Minecraft service must need attention'
[[ $(jv_status_overall Running 2 degraded healthy healthy healthy) == 'Attention needed' ]] || fail 'failed system services must need attention'
[[ $(jv_status_overall Running 0 running problem healthy healthy) == 'Attention needed' ]] || fail 'missing basic network connectivity must need attention'
[[ $(jv_status_overall Running 0 running healthy problem healthy) == 'Attention needed' ]] || fail 'unavailable configured storage must need attention'
[[ $(jv_status_overall Running 0 running healthy healthy problem) == 'Attention needed' ]] || fail 'configured backup timer failure must need attention'
[[ $(jv_status_overall Running 0 unknown healthy healthy healthy) == 'Unknown' ]] || fail 'unknown system state should remain Unknown'

status="${repo_root}/mjust/libexec/status"
grep -Fq "echo 'System'" "${status}" || fail 'status System section missing'
grep -Fq "echo 'Minecraft'" "${status}" || fail 'status Minecraft section missing'
grep -Fq "echo 'Network'" "${status}" || fail 'status Network section missing'
grep -Fq "echo 'Storage'" "${status}" || fail 'status Storage section missing'
grep -Fq "echo 'Backups'" "${status}" || fail 'status Backups section missing'
grep -Fq "echo 'Container'" "${status}" || fail 'status Container section missing'
grep -Fq 'systemctl --failed --type=service' "${status}" || fail 'status must summarize failed system services'
grep -Fq 'Same-filesystem backups help with world/configuration recovery, not physical disk failure.' "${status}" || fail 'same-filesystem backup warning missing'
grep -Fq 'mjust status --details' "${status}" || fail 'status must direct overflow failed-service detail to --details'
grep -Fq "echo 'Failed service detail'" "${status}" || fail 'detailed failed-service view missing'
grep -Fq "echo 'Network detail'" "${status}" || fail 'detailed network view missing'

echo 'status dashboard regression tests passed.'
