#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${repo_root}/mjust/libexec/common.sh"

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

echo 'mjust UI regression tests passed.'
