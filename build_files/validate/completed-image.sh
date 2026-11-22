#!/usr/bin/bash
set -euo pipefail
/usr/libexec/justvoxel/health/common
/usr/bin/glances -w --disable-webui --bind 127.0.0.1 --port 61208 --config /etc/glances/glances.conf >/tmp/glances-runtime.log 2>&1 &
glances_pid=$!
cleanup_glances() {
    kill "${glances_pid}" >/dev/null 2>&1 || true
    wait "${glances_pid}" >/dev/null 2>&1 || true
}
trap cleanup_glances EXIT
ready=0
for _ in $(seq 1 20); do
    if curl -fsS --max-time 1 http://127.0.0.1:61208/api/4/all | jq -e "has(\"cpu\") and has(\"mem\")" >/dev/null; then
        ready=1
        break
    fi
    sleep 0.5
done
if [[ "${ready}" -ne 1 ]]; then
    cat /tmp/glances-runtime.log >&2
    exit 1
fi
cleanup_glances
trap - EXIT
rm -f /tmp/glances-runtime.log /tmp/glances-root.log
if [[ "$(cat /usr/lib/justvoxel/variant)" == justvoxel-hws ]]; then
    /usr/libexec/justvoxel/health/hws
fi
bootc container lint --fatal-warnings
