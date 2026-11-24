#!/usr/bin/bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fail(){ echo "FAIL: $*" >&2; exit 1; }
# Static workflow invariants that require the full appliance at runtime.
import_files=(
    "${repo_root}/mjust/libexec/migration-import-backend"
    "${repo_root}/mjust/libexec/migration-import-common.sh"
    "${repo_root}/mjust/libexec/migration-import-plan-source.sh"
    "${repo_root}/mjust/libexec/migration-import-plan-destination.sh"
    "${repo_root}/mjust/libexec/migration-import-activate.sh"
)
import_text="$(cat "${import_files[@]}")"
import_frontend="${repo_root}/mjust/libexec/migration-import"
import_backend="${repo_root}/mjust/libexec/migration-import-backend"
grep -Fq 'migration-api.sh' "${import_frontend}" || fail 'migration Import frontend does not use the shared API helper'
grep -Fq 'jv_migration_import_plan_review' "${import_frontend}" || fail 'migration Import frontend does not plan through the Agent'
grep -Fq 'jv_migration_import_apply' "${import_frontend}" || fail 'migration Import frontend does not apply through the Agent'
grep -Fq 'Attached disk / partition' "${import_frontend}" || fail 'migration Import attached-disk source choice missing'
grep -Fq 'USB / removable storage' "${import_frontend}" || fail 'migration Import removable source choice missing'
grep -Fq 'Configured JustVoxel backup storage' "${import_frontend}" || fail 'migration Import configured-backup source choice missing'
grep -Fq 'Temporary NFS share' "${import_frontend}" || fail 'migration Import NFS source choice missing'
grep -Fq 'Temporary SMB/CIFS share' "${import_frontend}" || fail 'migration Import SMB source choice missing'
grep -Fq 'source_selection_required' "${import_frontend}" || fail 'migration Import transport source-entry review missing'
grep -Fq 'jv_migration_monitor_operation' "${import_frontend}" || fail 'migration Import frontend does not monitor the persistent Agent operation'
grep -Fq 'interrupt-safety.sh' "${import_backend}" || fail 'shared migration import backend does not load interruption safety'
grep -Fq 'JV_MIGRATION_API_MODE' "${import_backend}" || fail 'shared migration Import backend has no non-interactive Agent mode'
for api_file in "${repo_root}/mjust/libexec/migration-import-common.sh" "${repo_root}/mjust/libexec/migration-import-plan-source.sh" "${repo_root}/mjust/libexec/migration-import-activate.sh"; do
    grep -Fq 'JV_MIGRATION_API_MODE' "${api_file}" || fail "Import backend component lacks Agent-mode handling: ${api_file}"
done
export_frontend="${repo_root}/mjust/libexec/migration-export"
exporter="${repo_root}/mjust/libexec/migration-export-backend"
recovery="${repo_root}/mjust/libexec/migration-recover"
recovery_backend="${repo_root}/mjust/libexec/migration-recover-backend"
transport_files=(
    "${repo_root}/mjust/libexec/migration-transport.sh"
    "${repo_root}/mjust/libexec/migration-transport-device.sh"
    "${repo_root}/mjust/libexec/migration-transport-network.sh"
    "${repo_root}/mjust/libexec/migration-transport-ui.sh"
)
transport_text="$(cat "${transport_files[@]}")"
import_source_helper="${repo_root}/mjust/libexec/migration-import-source.sh"
common="${repo_root}/mjust/libexec/common.sh"
migration_common="${repo_root}/mjust/libexec/migration-common.sh"
validate_backend="${repo_root}/mjust/libexec/validate-backend"
template="${repo_root}/templates/config/minecraft.env.in"
menu="${repo_root}/mjust/libexec/menu"
justfile="${repo_root}/mjust/justfile"

# shellcheck disable=SC1090
source "${migration_common}"
available_bytes="$(jv_migration_available_bytes "${repo_root}")" || fail 'migration free-space helper failed on a local path'
[[ ${available_bytes} =~ ^[0-9]+$ ]] || fail 'migration free-space helper did not return a numeric byte count'
(( available_bytes > 0 )) || fail 'migration free-space helper returned no available space'

# Exercise the selected-instance loader and the actual cold manifest writer.
(
    fixture="$(mktemp -d)"
    trap 'rm -rf -- "${fixture}"' EXIT
    helpers="${repo_root}/mjust/libexec"
    mkdir -p "${fixture}/jv-abcdef" "${fixture}/run"
    source <(sed -e '/^source /d' -e "s|/etc/justvoxel/instances|$fixture|g" \
        -e "s|/etc/justvoxel/minecraft-backup.env|$fixture/no-backup.env|g" "${helpers}/migration-instance.sh")
    # Stand in for authoritative runtime validation at the loader boundary.
    jv_load_instance() {
        [[ $1 == jv-abcdef ]] || return 1
        JV_INSTANCE_ID=$1 JV_INSTANCE_CONTAINER="minecraft-$1" JV_INSTANCE_UNIT="minecraft-$1.service"
        MINECRAFT_SERVER_TYPE="$(jq -er '.runtime.software' "${fixture}/$1/config.json")"
        MINECRAFT_VERSION=1.21.8 MINECRAFT_LOADER_VERSION='' BEDROCK_ENABLED=no JV_NETWORK_MEMBER=no
        JV_INSTANCE_IMAGE_REF=fixture-image DATA_PATH="${fixture}/$MINECRAFT_SERVER_TYPE"
    }
    timedatectl() { echo UTC; }
    podman() { [[ $1 == image ]]; }
    bootc() { return 1; }
    install() { mkdir -p -- "${@: -1}"; }
    if (( EUID != 0 )); then
        chown() { [[ $1 == root:root && -f ${3} && ${3} == "${fixture}/run/"* ]]; }
    fi
    fixture_run="$fixture/run"
    server_reported='' geyser_reported=''
    sed -n '/^# Reinspect after the cold stop/,/^partial="/{ /^partial="/!p; }' "${exporter}" |
        sed -e "s|/usr/libexec/justvoxel/mjust/|$helpers/|g" \
            -e 's/JV_MIGRATION_RUN_ROOT/fixture_run/g' > "${fixture}/manifest.sh"
    for software in paper purpur spigot pufferfish leaf fabric neoforge forge quilt vanilla bedrock future; do
        jq -cn --arg software "$software" '{runtime:{software:$software,max_players:10,
            java_port:25565,bedrock_port:0,bedrock_tcp_port:19131,bedrock_udp_start:19140,
            bedrock_udp_end:19149,java_memory:"2G",container_memory:"4G",game_mode:"survival",
            difficulty:"normal",motd:"fixture"}}' > "${fixture}/jv-abcdef/config.json"
        jv_migration_load_instance jv-abcdef || fail "loader rejected $software"
        if [[ $software == bedrock ]]; then
            mkdir -p "$DATA_PATH/worlds/Bedrock level"
            touch "$DATA_PATH/worlds/Bedrock level/level.dat"
        else
            mkdir -p "$DATA_PATH/world"
            touch "$DATA_PATH/world/level.dat"
            echo 'online-mode=true' > "$DATA_PATH/server.properties"
            case "$software" in
                leaf|pufferfish) touch "$DATA_PATH/paper.yml" ;;
                quilt) mkdir "$DATA_PATH/mods"; touch "$DATA_PATH/mods/example.jar" ;;
            esac
        fi
        source "${fixture}/manifest.sh"
        jq -e --arg software "$software" '.minecraft.implementation==$software and
            .sourceInstanceID=="jv-abcdef" and .container.image=="fixture-image" and
            (if $software=="bedrock" then .minecraft.onlineMode==null and
              .networkHints.bedrockUDPStart==19140 and .plugins.count==0 else .minecraft.onlineMode end)' \
            "${manifest_tmp}" >/dev/null || fail "incorrect $software export manifest"
    done
)

