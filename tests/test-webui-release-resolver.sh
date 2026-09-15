#!/usr/bin/bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
resolver="${root}/build_files/resolve-webui-release.sh"
tmp="$(mktemp)"
trap 'rm -f "${tmp}"' EXIT

cat > "${tmp}" <<'JSON'
[
  {"tag_name":"v1.0.0","draft":false,"prerelease":false},
  {"tag_name":"v1.1.0-beta.1","draft":false,"prerelease":true},
  {"tag_name":"v1.1.0-beta.2","draft":false,"prerelease":true},
  {"tag_name":"v1.1.0-rc.1","draft":false,"prerelease":true},
  {"tag_name":"v2.0.0-beta.1","draft":true,"prerelease":true},
  {"tag_name":"1.9","draft":false,"prerelease":false}
]
JSON

stable="$(JV_WEBUI_RELEASES_FILE="${tmp}" "${resolver}" select stable)"
testing="$(JV_WEBUI_RELEASES_FILE="${tmp}" "${resolver}" select testing)"
[[ ${stable} == 1.0.0 ]]
[[ ${testing} == 1.1.0-rc.1 ]]

echo 'WebUI release resolver tests passed.'
