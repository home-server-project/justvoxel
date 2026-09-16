#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

helper=mjust/libexec/web-status-json
interrupt=mjust/libexec/interrupt-safety.sh
agent=management/cmd/justvoxel-management-agent/main.go
agent_control=management/cmd/justvoxel-management-agent/minecraft.go

grep -Fq "mc_version='Not configured'" "${helper}"
grep -Fq "mc_version='Unavailable'" "${helper}"
grep -Fq 'MINECRAFT_VERSION} != LATEST' "${helper}"
grep -Fq 'Starting minecraft server version' "${helper}"
grep -Fq 'configured:$configured' "${helper}"
grep -Fq 'web_players' "${helper}"
grep -Fq 'web_action' "${helper}"
grep -Fq -- '--confirm-players' "${helper}"

grep -Fq 'JV_INTERRUPT_CONFIRMATION_MODE:-interactive' "${interrupt}"
grep -Fq 'required)' "${interrupt}"
grep -Fq 'return 10' "${interrupt}"
grep -Fq 'confirmed)' "${interrupt}"

grep -Fq 'registerMinecraftRoutes(mux, s)' "${agent}"
grep -Fq 'GET /v1/players' "${agent_control}"
grep -Fq 'POST /v1/minecraft/start' "${agent_control}"
grep -Fq 'POST /v1/minecraft/stop' "${agent_control}"
grep -Fq 'POST /v1/minecraft/restart' "${agent_control}"
if grep -Eq 'exec\.Command(Context)?\([^,]+,[[:space:]]*request\.' "${agent_control}"; then
    echo 'ERROR: Minecraft management endpoint passes request-controlled commands to exec.' >&2
    exit 1
fi

echo 'JustVoxel WebUI Minecraft control policy checks passed.'