# Shared compatibility and destination-specific availability, without network access.
api_planner="${repo_root}/mjust/libexec/admin-migration-import-plan-json"
source_planner="${repo_root}/mjust/libexec/migration-import-plan-source.sh"
destination_planner="${repo_root}/mjust/libexec/migration-import-plan-destination.sh"
# Exercise the production planner through its staging-space check, before
# transaction creation. Only the appliance's root ownership is simulated.
(
    fixture="$(mktemp -d)"
    trap 'rm -rf -- "${fixture}"' EXIT
    wrapper="${fixture}/minecraft-instances/jv-abcdef"
    DATA_PATH="${wrapper}/data"
    mkdir -p -- "${DATA_PATH}" "${fixture}/source"
    sed 's|/usr/libexec/justvoxel/mjust/migration-archive|fixture_archive|g' "${source_planner}" > "${fixture}/planner.sh"
    fixture_archive() { echo '{"expandedBytes":1,"nativeBundle":false}'; }
    jv_migration_validate_source_name() { :; }
    jv_migration_source_identity() { echo fixture; }
    jv_migration_validate_data() { :; }
    jv_restore_validate_data_layout() { :; }
    stat() {
        if [[ $* == "-c %u:%g:%a -- ${wrapper}" ]]; then
            command stat -c '0:0:%a' -- "${wrapper}"
        else
            command stat "$@"
        fi
    }
    jv_migration_require_staging_space() {
        [[ $1 == "${wrapper}" ]] || exit 2
        exit 42
    }
    for mode in 700 755; do
        chmod "${mode}" -- "${wrapper}"
        result=0
        (source "${fixture}/planner.sh" "${fixture}/source") || result=$?
        expected=42
        [[ ${mode} == 700 ]] || expected=1
        [[ ${result} == "${expected}" && $(command stat -c '%a' -- "${wrapper}") == "${mode}" ]] ||
            fail "Import planner changed or incorrectly validated wrapper mode ${mode}"
    done
)
(
    source "${common}"
    server_supports_plugins paper || fail 'Paper must support plugins'
    server_supports_plugins purpur || fail 'Purpur must support plugins'
    if server_supports_plugins vanilla; then fail 'Vanilla must not support plugins'; fi

    availability_fixture="$(mktemp -d)"
    trap 'rm -rf -- "${availability_fixture}"' EXIT
    curl() { fail 'availability fixture attempted network access'; }
    paper_version_build_channel() {
        printf 'paper %s\n' "$1" >> "${availability_fixture}/calls"
        [[ $1 == 27.1 ]]
    }
    purpur_metadata() {
        printf 'purpur metadata\n' >> "${availability_fixture}/calls"
        printf '%s' '{"versions":["28.3"]}'
    }
    MINECRAFT_SERVER_TYPE=paper
    source_version=27.1
    server_version_available "$source_version" || fail 'Paper availability did not use the destination helper'
    [[ $(cat "${availability_fixture}/calls") == 'paper 27.1' ]] || fail 'wrong Paper availability path'
    if server_version_available 28.3; then fail 'unavailable Paper version accepted'; fi
    rm -f "${availability_fixture}/calls"
    MINECRAFT_SERVER_TYPE=purpur
    source_version=28.3
    server_version_available "$source_version" || fail 'Purpur availability did not use the destination helper'
    [[ $(cat "${availability_fixture}/calls") == 'purpur metadata' ]] || fail 'wrong Purpur availability path'
    if server_version_available 27.1; then fail 'unavailable Purpur version accepted'; fi
)
for planner in "${api_planner}" "${source_planner}"; do
    grep -Fq 'server_supports_plugins "${MINECRAFT_SERVER_TYPE-paper}"' "${planner}" || fail 'planner lacks shared destination compatibility gate'
    grep -Fq 'server_allows_in_place_switch' "${planner}" || fail 'planner lacks native compatibility authority'
    grep -Fq 'jv_migration_import_version_available' "${planner}" || fail 'planner lacks destination version availability preflight'
    if grep -Fq 'paper_version_has_stable_build' "${planner}"; then fail 'Import uses a Paper-only availability check'; fi
done
grep -Fq 'instance-version-publish' "${repo_root}/mjust/libexec/migration-import-activate.sh" || fail 'Import does not pin the source version'
grep -Fq '"${source_version}" pinned' "${repo_root}/mjust/libexec/migration-import-activate.sh" || fail 'Import does not retain the exact source version'

# Exercise the actual API planner with bounded source/configuration fixtures.
# Appliance helpers are replaced only in this temporary test copy.
(
    source "${common}"
    review_fixture="$(mktemp -d)"
    trap 'rm -rf -- "${review_fixture}"' EXIT
    export REVIEW_FIXTURE="${review_fixture}"
    mkdir -p "${review_fixture}/server" "${review_fixture}/helpers"
    REVIEW_CONFIG="${review_fixture}/config"
    REVIEW_QUADLET="${review_fixture}/quadlet"
    touch "$REVIEW_CONFIG" "$REVIEW_QUADLET"
    sed -e '/^source /d' -e '/^require_root$/d' -e '/^\[\[ ${1:-} == --instance/d' -e '/^jv_migration_load_instance /d' -e '/^shift 2$/d' \
        -e '/^case "${1:-discover}" in/,$d' \
        -e 's/JV_MIGRATION_INSTANCE_CONFIG/REVIEW_CONFIG/g; s/JV_MIGRATION_INSTANCE_QUADLET/REVIEW_QUADLET/g' \
        -e "s|/usr/libexec/justvoxel/mjust/|${review_fixture}/helpers/|g" \
        "$api_planner" > "${review_fixture}/planner.sh"
    source "${review_fixture}/planner.sh"
    cat > "${review_fixture}/helpers/migration-archive" <<'FIXTURE'
#!/usr/bin/bash
set -euo pipefail
case "$1" in
    inspect) jq -cn --argjson native "$REVIEW_NATIVE" '{expandedBytes:1024,nativeBundle:$native}' ;;
    extract|verify-native|native-candidate) exec "$REVIEW_ARCHIVE" "$@" ;;
    detect) [[ $REVIEW_NATIVE == false ]] || exit 1; jq -cn --arg root "$REVIEW_FIXTURE/server" --arg type "$REVIEW_CANDIDATE_TYPE" '{candidates:[{root:$root,sourceType:$type,minecraftVersion:"26.2",onlineMode:true,maxPlayers:10,javaPortHint:25565,pluginJarCount:0}]}' ;;
    *) exit 1 ;;
