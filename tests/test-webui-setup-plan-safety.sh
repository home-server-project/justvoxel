#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
helper="${repo_root}/mjust/libexec/admin-setup-plan-json"

[[ -f ${helper} ]] || { echo "missing A4.4.1 helper: ${helper}" >&2; exit 1; }
bash -n "${helper}"

# A4.4.1 is planning only. It may inspect storage and upstream metadata but it
# must not mutate filesystems, mounts, configuration, services, firewall, or
# runtime state.
if grep -Eq '(^|