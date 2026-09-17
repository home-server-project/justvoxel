#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fail() { echo "FAIL: $*" >&2; exit 1; }

ui="${repo_root}/mjust/libexec/update-minecraft"
backend="${repo_root}/mjust/libexec/update-minecraft-backend"

grep -Fq "echo 'Minecraft update'" "${ui}" || fail 'friendly update heading missing'
grep -Fq 'Game version:' "${ui}" || fail 'friendly game-version state missing'
grep -Fq 'Container image:' "${ui}" || fail 'friendly container state missing'
grep -Fq 'Running container:' "${ui}" || fail 'friendly running-container state missing'
grep -Fq 'No update is needed.' "${ui}" || fail 'current-state summary missing'
if grep -Fq 'Running image ID:' "${ui}"; then fail 'raw running image ID leaked into normal update UI'; fi
grep -Fq 'Running image ID:' "${backend}" || fail 'advanced maintenance backend lost image identity detail'
grep -Fq 'exec /usr/libexec/justvoxel/mjust/update-minecraft-backend' "${ui}" || fail 'friendly update UI does not hand maintenance to the safety backend'

echo 'Minecraft update UI regression tests passed.'
