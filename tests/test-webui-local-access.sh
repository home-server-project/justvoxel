#!/usr/bin/bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

web_script=mjust/libexec/web
motd=runtime/justvoxel-motd
unit=system_files/usr/lib/systemd/system/justvoxel-webui.service
firewall=system_files/usr/lib/firewalld/services/justvoxel-web.xml
agent=management/cmd/justvoxel-management-agent/main.go
resolved=system_files/etc/systemd/resolved.conf.d/90-justvoxel-mdns.conf
networkmanager=system_files/etc/NetworkManager/conf.d/90-systemd-resolved.conf
readme=README.md
doc=docs/WEBUI.md

grep -Fq 'readonly port=8099' "${web_script}"
grep -Fq 'http://127.0.0.1:${port}/healthz' "${web_script}"
grep -Fq 'http://$(primary_ipv4):${port}' "${web_script}"
grep -Fq 'wait_for_web_healthy()' "${web_script}"
grep -Fq 'for attempt in {1..10}; do' "${web_script}"
grep -Fq 'sleep 0.5' "${web_script}"
grep -Fq 'if ! wait_for_web_healthy; then' "${web_script}"
grep -Fq 'firewall-cmd --quiet --permanent --add-service=mdns' "${web_script}"
grep -Fq 'MulticastDNS=yes' "${resolved}"
grep -Fq 'dns=systemd-resolved' "${networkmanager}"
grep -Fq 'connection.mdns=2' "${networkmanager}"

grep -Fq "web_state='Starting...'" "${motd}"
grep -Fq "web_state='Ready'" "${motd}"
grep -Fq "web_state='Unavailable'" "${motd}"
grep -Fq "web_state='Disabled'" "${motd}"
grep -Fq 'web_initialized_marker=/var/lib/justvoxel/webui/initialized' "${motd}"
grep -Fq 'systemctl is-failed --quiet justvoxel-web-bootstrap.service' "${motd}"
grep -Fq 'web_browser="http://${host}.local:8099"' "${motd}"
grep -Fq 'web_direct="http://${ipv4}:8099"' "${motd}"
grep -Fq "printf '  Web interface:   %s" "${motd}"
grep -Fq "printf '  Open in browser: %s" "${motd}"
grep -Fq "printf '  Direct address:  %s" "${motd}"
grep -Fq "printf '  Enable with:" "${motd}"
grep -Fq "printf '  Check status:" "${motd}"

grep -Fq -- '--listen 0.0.0.0:8099' "${unit}"
grep -Fq 'port="8099"' "${firewall}"
grep -Fq 'Local HTTP management interface' "${firewall}"

if grep -Fq 'LoadCredential=' "${unit}"; then
    echo 'ERROR: WebUI systemd service still loads TLS credentials.' >&2
    exit 1
fi
if grep -Eq '8443|https://127\.0\.0\.1|curl[[:space:]]+-k' "${web_script}" "${motd}" "${unit}" "${firewall}"; then
    echo 'ERROR: legacy self-signed HTTPS runtime configuration remains.' >&2
    exit 1
fi
if grep -Eq 'ensureTLS|tls\.crt|tls\.key|crypto/x509|crypto/ecdsa|crypto/elliptic' "${agent}"; then
    echo 'ERROR: management agent still contains self-signed TLS generation.' >&2
    exit 1
fi

grep -Fq 'plain HTTP on TCP port `8099`' "${doc}"
grep -Fq 'Do not forward TCP port `8099` directly to the public Internet.' "${doc}"
grep -Fq 'plain HTTP on TCP port `8099`' "${readme}"

echo 'JustVoxel local WebUI HTTP and mDNS discovery policy checks passed.'