esac
FIXTURE
    cat > "${review_fixture}/helpers/validate-data-mount" <<'FIXTURE'
#!/usr/bin/bash
exit 0
FIXTURE
    cat > "${review_fixture}/helpers/web-status-json" <<'FIXTURE'
#!/usr/bin/bash
printf '%s\n' '{"state":"stopped","online":0,"names":[]}'
FIXTURE
    chmod +x "${review_fixture}/helpers/"*
    export REVIEW_ARCHIVE="$repo_root/mjust/libexec/migration-archive"
    make_native_source() {
        local software="$1" version=26.2 loader='' data="$review_fixture/native-data"
        rm -rf "$data"
        mkdir -p "$data"
        if [[ $software == bedrock ]]; then
            version=1.21.90.3
            mkdir -p "$data/worlds/fixture"
            touch "$data/worlds/fixture/level.dat"
        else
            mkdir -p "$data/world"
            touch "$data/world/level.dat"
            echo online-mode=true > "$data/server.properties"
            case "$software" in
                pufferfish|leaf) touch "$data/paper.yml" ;;
                quilt) mkdir "$data/mods"; touch "$data/mods/quilt.jar" ;;
            esac
            if server_is_mod_loader "$software"; then loader=1.2.3; fi
        fi
        jq -cn --arg software "$software" --arg version "$version" --arg loader "$loader" \
            '{schema:"org.justvoxel.migration",schemaVersion:1,minecraft:{implementation:$software,version:$version,loaderVersion:$loader}}' > "$review_fixture/manifest.json"
        rm -f "$review_fixture/native.tar.gz"
        "$REVIEW_ARCHIVE" create-native "$data" "$review_fixture/manifest.json" "$review_fixture/native.tar.gz"
    }
    require_config() { :; }
    jv_migration_require_staging_space() { :; }
    jv_migration_import_transport_identity() { echo fixture; }
    jv_migration_validate_data() { :; }
    jv_migration_players_json() { printf '%s\n' '{"state":"stopped","online":0,"names":[]}'; }
    jv_migration_import_source_prepare() {
        JV_MIGRATION_IMPORT_SOURCE_PATH="${review_fixture}/server"
        if [[ $REVIEW_NATIVE == true ]]; then JV_MIGRATION_IMPORT_SOURCE_PATH="$review_fixture/native.tar.gz"; fi
        JV_MIGRATION_IMPORT_SOURCE_IDENTITY="$(printf '%s\n%s\n' fixture "$DATA_PATH" | sha256sum | awk '{print $1}')"
    }
    jv_migration_import_source_cleanup() { :; }
    jv_restore_validate_data_layout() { :; }
    jv_migration_check_candidate_port() { :; }
    # A rejected software plan must never ask for version availability.
    server_version_available() {
        if server_is_mod_loader "$MINECRAFT_SERVER_TYPE"; then
            [[ $MINECRAFT_LOADER_VERSION == 1.2.3 ]] || return 1
        fi
        printf '%s\n' "$MINECRAFT_SERVER_TYPE" >> "${review_fixture}/availability"
    }
    DATA_PATH="${review_fixture}/server" BACKUP_PATH="${review_fixture}/backups"
    JAVA_PORT=25565 BEDROCK_PORT=19132 BEDROCK_ENABLED=no
    JAVA_MEMORY=4G CONTAINER_MEMORY=6G TIMEZONE=UTC MINECRAFT_IMAGE_TAG=stable
    MINECRAFT_UID=1001 MINECRAFT_GID=1001 BACKUP_KEEP=7
    BACKUP_SCHEDULE='*-*-* 04:30:00' BACKUP_TIMER_ENABLED=yes
    review_request="$(jq -cn --arg path "$DATA_PATH" '{source:{kind:"local",path:$path},destination:{java_port:25565,bedrock_port:19132,backup_keep:7,backup_automatic:true,storage:{},backups:{}}}')"
    for source_software in paper purpur; do
        export REVIEW_SOURCE_TYPE="$source_software" REVIEW_NATIVE=true REVIEW_CANDIDATE_TYPE=itzg-paper
        make_native_source "$source_software"
        MINECRAFT_SERVER_TYPE=vanilla
        rejected="$(plan <<< "$review_request")"
        jq -e --arg source "$source_software" --arg data "$DATA_PATH" '
            .ok==false and .code=="unsupported_server_type" and
            .normalized.source.server_type==$source and .normalized.source.minecraft_version=="26.2" and
            .normalized.source.online_mode=="online" and .normalized.source.max_players==10 and
            .normalized.destination.server_type=="vanilla" and .normalized.destination.data_path==$data and
            .normalized.destination.java_memory=="4G" and
            .requirements==null and .context==null and .plan_fingerprint==null
        ' >/dev/null <<< "$rejected" || fail 'incompatible plan lost review facts or became Apply-ready'
        [[ ! -e ${review_fixture}/availability ]] || fail 'rejected plan checked version availability'
        for destination_software in paper purpur; do
            MINECRAFT_SERVER_TYPE="$destination_software"
            compatible="$(plan <<< "$review_request")"
            jq -e --arg source "$source_software" --arg destination "$destination_software" '
                .ok==true and .normalized.source.server_type==$source and
                .normalized.destination.server_type==$destination and
                .requirements.import_confirmation_required==true and
                .requirements.vanilla_confirmation_required==false and .context!=null
            ' >/dev/null <<< "$compatible" || fail 'compatible Paper/Purpur planning changed'
            [[ $(cat "${review_fixture}/availability") == "$destination_software" ]] || fail 'compatible plan skipped version availability'
            rm "${review_fixture}/availability"
        done
    done
    for software in paper purpur spigot pufferfish leaf fabric neoforge forge quilt vanilla bedrock; do
        export REVIEW_NATIVE=true
        MINECRAFT_SERVER_TYPE="$software"
        MINECRAFT_IMAGE_TAG=java25
        if [[ $software == spigot ]]; then MINECRAFT_IMAGE_TAG=java25-jdk; fi
        if [[ $software == bedrock ]]; then MINECRAFT_IMAGE_TAG=stable; fi
        JAVA_PORT=25565 BEDROCK_PORT=0
        if [[ $software == bedrock ]]; then JAVA_PORT=0; fi
        make_native_source "$software"
        compatible="$(plan <<< "$review_request")"
        jq -e --arg software "$software" '.ok and .normalized.source.candidate_type==$software
            and .normalized.source.server_type==$software and .normalized.destination.server_type==$software
            and (.requirements.online_mode_confirmation_required|not)' <<< "$compatible" >/dev/null || fail "native $software plan failed"
        if server_is_mod_loader "$software"; then
            jq -e '.normalized.source.loader_version=="1.2.3"' <<< "$compatible" >/dev/null || fail 'review lost loader'
        fi
        case "$software" in
            fabric) MINECRAFT_SERVER_TYPE=forge ;;
            bedrock) MINECRAFT_SERVER_TYPE=paper ;;
            vanilla) MINECRAFT_SERVER_TYPE=bedrock ;;
            *) continue ;;
        esac
        rejected="$(plan <<< "$review_request")"
        jq -e '.ok==false and .code=="unsupported_server_type"' <<< "$rejected" >/dev/null || fail 'unsupported conversion accepted'
    done
    JAVA_PORT=25565 BEDROCK_PORT=19132
    export REVIEW_SOURCE_TYPE=vanilla REVIEW_NATIVE=false REVIEW_CANDIDATE_TYPE=vanilla
    MINECRAFT_SERVER_TYPE=paper
    guarded="$(plan <<< "$review_request")"
    jq -e '
        .ok==true and .normalized.source.server_type=="vanilla" and
        .normalized.destination.server_type=="paper" and
        .requirements.vanilla_confirmation_required==true and
        any(.warnings[]; .code=="vanilla_conversion") and .context!=null
    ' >/dev/null <<< "$guarded" || fail 'guarded Vanilla to Paper planning changed'
)

