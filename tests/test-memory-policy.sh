#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${repo_root}/mjust/libexec/common.sh"

fail() {
    echo "FAIL: $*" >&2
    exit 1
}

TEST_MEM_KIB=0
TEST_SWAP_KIB=$((64 * 1024 * 1024))

# Override only the /proc/meminfo lookup used by suggest_memory_values().
# Swap is intentionally huge and ignored: recommendations must use MemTotal only.
awk() {
    if [[ ${1:-} == '/^MemTotal:/ {print $2}' && ${2:-} == /proc/meminfo ]]; then
        printf '%s\n' "${TEST_MEM_KIB}"
        return 0
    fi
    command awk "$@"
}

check_plan() {
    local gib="$1" expected="$2" actual
    TEST_MEM_KIB=$((gib * 1024 * 1024))
    actual="$(suggest_memory_values)"
    [[ ${actual} == "${expected}" ]] || fail "${gib} GiB physical RAM produced '${actual}', expected '${expected}'"
}

check_plan 4  '2G 3G'
check_plan 6  '2G 3G'
check_plan 8  '4G 6G'
check_plan 12 '4G 6G'
check_plan 16 '6G 8G'
check_plan 32 '8G 12G'

# Prove that swap/zram capacity is not part of the recommendation input.
TEST_MEM_KIB=$((4 * 1024 * 1024))
[[ $(suggest_memory_values) == '2G 3G' ]] || fail 'swap/zram influenced the 4 GiB physical-RAM recommendation'

echo 'Minecraft physical-RAM recommendation tests passed.'
