#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
packages="${repo_root}/build_files/packages.env"
build_common="${repo_root}/build_files/build-common.sh"
justfile="${repo_root}/mjust/justfile"
menu="${repo_root}/mjust/libexec/menu"
resources="${repo_root}/mjust/libexec/resources"

fail() {
    echo "FAIL: $*" >&2
    exit 1
}

grep -Eq '^btop \\$' "${packages}" || fail 'btop is not declared as a common JustVoxel package'
grep -Fq 'python3 btop; do' "${build_common}" || fail 'completed image does not verify the btop command'
grep -Fq 'resources:' "${justfile}" || fail 'mjust resources recipe is missing'
grep -Fq '@/usr/libexec/justvoxel/mjust/resources' "${justfile}" || fail 'mjust resources is not routed through the JustVoxel wrapper'
grep -Fq "'System resources'" "${menu}" || fail 'System resources menu entry is missing'
grep -Fq "'System resources') /usr/bin/mjust resources || true ;;" "${menu}" || fail 'System resources must return directly to the System menu without an extra pause'
grep -Fq 'mjust resources' "${menu}" || fail 'mjust resources is not discoverable from the UI/help'
grep -Fq 'Press q to return to JustVoxel' "${menu}" || fail 'System preview does not explain the btop q exit key'
grep -Fq 'Ctrl-C also exits the monitor' "${menu}" || fail 'System preview does not explain Ctrl-C exit behavior'

stderr_file="$(mktemp)"
trap 'rm -f "${stderr_file}"' EXIT
if TERM=dumb bash "${resources}" >/dev/null 2>"${stderr_file}"; then
    fail 'resources wrapper must refuse a non-interactive terminal'
fi
grep -Fq 'System resources requires an interactive terminal.' "${stderr_file}" || fail 'non-interactive resources error is not friendly'
grep -Fq 'ssh -t <server> mjust resources' "${stderr_file}" || fail 'resources wrapper does not explain SSH terminal allocation'

echo 'system resources regression tests passed.'