# Exercise the managed-container ownership decision without a Podman daemon.
port_fixture="$(mktemp -d)"
trap 'rm -rf -- "${port_fixture}"' EXIT
cat > "${port_fixture}/podman" <<'EOF'
#!/usr/bin/bash
case "$1 $2 $3 $4 $5" in
    'inspect --type container --format {{.State.Running}}')
        [[ $6 == minecraft-jv-aaaaaa && ${FAKE_MINECRAFT_STATE:-missing} != missing ]] || exit 1
        printf '%s\n' "${FAKE_MINECRAFT_STATE}"
        ;;
    'port minecraft-jv-aaaaaa 19132/udp  ')
        [[ ${FAKE_MINECRAFT_MAPPING:-missing} != missing ]] || exit 1
        printf '%s\n' "${FAKE_MINECRAFT_MAPPING}"
        ;;
    *) exit 1 ;;
esac
EOF
cat > "${port_fixture}/ss" <<'EOF'
#!/usr/bin/bash
if [[ ${FAKE_TCP_LISTENER:-no} == yes && $2 == -ltn ]]; then
    printf '%s\n' 'LISTEN 0 0 0.0.0.0:19132 0.0.0.0:*'
fi
if [[ ${FAKE_UDP_LISTENER:-no} == yes && $2 == -lun ]]; then
    printf '%s\n' 'UNCONN 0 0 0.0.0.0:19132 0.0.0.0:*'
fi
EOF
chmod 0755 "${port_fixture}/podman" "${port_fixture}/ss"
export PATH="${port_fixture}:${PATH}"
JV_INSTANCE_ID=jv-aaaaaa JV_INSTANCE_CONTAINER=minecraft-jv-aaaaaa
export FAKE_MINECRAFT_STATE=running FAKE_MINECRAFT_MAPPING=0.0.0.0:19132 FAKE_UDP_LISTENER=yes
FAKE_MINECRAFT_STATE=true
jv_migration_running_minecraft_owns_port udp 19132 || fail 'running minecraft UDP mapping was not recognized'
if jv_migration_running_minecraft_owns_port tcp 19132; then fail 'UDP mapping was recognized as TCP'; fi
FAKE_MINECRAFT_STATE=false
if jv_migration_running_minecraft_owns_port udp 19132; then fail 'stopped minecraft was recognized as owning a port'; fi
FAKE_MINECRAFT_STATE=missing
if jv_migration_running_minecraft_owns_port udp 19132; then fail 'missing minecraft was recognized as owning a port'; fi
FAKE_MINECRAFT_STATE=true FAKE_MINECRAFT_MAPPING=0.0.0.0:19133
if jv_migration_running_minecraft_owns_port udp 19132; then fail 'different host mapping was recognized as requested port'; fi
FAKE_MINECRAFT_MAPPING=0.0.0.0:19132
jv_migration_check_import_bedrock_port 19132 yes 19132 || fail 'enabled current Bedrock port was rejected'
jv_migration_check_import_bedrock_port 19132 no 19133 || fail 'running managed Bedrock port was rejected after cross-play disable'
# A legacy UDP mapping never grants ownership of an unrelated TCP listener.
export FAKE_TCP_LISTENER=yes
if jv_migration_check_import_bedrock_port 19132 yes 19132 2>/dev/null; then fail 'legacy enabled cross-play hid a TCP conflict'; fi
FAKE_TCP_LISTENER=no
FAKE_MINECRAFT_MAPPING=0.0.0.0:19133
if jv_migration_check_import_bedrock_port 19132 no 19132 2>/dev/null; then fail 'unrelated Bedrock listener was allowed'; fi
grep -Fq 'jv_migration_check_import_bedrock_port "$bedrock_port" "$BEDROCK_ENABLED" "$BEDROCK_PORT"' "${repo_root}/mjust/libexec/admin-migration-import-plan-json" || fail 'API Import planner does not use shared Bedrock check'
grep -Fq 'jv_migration_load_instance "${JV_INSTANCE_ID}"' "${repo_root}/mjust/libexec/migration-import-plan-destination.sh" || fail 'CLI Import planner does not use shared Bedrock check'

recovery_transaction="${repo_root}/mjust/libexec/admin-migration-recovery-transaction-json"
validator_line='    jv_migration_validate_current >&2'
grep -Fxq "${validator_line}" "${recovery_transaction}" || fail 'recovery validator stdout is not isolated from JSON events'
cat > "${port_fixture}/restore-runtime-validate" <<'EOF'
#!/usr/bin/bash
echo 'RCON: checked'
echo 'Minecraft: checked'
[[ ${FAKE_VALIDATOR_FAIL:-no} == no ]]
EOF
chmod 0755 "${port_fixture}/restore-runtime-validate"
validator_call="${validator_line/jv_migration_validate_current/${port_fixture}\/restore-runtime-validate}"
bash -e -c "printf '%s\\n' '{\"event\":\"progress\"}'; ${validator_call}; printf '%s\\n' '{\"event\":\"result\"}'" > "${port_fixture}/recovery.out" 2> "${port_fixture}/recovery.err" || fail 'successful recovery validator fixture failed'
[[ $(wc -l < "${port_fixture}/recovery.out") == 2 ]] || fail 'validator diagnostics contaminated recovery JSON stdout'
jq -e '.event == "progress" or .event == "result"' "${port_fixture}/recovery.out" >/dev/null || fail 'recovery fixture emitted non-JSON stdout'
grep -Fq 'RCON: checked' "${port_fixture}/recovery.err" || fail 'validator diagnostics were not preserved on stderr'
if FAKE_VALIDATOR_FAIL=yes bash -e -c "${validator_call}; printf '%s\\n' '{\"event\":\"result\"}'" > "${port_fixture}/failed.out" 2> "${port_fixture}/failed.err"; then fail 'validator failure did not fail recovery'; fi
[[ ! -s "${port_fixture}/failed.out" ]] || fail 'failed validator emitted recovery result'

for text in \
    'Type IMPORT to continue:' \
    'plugins are executable server code' \
    'jv_migration_snapshot_runtime' \
    'jv_stop_minecraft_adaptive' \
    'online-mode=false' \
    'mjust migration-recover'; do
    grep -Fq "${text}" <<< "${import_text}" || fail "import workflow invariant missing: ${text}"
