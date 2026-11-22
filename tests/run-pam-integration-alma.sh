#!/usr/bin/bash
set -euo pipefail

if [[ ${EUID} -ne 0 ]]; then
    echo 'ERROR: PAM integration test runner must run as root inside AlmaLinux.' >&2
    exit 1
fi

[[ -r /etc/pam.d/justvoxel ]] || {
    echo 'ERROR: /etc/pam.d/justvoxel was not mounted into the AlmaLinux test environment.' >&2
    exit 1
}

authselect select local --force >/dev/null

cd /work

readonly test_user=jv-pam-test
readonly old_password='JustVoxel-PAM-Old-2026!'
readonly new_password='JustVoxel-PAM-New-2026!'
readonly final_password='JustVoxel-PAM-Final-2026!'

useradd --create-home "${test_user}"
printf '%s:%s\n' "${test_user}" "${old_password}" | chpasswd

export JUSTVOXEL_PAM_INTEGRATION=1
export JUSTVOXEL_PAM_TEST_USER="${test_user}"
export JUSTVOXEL_PAM_TEST_OLD_PASSWORD="${old_password}"
export JUSTVOXEL_PAM_TEST_NEW_PASSWORD="${new_password}"
export JUSTVOXEL_PAM_TEST_FINAL_PASSWORD="${final_password}"

CGO_ENABLED=1 go test -mod=readonly -v ./internal/systemauth -run '^TestPAMSystemAccountLifecycleIntegration$'
