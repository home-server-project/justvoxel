#!/usr/bin/bash
jv_migration_transport_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${jv_migration_transport_dir}/migration-transport-device.sh"
source "${jv_migration_transport_dir}/migration-transport-network.sh"
source "${jv_migration_transport_dir}/migration-transport-ui.sh"
unset jv_migration_transport_dir