done
grep -Fq 'migration-api.sh' "${export_frontend}" || fail 'migration Export frontend does not use shared API helper'
grep -Fq "stat -Lc 'local:%d:%i'" "${repo_root}/mjust/libexec/admin-migration-export-plan-json" || fail 'Export local target identity still depends on mutable directory metadata'
grep -Fq "stat -Lc '%d:%i' \"\$DATA_PATH\"" "${repo_root}/mjust/libexec/admin-migration-export-plan-json" || fail 'Export data identity still depends on mutable world size/mtime'
if grep -Fq "local:%d:%i:%Y" "${repo_root}/mjust/libexec/admin-migration-export-plan-json"; then fail 'Export local target identity includes mutable mtime'; fi
if grep -Fq "%d:%i:%s:%Y" "${repo_root}/mjust/libexec/admin-migration-export-plan-json"; then fail 'Export data identity includes mutable size/mtime'; fi
for text in '.partial' 'verify-native' 'flock -n' 'jv_player_check_before_interrupt' 'sync -f' 'mv -- "${partial}"'; do
    grep -Fq "${text}" "${exporter}" || fail "shared export backend invariant missing: ${text}"
done
for fs in ext4 xfs btrfs vfat exfat; do
    grep -Fq "${fs}" <<< "${transport_text}" || fail "temporary media allowlist missing ${fs}"
done
grep -Fq 'NTFS removable media is not supported' <<< "${transport_text}" || fail 'NTFS refusal is missing'
grep -Fq 'jv_migration_import_source_prepare' "${import_source_helper}" || fail 'shared Import source resolver is missing'
grep -Fq 'jv_migration_import_source_entries_json' "${import_source_helper}" || fail 'shared Import source candidate discovery is missing'
grep -Fq 'jv_migration_import_source_cleanup' "${import_source_helper}" || fail 'shared Import source cleanup is missing'
bash -n "${import_source_helper}" || fail 'shared Import source resolver failed bash syntax validation'
grep -Fq 'source_kinds:["local","backup","device","nfs","smb"]' "${repo_root}/mjust/libexec/admin-migration-import-plan-json" || fail 'Import discovery does not advertise full source transport parity'
grep -Fq 'JV_MIGRATION_IMPORT_SOURCE_IDENTITY' "${repo_root}/mjust/libexec/admin-migration-import-transaction-json" || fail 'Import execution does not revalidate remounted source identity'
if grep -Fq 'admin-migration-import-plan-json plan' "${repo_root}/mjust/libexec/admin-migration-import-transaction-json"; then fail 'Import transaction must not re-plan the archive after Agent Apply review'; fi
if grep -Fq 'postplan=' "${repo_root}/mjust/libexec/admin-migration-import-transaction-json"; then fail 'Import transaction must not post-failure re-plan or retry the archive'; fi
if grep -Fq 'jv_migration_source_identity "${JV_MIGRATION_SOURCE}"' "${repo_root}/mjust/libexec/migration-import-activate.sh"; then fail 'Import activation must not re-read the original source after staging'; fi
grep -Fq 'jv_migration_api_result "${preactivation_outcome}" pre-activation' "${repo_root}/mjust/libexec/migration-import-common.sh" || fail 'Import backend does not report bounded pre-activation failure directly'
activation="${repo_root}/mjust/libexec/migration-import-activate.sh"
grep -Fq 'jv_migration_start_with_maintenance_handoff' "${activation}" || fail 'Import startup bypassed handoff'
grep -Fq 'if jv_migration_start_with_maintenance_handoff' "${repo_root}/mjust/libexec/migration-import-common.sh" || fail 'preactivation restart bypassed handoff'
grep -Fq 'flock -w 5 9' "${repo_root}/mjust/libexec/instance-crossplay-start" || fail 'ExecStartPre maintenance exclusion missing'
preflight_line="$(grep -n '^/usr/libexec/justvoxel/management-agent instance-version-migration-preflight ' "${activation}" | cut -d: -f1)"
for marker in 'final_players=' 'jv_stop_minecraft_adaptive ' 'live_modified=yes' 'jv_migration_activate_data ' 'instance-version-publish '; do
    line="$(grep -nF "${marker}" "${activation}" | head -n1 | cut -d: -f1)"
    (( preflight_line > 0 && preflight_line < line )) || fail "publication preflight must precede ${marker}"
done
for writer in migration-import-common.sh migration-import-plan-destination.sh migration-export-backend; do
    grep -Fq 'jv_migration_private_temp' "${repo_root}/mjust/libexec/${writer}" || fail "private metadata bypass in ${writer}"
done
for target in '${transaction}/state' '${transaction}/evidence.json' '${JV_MIGRATION_RECOVERY_ROOT}/${key}'; do
    grep -Fq 'jv_migration_publish_private "${tmp}" "'"${target}"'"' "${migration_common}" || fail "private publication bypass for ${target}"
done
grep -Fq 'jv_migration_publish_private "${portable_tmp}" "${transaction}/portable.json"' "${repo_root}/mjust/libexec/migration-import-plan-destination.sh" || fail 'portable settings publication bypass'
grep -Fq 'jv_migration_publish_private "${tmp}" "${result_file}"' "${repo_root}/mjust/libexec/migration-import-common.sh" || fail 'result publication bypass'
[[ $(grep -c '="$(jv_migration_online_count)"' "${activation}") == 2 ]] || fail 'Import needs shared initial and final player checks'
if grep -Fq 'rcon-cli' "${activation}"; then fail 'generic activation depends on Java RCON'; fi
grep -Fq 'jv_migration_validate_runtime "${source_version}"' "${activation}" || fail 'Import does not verify the concrete source version'
if grep -Eq 'rcon-cli.*version' "${activation}" "${repo_root}/mjust/libexec/restore-runtime-validate"; then fail 'Import/restore bypasses readiness-aware version verification'; fi
grep -Fq 'verify-minecraft-stack "${1:-${MINECRAFT_VERSION}}"' "${repo_root}/mjust/libexec/restore-runtime-validate" || fail 'restore does not use the authoritative version verifier'
import_transaction="${repo_root}/mjust/libexec/admin-migration-import-transaction-json"
if grep -Eq '/usr/libexec/justvoxel/mjust/(restore-runtime-validate|validate|validate-backend)([[:space:]]|$)' "${import_transaction}"; then fail 'API wrapper repeats backend validation'; fi
# Backend success must pass through the journal's verifying state without validation work.
import_success_branch="$(sed -n '/^[[:space:]]*succeeded)/,/^[[:space:]]*;;/p' "${import_transaction}")"
[[ $(grep -c 'emit_progress verifying import_verify ' "${import_transaction}") == 1 ]] || fail 'Import transaction must emit exactly one verifying event'
grep -Fq 'emit_progress verifying import_verify ' <<< "${import_success_branch}" || fail 'Import backend success does not finalize the verifying state'
if grep -Eq 'restore-runtime-validate|validate-backend|(^|[^[:alnum:]_-])validate([^[:alnum:]_-]|$)' <<< "${import_success_branch}"; then fail 'Import success branch repeats backend validation'; fi
execute_line="$(grep -n '^emit_progress running import_execute ' "${import_transaction}" | cut -d: -f1)"
verify_line="$(grep -n '^[[:space:]]*emit_progress verifying import_verify ' "${import_transaction}" | cut -d: -f1)"
result_line="$(grep -n '^[[:space:]]*emit_result succeeded "\$status"$' "${import_transaction}" | cut -d: -f1)"
[[ -n ${execute_line} && -n ${verify_line} && -n ${result_line} ]] || fail 'Import transaction success event sequence is incomplete'
(( execute_line < verify_line && verify_line < result_line )) || fail 'Import transaction must emit running, verifying, then succeeded'
if grep -Fq 'Migration and Minecraft-version upgrade remain separate' "${activation}"; then fail 'Import completion retains stale version-upgrade wording'; fi
runtime_line="$(grep -n '^jv_migration_validate_runtime ' "${activation}" | cut -d: -f1)"
validated_line="$(grep -n '^jv_migration_write_state .* validated ' "${activation}" | cut -d: -f1)"
success_line="$(grep -n '^jv_migration_api_result succeeded validated ' "${activation}" | cut -d: -f1)"
(( runtime_line < validated_line && validated_line < success_line )) || fail 'success precedes selected runtime validation'
grep -Fq 'jv_migration_recover_selected' "${repo_root}/mjust/libexec/migration-import-common.sh" || fail 'rollback does not use selected durable evidence'
grep -Fq 'if [[ ${JUSTVOXEL_RUNTIME_ALREADY_VALIDATED:-0} == 1 ]]; then' "${validate_backend}" || fail 'full validation lacks explicit runtime-only skip'
if grep -Fq 'storage_write_network_fstab' <<< "${transport_text}"; then fail 'temporary migration transport must not write fstab'; fi
grep -Fq 'JV_MIGRATION_SMB_CREDENTIALS="${JV_MIGRATION_TRANSPORT_ROOT}/smb.credentials"' <<< "${transport_text}" || fail 'temporary SMB credentials are not under /run transport state'
grep -Fq 'chmod 0600 "${JV_MIGRATION_SMB_CREDENTIALS}"' <<< "${transport_text}" || fail 'temporary SMB credentials are not mode 0600'
for token in GAME_MODE DIFFICULTY WHITELIST_ENABLED ENFORCE_WHITELIST; do
    grep -Fq "@@${token}@@" "${template}" || fail "Minecraft template missing migration-safe token ${token}"
done
grep -Fq 'GAME_MODE="${GAME_MODE:-survival}"' "${common}" || fail 'backward-compatible game-mode default missing'
grep -Fq 'WHITELIST_ENABLED="${WHITELIST_ENABLED:-yes}"' "${common}" || fail 'backward-compatible whitelist default missing'
grep -Fq 'JUSTVOXEL_REGENERATE_RCON' "${common}" || fail 'RCON regeneration support missing'

grep -Fq 'migration-api.sh' "${recovery}" || fail 'migration Recovery frontend does not use shared API helper'
grep -Fq 'jv_migration_recovery_plan' "${recovery}" || fail 'migration Recovery frontend does not plan through the Agent'
grep -Fq 'jv_migration_recovery_apply' "${recovery}" || fail 'migration Recovery frontend does not apply through the Agent'
if grep -Fq 'rolled-back-fresh' "${recovery_backend}"; then fail 'fresh recovery remains active'; fi
grep -Fq 'admin-migration-recovery-transaction-json --instance' "${recovery_backend}" || fail 'configured migration recovery must validate the restored Minecraft runtime'
grep -Fq 'jv_migration_validate_evidence' "${migration_common}" || fail 'fresh migration recovery does not prove generated runtime files are absent'
grep -Fq 'FINALIZE ROLLBACK' "${recovery_backend}" || fail 'migration recovery destructive confirmation missing'
grep -Fq 'jv_migration_clear_recovery' "${repo_root}/mjust/libexec/admin-migration-recovery-transaction-json" || fail 'migration recovery registry cleanup missing'

grep -Fq "warn '/dev/zram0 is not available" "${validate_backend}" || fail 'missing advisory zram-disabled validation path'
grep -Fq "warn 'zram0 is not active as swap" "${validate_backend}" || fail 'missing advisory zram-inactive validation path'
if grep -Fq "fail '/dev/zram0 is not available" "${validate_backend}"; then fail 'zram absence must not be a fatal appliance validation failure'; fi
if grep -Fq "fail 'zram0 is not active as swap" "${validate_backend}"; then fail 'zram inactivity must not be a fatal appliance validation failure'; fi

grep -Fq 'Migration' "${menu}" || fail 'Migration TUI missing'
if grep -Fq 'Import existing Minecraft server' "${menu}"; then fail 'fresh-appliance menu must not expose an import entry'; fi
grep -Fq "jui_choose 'Migration' 'Export server' 'Import server'" "${menu}" || fail 'Migration submenu import entry missing'
grep -Fq "'Export server') migration_instance export ;;" "${menu}" || fail 'Migration submenu export entry does not dispatch through migration_instance'
grep -Fq "'Import server') migration_instance import ;;" "${menu}" || fail 'Migration submenu import entry does not dispatch through migration_instance'
grep -Fq 'Recover / finalize interrupted import' "${menu}" || fail 'migration recovery entry missing from TUI'
grep -Fq 'mjust export' "${menu}" || fail 'export command not discoverable in TUI'
grep -Fq 'mjust import' "${menu}" || fail 'import command not discoverable in TUI'
grep -Fq 'mjust migration-recover' "${menu}" || fail 'migration recovery command not discoverable in TUI'
grep -Fq 'export destination=""' "${justfile}" || fail 'mjust export recipe missing'
grep -Fq 'import source=""' "${justfile}" || fail 'mjust import recipe missing'
grep -Fq 'migration-recover transaction=""' "${justfile}" || fail 'mjust migration-recover recipe missing'

echo 'migration workflow invariant tests passed.'

# Instance boundaries and pre-destructive staging are enforced by the actual helpers.
for backend in migration-import-backend migration-export-backend admin-migration-import-plan-json admin-migration-export-plan-json admin-migration-import-transaction-json admin-migration-export-transaction-json admin-migration-recovery-json admin-migration-recovery-transaction-json; do
    grep -Fq '== --instance' "${repo_root}/mjust/libexec/${backend}" || fail "missing explicit selection: ${backend}"
    if grep -Eq 'minecraft[.]service|podman exec minecraft ' "${repo_root}/mjust/libexec/${backend}"; then fail "fixed runtime in ${backend}"; fi
done
for forbidden in 'minecraftInstances.register(' 'mode=fresh' 'write_main_config' 'render_runtime'; do
    if grep -Fq "${forbidden}" "${repo_root}/management/cmd/justvoxel-management-agent/admin_migration_import_apply.go" "${api_planner}" "${activation}"; then fail "legacy Import behavior: ${forbidden}"; fi
done
grep -Fq 'sourceInstanceID: $sourceInstanceID' "${exporter}" || fail 'export lacks informational provenance'
if grep -Fq 'sourceInstanceID' "${source_planner}" "${destination_planner}"; then fail 'source provenance controls destination'; fi
grep -Fq 'jv_migration_load_instance "${JV_INSTANCE_ID}"' "${destination_planner}" || fail 'destination is not authoritative'
grep -Fq 'instance_id:$id' "${migration_common}" || fail 'missing durable instance evidence'
grep -Fq '.instance_id==$id' "${migration_common}" || fail 'recovery does not bind evidence to selected instance'
grep -Fq 'if [[ ${minecraft_was_active} != yes ]]; then systemctl stop "${JV_INSTANCE_UNIT}"; fi' "${activation}" || fail 'stopped Import destination remains running'

# Exercise the real rename/rollback implementation with two isolated destinations.
# Runtime commands are mocks; no host services, container engine or live paths run.
(
    fixture="$(mktemp -d)"
    trap 'rm -rf -- "${fixture}"' EXIT
    JV_INSTANCE_ID=jv-aaaaaa
    JV_INSTANCE_UNIT=minecraft-jv-aaaaaa.service
    DATA_PATH="${fixture}/minecraft-instances/${JV_INSTANCE_ID}/data"
    sibling="${fixture}/minecraft-instances/jv-bbbbbb/data"
    mkdir -p "${DATA_PATH}" "${sibling}" "${fixture}/runtime"
    printf old > "${DATA_PATH}/world"
    printf sibling > "${sibling}/world"
    JV_MIGRATION_INSTANCE_CONFIG="${fixture}/runtime/config.json"
    printf environment > "${fixture}/runtime/minecraft.env"
    printf quadlet > "${fixture}/runtime/minecraft-${JV_INSTANCE_ID}.container"
    jq -cn --arg id "${JV_INSTANCE_ID}" --arg path "${DATA_PATH}" \
        --arg env "$(sha256sum "${fixture}/runtime/minecraft.env" | awk '{print $1}')" \
        --arg quadlet "$(sha256sum "${fixture}/runtime/minecraft-${JV_INSTANCE_ID}.container" | awk '{print $1}')" \
        '{runtime:{id:$id},storage:{data_path:$path},environment_sha256:$env,quadlet_sha256:$quadlet}' > "${JV_MIGRATION_INSTANCE_CONFIG}"
    chmod 0600 "${fixture}/runtime/"*
    jv_migration_runtime_paths() { printf '%s\n' "${JV_MIGRATION_INSTANCE_CONFIG}" "${fixture}/runtime/minecraft.env" "${fixture}/runtime/minecraft-${JV_INSTANCE_ID}.container"; }
    jv_migration_load_instance() { [[ $1 == jv-aaaaaa ]]; }
    jv_migration_validate_runtime() {
        if flock -n "${JV_MAINTENANCE_LOCK}" true; then fail 'validated without reacquiring maintenance'; fi
        printf 'validate %s\n' "${JV_INSTANCE_ID}" >> "${fixture}/calls"
    }
    # Unprivileged source checks model only the appliance account mapping.
    # Root test environments exercise the actual inode ownership transitions.
    if (( EUID != 0 )); then
        chown() { [[ $1 == root:root && -f ${3} && ${3} == "${fixture}/"* ]]; }
        stat() {
            case "${2}" in
                %u:%g:%a|%u:%g:%a:%h)
                    command stat -c "${2//%u:%g/0:0}" -- "${@: -1}" ;;
                *) command stat "$@" ;;
            esac
        }
    else
        chown root:root -- "${fixture}/runtime/"*
    fi
    JV_MAINTENANCE_LOCK="${fixture}/maintenance.lock"
    systemctl() {
        if [[ $1 == start ]]; then
            # Independent open description, as used by the real ExecStartPre.
            exec 7>"${JV_MAINTENANCE_LOCK}"
            flock -n 7 || return 1
            [[ ${start_fault:-} == contended ]] || exec 7>&-
            [[ ${start_fault:-} != failed ]] || return 1
        fi
        printf '%s\n' "$*" >> "${fixture}/calls"
        [[ $* == 'daemon-reload' || $* == *"${JV_INSTANCE_UNIT}" ]] || fail 'rollback targeted a sibling unit'
    }
    for minecraft_was_active in yes no; do
        transaction="$(dirname -- "${DATA_PATH}")/.justvoxel-import-${minecraft_was_active}"
        mkdir -m0700 "${transaction}"
        if (( EUID == 0 )); then
            # Simulate files inheriting the service GID regardless of runner GID.
            chmod 0700 "${transaction%/*}"
            chgrp 12345 "${transaction}"
            chmod g+s "${transaction}"
        fi
        mkdir "${transaction}/staged-server"
        printf imported > "${transaction}/staged-server/world"
        source_identity=external-without-justvoxel-id
        jv_migration_snapshot_runtime "${transaction}"
        jv_migration_write_state "${transaction}" switching external paper
        jv_migration_activate_data "${transaction}" "${DATA_PATH}" "${transaction}/staged-server" yes
        [[ $(cat "${DATA_PATH}/world") == imported ]] || fail 'selected data was not activated'
        if (( EUID == 0 )); then
            # Model NSS on an appliance without requiring a host service account.
            service_gid=12345
            getent() { [[ $* == 'group justvoxel-web' ]] && echo 'justvoxel-web:x:12345:'; }
            evidence="${transaction}/evidence.json"
            chown 0:0 -- "${transaction}"
            chmod g-s "${transaction}"
            for file in "${evidence}" "${transaction}/state"; do
                [[ $(stat -c '%u:%g:%a' "${file}") == 0:0:600 ]] || fail 'metadata was not published privately'
                chown "0:${service_gid}" -- "${file}"
            done
            parent="${transaction%/*}"
            snapshot="${transaction}/destination-state"
            portable="${transaction}/portable.json"
            chown 0:0 -- "${parent}"
            chmod 0755 "${parent}"
            chown "0:${service_gid}" -- "${snapshot}"
            chmod =700 "${snapshot}"
            jq -cn --arg id "${JV_INSTANCE_ID}" '{instance_id:$id,game_mode:"survival",difficulty:"easy",
                whitelist:false,enforce_whitelist:false,max_players:20,motd:"fixture",source_software:"vanilla",
                version:"1.21",loader_version:""}' > "${portable}"
            chmod 0600 "${portable}"
            chown "0:${service_gid}" -- "${portable}"
            if jv_migration_validate_evidence "${transaction}"; then fail 'legacy group passed canonical validation'; fi
            mv "${evidence}" "${transaction}/original"
            reject_legacy() {
                if jv_migration_normalize_legacy_evidence "${transaction}"; then fail "accepted $1"; fi
                [[ $(stat -c %g "${transaction}/original") == "${service_gid}" ]] || fail 'changed unrelated inode'
            }
            reject_legacy 'unknown metadata filename'
            ln -s original "${evidence}"
            reject_legacy symlink
            rm "${evidence}"
            mkdir "${evidence}"
            reject_legacy directory
            rmdir "${evidence}"
            cp -p "${transaction}/original" "${evidence}"
            for identity in "1:${service_gid}" '0:12346'; do
                chown "${identity}" "${evidence}"
                reject_legacy 'unexpected owner or group'
            done
            chown "0:${service_gid}" "${evidence}"
            chmod 0640 "${evidence}"
            reject_legacy mode
            chmod 0600 "${evidence}"
            for filter in '.instance_id="jv-bbbbbb"' '.data_path="/wrong"' '.transaction="/wrong"' '.config_identity=("0"*64)'; do
                jq "${filter}" "${transaction}/original" > "${evidence}"
                reject_legacy 'mismatched evidence'
            done
            printf '{' > "${evidence}"
            reject_legacy 'malformed evidence'
            truncate -s 65537 "${evidence}"
            reject_legacy 'oversized evidence'
            cp -p "${transaction}/original" "${evidence}"
            if DATA_PATH="${sibling}" jv_migration_normalize_legacy_evidence "${transaction}"; then fail 'accepted sibling path'; fi
            for mode in 0775 0777; do
                chmod "${mode}" "${parent}"
                reject_legacy "parent mode ${mode}"
            done
            chmod 0755 "${parent}"
            for owner in 1:0 "0:${service_gid}"; do
                chown "${owner}" "${parent}"
                reject_legacy "parent ownership ${owner}"
            done
            chown 0:0 "${parent}"
            chown "0:${service_gid}" "${transaction}"
            reject_legacy 'transaction group'
            chown 0:0 "${transaction}"
            chown 0:12346 "${snapshot}"
            reject_legacy 'snapshot group'
            chown "0:${service_gid}" "${snapshot}"
            for file in "${snapshot}" "${transaction}/state" "${portable}"; do
                mv "${file}" "${file}.saved"
                ln -s "${file}.saved" "${file}"
                reject_legacy "symlink ${file##*/}"
                rm "${file}"
                mv "${file}.saved" "${file}"
            done
            cp -p "${portable}" "${portable}.saved"
            printf '{' > "${portable}"
            reject_legacy 'malformed portable JSON'
            jq '.instance_id="jv-bbbbbb"' "${portable}.saved" > "${portable}"
            reject_legacy 'foreign portable instance'
            cp -p "${portable}.saved" "${portable}"
            for file in "${snapshot}" "${transaction}/state" "${portable}" "${evidence}"; do
                [[ $(stat -c %g "${file}") == "${service_gid}" ]] || fail 'rejected repair changed ownership'
            done
            # Run the real entry points with the fixture runtime and registry.
            helpers="${repo_root}/mjust/libexec"
            recovery_json() (
                source <(sed '/^source /d; /^require_root$/d' "${helpers}/admin-migration-recovery-json") "$@"
            )
            selected=(--instance "${JV_INSTANCE_ID}")
            discovered="$(recovery_json "${selected[@]}" discover)"
            jq -e --arg tx "${transaction}" 'any(.transactions[]; .transaction==$tx and
                .mode!="needs_attention" and .finalizable==true)' <<< "${discovered}" >/dev/null || fail 'legacy discovery unavailable'
            [[ $(stat -c '%u:%g:%a' "${parent}") == 0:0:755 ]] || fail 'legacy parent changed'
            for file in "${transaction}" "${snapshot}"; do
                [[ $(stat -c '%u:%g:%a' "${file}") == 0:0:700 ]] || fail 'incorrect directory metadata'
            done
            for file in "${evidence}" "${transaction}/state" "${portable}"; do
                [[ $(stat -c '%u:%g:%a' "${file}") == 0:0:600 ]] || fail 'incorrect private metadata'
            done
            jv_migration_validate_evidence "${transaction}" || fail 'normalized evidence failed validation'
            jv_migration_normalize_legacy_evidence "${transaction}" || fail 'legacy repair was not idempotent'
            request="$(jq -cn --arg tx "${transaction}" '{transaction:$tx}')"
            plan="$(recovery_json "${selected[@]}" plan <<< "${request}")"
            identity="$(jq -er 'select(.ok and .normalized.finalizable) | .context.state_identity' <<< "${plan}")"
            if [[ ${minecraft_was_active} == yes ]]; then
                apply_request="$(jq -c --arg id "${identity}" '. + {state_identity:$id}' <<< "${request}")"
                recovery_apply() (
                    source <(sed -e '/^source /d; /^require_root$/d' \
                        -e 's|/usr/libexec/justvoxel/mjust/admin-migration-recovery-json |recovery_json |' \
                        -e 's|${JV_MIGRATION_RECOVERY_ROOT}|${fixture}/registry|g' \
                        "${helpers}/admin-migration-recovery-transaction-json") "${selected[@]}"
                )
                # Keep every registry operation under the temporary fixture.
                jv_migration_clear_recovery() { rm -f -- "${fixture}/registry/$(jv_migration_recovery_key "$1")"; }
                mkdir "${fixture}/registry"
                touch "${fixture}/registry/$(jv_migration_recovery_key "${transaction}")"
                printf changed > "${fixture}/runtime/minecraft.env"
                set +e
                for start_fault in failed contended; do
                    (recovery_apply <<< "${apply_request}")
                    failed_status=$?
                    [[ ${failed_status} != 0 && -f ${transaction}/evidence.json ]] || fail "lost recovery on ${start_fault} start"
                done
                unset start_fault
                (jv_migration_validate_runtime() { return 1; }; recovery_apply <<< "${apply_request}")
                failed_status=$?
                set -e
                [[ ${failed_status} != 0 && -d ${transaction} ]] || fail 'failed validation lost recovery'
                cmp "${snapshot}/minecraft.env" "${fixture}/runtime/minecraft.env" || fail 'runtime was not restored'
                recovery_apply <<< "${apply_request}"
                [[ ! -e ${transaction} && $(cat "${DATA_PATH}/world") == old &&
                   $(cat "${sibling}/world") == sibling ]] || fail 'selected recovery did not finalize'
                continue
            fi
        fi
        exec 9>"${JV_MAINTENANCE_LOCK}"
        flock -n 9 || fail 'could not take initial maintenance lock'
        if flock -n "${JV_MAINTENANCE_LOCK}" true; then fail 'fixture lacks initial exclusion'; fi
        jv_migration_recover_selected "${transaction}" || fail 'selected rollback failed'
        if flock -n "${JV_MAINTENANCE_LOCK}" true; then fail 'rollback returned without maintenance'; fi
        [[ $(cat "${DATA_PATH}/world") == old && $(cat "${sibling}/world") == sibling ]] || fail 'rollback changed sibling data'
        [[ $(cat "${transaction}/failed-import/world") == imported ]] || fail 'failed imported data was lost'
        if [[ ${minecraft_was_active} == no ]]; then
            [[ $(tail -n1 "${fixture}/calls") == "stop ${JV_INSTANCE_UNIT}" ]] || fail 'stopped destination was left running'
        fi
        # Repeating recovery after a crash following the restore rename is safe.
        jv_migration_recover_selected "${transaction}" || fail 'rollback is not restart-safe'
        cp "${transaction}/evidence.json" "${transaction}/valid-evidence"
        jq '.instance_id="jv-bbbbbb"' "${transaction}/valid-evidence" > "${transaction}/evidence.json"
        before="$(wc -l < "${fixture}/calls")"
        if jv_migration_recover_selected "${transaction}"; then fail 'foreign recovery identity accepted'; fi
        [[ $(wc -l < "${fixture}/calls") == "${before}" ]] || fail 'foreign recovery touched runtime'
        jq 'del(.instance_id)' "${transaction}/valid-evidence" > "${transaction}/evidence.json"
        if jv_migration_recover_selected "${transaction}"; then fail 'old recovery evidence guessed a destination'; fi
        exec 9>&-
    done
)
