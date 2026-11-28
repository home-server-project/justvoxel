#!/usr/bin/bash
set -euo pipefail

readonly JV_CONFIG_DIR=/etc/justvoxel
readonly JV_CONFIG=${JV_CONFIG_DIR}/justvoxel.conf
readonly JV_MC_ENV=${JV_CONFIG_DIR}/minecraft.env
readonly JV_BACKUP_ENV=${JV_CONFIG_DIR}/minecraft-backup.env
readonly JV_TEMPLATE_ROOT=/usr/share/justvoxel/templates
readonly JV_QUADLET=/etc/containers/systemd/minecraft.container
readonly JV_BACKUP_SERVICE=/etc/systemd/system/minecraft-backup.service
readonly JV_BACKUP_TIMER=/etc/systemd/system/minecraft-backup.timer
readonly JV_MAINTENANCE_LOCK=/run/justvoxel-minecraft-maintenance.lock
readonly JV_STATE_DIR=/var/lib/justvoxel/state
readonly JV_SETUP_IN_PROGRESS=/var/lib/justvoxel/management/setup-in-progress
readonly JV_PREVIOUS_IMAGE_STATE=${JV_STATE_DIR}/minecraft-previous-image-id
readonly JV_MINECRAFT_IMAGE_REPO=docker.io/itzg/minecraft-server
readonly JV_ITZG_IMAGES_URL=https://raw.githubusercontent.com/itzg/docker-minecraft-server/refs/heads/master/images.json
readonly JV_GEYSER_VERSIONS_URL=https://raw.githubusercontent.com/GeyserMC/GeyserWebsite/refs/heads/master/src/data/versions.json
readonly JV_PAPER_USER_AGENT='JustVoxel/0.1 (https://github.com/home-server-project/justvoxel)'

# Resolve one identity for an exact mount point. Repeated identical rows are
# harmless; conflicting rows or a missing identity must never pass validation.
jv_exact_mount_identity() {
    local field="$1" mountpoint="$2" target value identity='' saw_target=false
    case "${field}" in UUID|SOURCE) ;; *) return 1 ;; esac
    while IFS= read -r target; do
        [[ ${target} == "${mountpoint}" ]] || return 1
        saw_target=true
    done < <(findmnt -rn -o TARGET --mountpoint "${mountpoint}" 2>/dev/null)
    [[ ${saw_target} == true ]] || return 1
    while IFS= read -r value; do
        [[ -n ${value} ]] || return 1
        if [[ -n ${identity} && ${identity} != "${value}" ]]; then
            return 1
        fi
        identity="${value}"
    done < <(findmnt -rn -o "${field}" --mountpoint "${mountpoint}" 2>/dev/null)
    [[ -n ${identity} ]] || return 1
    printf '%s\n' "${identity}"
}

jv_variant_raw() {
    cat /usr/lib/justvoxel/variant 2>/dev/null || printf 'unknown\n'
}

jv_variant_kind() {
    local raw="${1:-}"
    [[ -n ${raw} ]] || raw="$(jv_variant_raw)"
    case "${raw}" in
        dev|justvoxel-dev) printf 'dev\n' ;;
        vm|justvoxel-vm) printf 'vm\n' ;;
        hws|justvoxel-hws) printf 'hws\n' ;;
        *) printf 'unknown\n' ;;
    esac
}

jv_variant_name() {
    case "$(jv_variant_kind "${1:-}")" in
        dev) printf 'Development\n' ;;
        vm) printf 'VM\n' ;;
        hws) printf 'HWS\n' ;;
        *) printf 'Unknown\n' ;;
    esac
}

jv_variant_is_vm() {
    [[ $(jv_variant_kind "${1:-}") == vm ]]
}

jv_variant_is_hws() {
    [[ $(jv_variant_kind "${1:-}") == hws ]]
}

require_root() {
    if [[ ${EUID} -ne 0 ]]; then
        echo 'ERROR: this operation must run as root.' >&2
        exit 1
    fi
}

require_config() {
    if [[ ! -r ${JV_CONFIG} ]]; then
        echo 'ERROR: JustVoxel has not been configured yet. Run: mjust setup' >&2
        exit 1
    fi
    # shellcheck disable=SC1090
    MINECRAFT_SERVER_TYPE=paper
    source "${JV_CONFIG}"
    # Legacy readers may still need a timezone value, always from the appliance.
    TIMEZONE="$(timedatectl show --property=Timezone --value 2>/dev/null || true)"
    TIMEZONE="${TIMEZONE:-UTC}"

    # Compatibility defaults for configurations created before image/version
    # policy and portable Minecraft settings became administrator-configurable.
    MINECRAFT_SERVER_TYPE="${MINECRAFT_SERVER_TYPE-paper}"
    validate_server_capabilities || return 1
    MINECRAFT_IMAGE_TAG="${MINECRAFT_IMAGE_TAG:-latest}"
    GAME_MODE="${GAME_MODE:-survival}"
    DIFFICULTY="${DIFFICULTY:-normal}"
    WHITELIST_ENABLED="${WHITELIST_ENABLED:-yes}"
    ENFORCE_WHITELIST="${ENFORCE_WHITELIST:-yes}"
    BEDROCK_MANAGED_PLUGINS="${BEDROCK_MANAGED_PLUGINS:-yes}"
    if [[ -z ${MINECRAFT_VERSION_MODE:-} ]]; then
        if [[ ${MINECRAFT_VERSION:-} == LATEST ]]; then
            MINECRAFT_VERSION_MODE=latest
        else
            MINECRAFT_VERSION_MODE=pinned
        fi
    fi
}

prompt_default() {
    local prompt="$1" default="$2" value
    read -r -p "${prompt} [${default}]: " value
    printf '%s' "${value:-${default}}"
}

confirm() {
    local prompt="$1" answer
    read -r -p "${prompt} [y/N]: " answer
    [[ ${answer,,} == y || ${answer,,} == yes ]]
}

# Menu/help text is deliberately written to stderr. Many callers use command
# substitution and stdout must contain only the selected value.
choose() {
    local prompt="$1"
    shift
    local options=("$@") choice index
    printf '%s\n' "${prompt}" >&2
    for index in "${!options[@]}"; do
        printf '  %d. %s\n' "$((index + 1))" "${options[index]}" >&2
    done
    while true; do
        read -r -p 'Selection: ' choice
        if [[ ${choice} =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#options[@]} )); then
            printf '%s' "${choice}"
            return 0
        fi
        echo 'Please enter a valid number.' >&2
    done
}

# Defaults are opt-in and limited to menus where pressing Enter is safe.
# Destructive/storage menus continue to use choose(), which has no default.
choose_default() {
    local prompt="$1" default="$2"
    shift 2
    local options=("$@") choice index
    [[ ${default} =~ ^[0-9]+$ ]] && (( default >= 1 && default <= ${#options[@]} )) || {
        echo 'ERROR: invalid menu default.' >&2
        return 2
    }
    printf '%s\n' "${prompt}" >&2
    for index in "${!options[@]}"; do
        printf '  %d. %s\n' "$((index + 1))" "${options[index]}" >&2
    done
    while true; do
        read -r -p "Selection [${default}]: " choice
        choice="${choice:-${default}}"
        if [[ ${choice} =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#options[@]} )); then
            printf '%s' "${choice}"
            return 0
        fi
        echo 'Please enter a valid number.' >&2
    done
}

validate_port() {
    local port="$1"
    [[ ${port} =~ ^[0-9]+$ ]] && (( port >= 1 && port <= 65535 ))
}

validate_memory() {
    [[ $1 =~ ^[1-9][0-9]*[MmGg]$ ]]
}

memory_to_mib() {
    local value="${1^^}" number="${1%?}"
    if [[ ${value} == *G ]]; then
        printf '%d' $((number * 1024))
    else
        printf '%d' "${number}"
    fi
}

suggest_memory_values() {
    local total_mib nominal_gib heap_mib maximum_mib allowed_mib
    total_mib=$(( $(awk '/^MemTotal:/ {print $2}' /proc/meminfo 2>/dev/null || echo 0) / 1024 ))
    # Same Recommended policy as settings.js. Round only the sizing class;
    # reserve checks always use actual MemTotal, never swap/zram.
    nominal_gib=$(( (total_mib + 512) / 1024 ))
    (( total_mib > 0 )) || nominal_gib=8
    if (( nominal_gib >= 16 )); then
        heap_mib=6144 maximum_mib=8192
    elif (( nominal_gib >= 8 )); then
        heap_mib=4096 maximum_mib=6144
    elif (( nominal_gib >= 6 )); then
        heap_mib=3072 maximum_mib=4096
    elif (( nominal_gib >= 4 )); then
        heap_mib=2048 maximum_mib=3072
    else
        heap_mib=1024 maximum_mib=2048
    fi
    if (( total_mib > 0 )); then
        allowed_mib=$(( total_mib - 1024 ))
        (( allowed_mib >= 1024 )) || return 1
        # Preserve the tier heap when a MiB-precision container limit fits.
        (( maximum_mib <= allowed_mib )) || maximum_mib=${allowed_mib}
        if (( heap_mib > maximum_mib - 512 )); then
            heap_mib=$(( maximum_mib >= 2048 ? maximum_mib - 1024 : maximum_mib / 2 ))
        fi
    fi
    local heap maximum
    if (( heap_mib % 1024 == 0 )); then heap="$((heap_mib / 1024))G"; else heap="${heap_mib}M"; fi
    if (( maximum_mib % 1024 == 0 )); then maximum="$((maximum_mib / 1024))G"; else maximum="${maximum_mib}M"; fi
    printf '%s %s\n' "${heap}" "${maximum}"
}

normalize_daily_backup_time() {
    local value="$1" hour minute
    [[ ${value} =~ ^([0-9]{1,2}):([0-9]{2})$ ]] || return 1
    hour="${BASH_REMATCH[1]}"
    minute="${BASH_REMATCH[2]}"
    (( 10#${hour} <= 23 && 10#${minute} <= 59 )) || return 1
    printf '%02d:%02d' "$((10#${hour}))" "$((10#${minute}))"
}

daily_backup_schedule_from_time() {
    local normalized
    normalized="$(normalize_daily_backup_time "$1")" || return 1
    printf '*-*-* %s:00' "${normalized}"
}

validate_positive_int() {
    [[ $1 =~ ^[1-9][0-9]*$ ]]
}

validate_nonroot_id() {
    local value="${1:-}"
    [[ ${value} =~ ^[0-9]+$ ]] || return 1
    (( 10#${value} > 0 && 10#${value} <= 4294967294 ))
}

resolve_minecraft_ids() {
    local uid gid

    if validate_nonroot_id "${SUDO_UID:-}" && validate_nonroot_id "${SUDO_GID:-}"; then
        printf '%s %s sudo\n' "${SUDO_UID}" "${SUDO_GID}"
        return 0
    fi

    if id voxel >/dev/null 2>&1; then
        uid="$(id -u voxel)"
        gid="$(id -g voxel)"
        if validate_nonroot_id "${uid}" && validate_nonroot_id "${gid}"; then
            printf '%s %s voxel\n' "${uid}" "${gid}"
            return 0
        fi
    fi

    printf '1000 1000 fallback\n'
}

validate_storage_path() {
    [[ $1 =~ ^/[A-Za-z0-9._/-]+$ && $1 != / ]]
}

validate_simple_text() {
    local value="$1"
    [[ ${value} != *$'\n'* && ${value} != *$'\r'* ]]
}

validate_game_mode() {
    case "${1:-}" in
        survival|creative|adventure|spectator) return 0 ;;
        *) return 1 ;;
    esac
}

shell_quote_assignment() {
    local key="$1" value="$2"
    printf '%s=' "${key}"
    printf '%q' "${value}"
    printf '\n'
}

minecraft_image_ref() {
    printf '%s:%s' "${JV_MINECRAFT_IMAGE_REPO}" "${MINECRAFT_IMAGE_TAG}"
}

validate_minecraft_image_tag() {
    local tag="$1" catalog
    [[ ${tag} =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]] || return 1

    skopeo inspect "docker://${JV_MINECRAFT_IMAGE_REPO}:${tag}" >/dev/null 2>&1 || return 1

    # Upstream publishes a machine-readable list of moving Java/image tags.
    # Reject entries upstream explicitly marks deprecated. Exact release tags are
    # not in this catalog and are accepted when the registry confirms they exist.
    catalog="$(curl -fsSL "${JV_ITZG_IMAGES_URL}" 2>/dev/null || true)"
    if [[ -n ${catalog} ]] && jq -e --arg tag "${tag}" \
        '.[] | select(.tag == $tag and (.deprecated // false) == true)' \
        <<< "${catalog}" >/dev/null 2>&1; then
        return 2
    fi
    return 0
}

resolve_latest_itzg_release() {
    curl -fsSL -H "User-Agent: ${JV_PAPER_USER_AGENT}" \
        https://api.github.com/repos/itzg/docker-minecraft-server/releases/latest 2>/dev/null \
        | jq -r '.tag_name // empty'
}

write_main_config() {
    MINECRAFT_SERVER_TYPE="${MINECRAFT_SERVER_TYPE-paper}"
    validate_server_capabilities || return 1
    install -d -m0700 -o root -g root "${JV_CONFIG_DIR}"
    local tmp
    tmp="$(mktemp "${JV_CONFIG_DIR}/.justvoxel.conf.XXXXXX")"
    chmod 0600 "${tmp}"
    {
        echo '# JustVoxel administrator-owned configuration.'
        echo '# Generated by mjust. bootc image updates do not overwrite this file.'
        shell_quote_assignment DATA_PATH "${DATA_PATH}"
        shell_quote_assignment DATA_MOUNT_POINT "${DATA_MOUNT_POINT}"
        shell_quote_assignment DATA_EXPECTED_UUID "${DATA_EXPECTED_UUID}"
        shell_quote_assignment DATA_EXPECTED_SOURCE "${DATA_EXPECTED_SOURCE}"
        shell_quote_assignment BACKUP_TYPE "${BACKUP_TYPE}"
        shell_quote_assignment BACKUP_PATH "${BACKUP_PATH}"
        shell_quote_assignment BACKUP_MOUNT_POINT "${BACKUP_MOUNT_POINT}"
        shell_quote_assignment BACKUP_EXPECTED_UUID "${BACKUP_EXPECTED_UUID}"
        shell_quote_assignment BACKUP_EXPECTED_SOURCE "${BACKUP_EXPECTED_SOURCE}"
        shell_quote_assignment BACKUP_KEEP "${BACKUP_KEEP}"
        shell_quote_assignment BACKUP_SCHEDULE "${BACKUP_SCHEDULE}"
        shell_quote_assignment BACKUP_TIMER_ENABLED "${BACKUP_TIMER_ENABLED}"
        shell_quote_assignment JAVA_PORT "${JAVA_PORT}"
        shell_quote_assignment BEDROCK_ENABLED "${BEDROCK_ENABLED}"
        shell_quote_assignment BEDROCK_PORT "${BEDROCK_PORT}"
        shell_quote_assignment BEDROCK_MANAGED_PLUGINS "${BEDROCK_MANAGED_PLUGINS:-yes}"
        shell_quote_assignment JAVA_MEMORY "${JAVA_MEMORY}"
        shell_quote_assignment CONTAINER_MEMORY "${CONTAINER_MEMORY}"
        shell_quote_assignment MINECRAFT_SERVER_TYPE "${MINECRAFT_SERVER_TYPE}"
        shell_quote_assignment MINECRAFT_IMAGE_TAG "${MINECRAFT_IMAGE_TAG}"
        shell_quote_assignment MINECRAFT_VERSION_MODE "${MINECRAFT_VERSION_MODE}"
        shell_quote_assignment MINECRAFT_VERSION "${MINECRAFT_VERSION}"
        shell_quote_assignment MINECRAFT_UID "${MINECRAFT_UID}"
        shell_quote_assignment MINECRAFT_GID "${MINECRAFT_GID}"
        shell_quote_assignment GAME_MODE "${GAME_MODE:-survival}"
        shell_quote_assignment DIFFICULTY "${DIFFICULTY:-normal}"
        shell_quote_assignment WHITELIST_ENABLED "${WHITELIST_ENABLED:-yes}"
        shell_quote_assignment ENFORCE_WHITELIST "${ENFORCE_WHITELIST:-yes}"
        shell_quote_assignment MAX_PLAYERS "${MAX_PLAYERS}"
        shell_quote_assignment MOTD "${MOTD}"
    } > "${tmp}"
    mv -f "${tmp}" "${JV_CONFIG}"
    chown root:root "${JV_CONFIG}"
    chmod 0600 "${JV_CONFIG}"
}

sed_replacement_escape() {
    local value="$1"
    value=${value//\\/\\\\}
    value=${value//&/\\&}
    value=${value//|/\\|}
    printf '%s' "${value}"
}

replace_token() {
    local file="$1" token="$2" value="$3" escaped
    escaped="$(sed_replacement_escape "${value}")"
    sed -i "s|@@${token}@@|${escaped}|g" "${file}"
}

systemd_env_escape() {
    local value="$1"
    value=${value//\\/\\\\}
    value=${value//\"/\\\"}
    printf '%s' "${value}"
}

path_regex_escape() {
    printf '%s' "$1" | sed 's/[][\\.^$*+?(){}|]/\\&/g'
}

apply_data_selinux() {
    local escaped
    escaped="$(path_regex_escape "${DATA_PATH}")"
    if ! semanage fcontext -a -t container_file_t "${escaped}(/.*)?" 2>/dev/null; then
        semanage fcontext -m -t container_file_t "${escaped}(/.*)?"
    fi
    restorecon -RF "${DATA_PATH}"
}

capture_backup_mount_identity() {
    BACKUP_EXPECTED_UUID=''
    BACKUP_EXPECTED_SOURCE=''
    if [[ -z ${BACKUP_MOUNT_POINT} ]]; then
        return 0
    fi
    if ! mountpoint -q -- "${BACKUP_MOUNT_POINT}"; then
        echo "ERROR: selected backup mount is not mounted: ${BACKUP_MOUNT_POINT}" >&2
        return 1
    fi
    BACKUP_EXPECTED_SOURCE="$(jv_exact_mount_identity SOURCE "${BACKUP_MOUNT_POINT}" 2>/dev/null || true)"
    if [[ ${BACKUP_TYPE} == disk || ${BACKUP_TYPE} == partition ]]; then
        BACKUP_EXPECTED_UUID="$(jv_exact_mount_identity UUID "${BACKUP_MOUNT_POINT}" 2>/dev/null || true)"
        if [[ -z ${BACKUP_EXPECTED_UUID} ]]; then
            echo 'ERROR: could not determine a filesystem UUID for the selected local backup mount.' >&2
            return 1
        fi
    fi
}

geyser_supported_java_version_from_json() {
    local metadata="$1" version
    version="$(jq -ers 'select(length == 1) | .[0].java.supported | select(type == "string")' <<< "${metadata}" 2>/dev/null)" || return 6
    [[ ${version} =~ ^[0-9]+([.][0-9]+){1,2}$ ]] || return 6
    printf '%s' "${version}"
}

resolve_geyser_supported_java_version() {
    local metadata status
    metadata="$(curl --connect-timeout 3 --max-time 8 -fsSL -H "User-Agent: ${JV_PAPER_USER_AGENT}" "${JV_GEYSER_VERSIONS_URL}" 2>/dev/null)" || {
        status=$?
        if (( status == 28 )); then return 4; fi
        return 5
    }
    geyser_supported_java_version_from_json "${metadata}"
}

bedrock_crossplay_supports_version() {
    local minecraft_version="$1" geyser_java_version="$2"
    [[ -n ${minecraft_version} && -n ${geyser_java_version} && ${minecraft_version} == "${geyser_java_version}" ]]
}

jv_crossplay_viaversion_supports() {
    local game="$1" geyser="$2" metadata result status project=P1OZGk5p mode="${3:-compatible}"
    [[ ${game} =~ ^[0-9]+[.][0-9]+([.][0-9]+)?$ && ${geyser} =~ ^[0-9]+[.][0-9]+([.][0-9]+)?$ ]] || return 1
    metadata="$(curl --connect-timeout 3 --max-time 6 --max-filesize 2097152 -fsS --get \
        -H "User-Agent: ${JV_PAPER_USER_AGENT}" \
        --data-urlencode 'loaders=["paper"]' --data-urlencode "game_versions=[\"${game}\"]" \
        --data-urlencode 'include_changelog=false' \
        "https://api.modrinth.com/v2/project/${project}/version" 2>/dev/null)" || {
        status=$?
        if (( status == 28 )); then return 4; fi
        return 5
    }
    result="$(jq -ers --arg project "${project}" --arg game "${game}" --arg geyser "${geyser}" --arg mode "${mode}" '
        def compatible_release:
            .status == "listed" and .version_type == "release" and
            (.game_versions | index($game) != null and index($geyser) != null) and
            (.loaders | index("paper") != null);
        def primary_file:
            type == "object" and .primary == true and
            (.filename | type == "string" and endswith(".jar")) and
            (.size | type == "number" and . > 0) and
            (.url | type == "string" and startswith("https://cdn.modrinth.com/data/" + $project + "/"));
        def strings: type == "array" and all(.[]; type == "string");
        select(length == 1) | .[0] | select(type == "array" and
            all(.[]; type == "object" and .project_id == $project and
                (.status | type == "string") and (.version_type | type == "string") and
                (.game_versions | strings) and (.loaders | strings) and (.files | type == "array"))) |
        if $mode == "artifact" then
            map(select(compatible_release and
                (.id | type == "string" and test("^[A-Za-z0-9]{8}$")) and
                (.date_published | type == "string") and
                (.dependencies | type == "array" and all(.[]; .dependency_type == "optional" or .dependency_type == "embedded")))) |
            sort_by(.date_published, .id) | reverse |
            map(.id as $id | [.files[] | select(primary_file)] | select(length == 1) | .[0] |
                select(.filename | test("^[A-Za-z0-9._+-]{1,195}[.]jar$")) |
                select(.url == ("https://cdn.modrinth.com/data/" + $project + "/versions/" + $id + "/" + .filename)) |
                select(.size <= 67108864 and (.hashes.sha512 | type == "string" and test("^[a-f0-9]{128}$"))) |
                {project:"viaversion", url, sha512:.hashes.sha512}) | .[0] // empty | @json
        else
            if any(.[]; compatible_release and any(.files[]; primary_file)) then "verified"
            elif any(.[]; compatible_release) then "artifact_unverified"
            else "unsupported" end
        end
    ' <<< "${metadata}" 2>/dev/null)" || return 6
    if [[ ${mode} == artifact ]]; then
        printf '%s\n' "${result}"
    else
        case "${result}" in
            verified) return 0 ;;
            unsupported) return 8 ;;
            *) return 7 ;;
        esac
    fi
}

# Verify an exact runtime against the current Geyser Java target.
jv_managed_crossplay_compatible() {
    jv_managed_crossplay_check "$@" >/dev/null
}

# The matrix records only fixed failure codes from the same compatibility check.
jv_managed_crossplay_check() {
    local software="$1" game="$2" loader="$3" geyser="$4" reason status
    [[ -n ${software} && ${game} =~ ^[0-9]+([.][0-9]+){1,2}$ && ${geyser} =~ ^[0-9]+([.][0-9]+){1,2}$ ]] &&
        server_supports_managed_crossplay "${software}" || { printf crossplay_selection_invalid; return 1; }
    if server_supports_viaversion "${software}"; then
        bedrock_crossplay_supports_version "${game}" "${geyser}" && return 0
        if [[ $(printf '%s\n' "${game}" "${geyser}" | sort -V | tail -n1) != "${geyser}" ]]; then
            reason=geyser_java_target_mismatch
        elif jv_crossplay_viaversion_supports "${game}" "${geyser}"; then
            return 0
        else
            status=$?
            case ${status} in
                4) reason=viaversion_request_timeout ;;
                5) reason=viaversion_request_failed ;;
                6) reason=viaversion_response_invalid ;;
                8) reason=viaversion_exact_release_unsupported ;;
                *) reason=viaversion_release_unverified ;;
            esac
        fi
    elif ! bedrock_crossplay_supports_version "${game}" "${geyser}"; then
        reason=geyser_java_target_mismatch
    elif [[ -z ${loader} ]]; then
        reason=crossplay_loader_missing
    elif jv_crossplay_mod_compatible "${software}" "${game}" "${loader}" >/dev/null; then
        return 0
    else
        status=$?
        reason=crossplay_mod_artifacts_unverified
        [[ ${status} != 124 ]] || reason=crossplay_mod_request_timeout
    fi
    printf '%s' "${reason}"
    return 1
}

paper_version_has_stable_build() {
    local version="$1" builds
    builds="$(curl -fsSL -H "User-Agent: ${JV_PAPER_USER_AGENT}" \
        "https://fill.papermc.io/v3/projects/paper/versions/${version}/builds" 2>/dev/null)" || return 2
    if jq -e '.ok == false' <<< "${builds}" >/dev/null 2>&1; then
        return 1
    fi
    jq -e 'any(.[]; .channel == "STABLE" and (.downloads."server:default".url // "") != "")' <<< "${builds}" >/dev/null
}

# Exit 3 requires authoritative absence. Transport errors are unverified.
java_exact_publisher_response() {
    local response status attempt attempts=1
    if [[ ${classify_exact_release:-0} != 1 ]]; then
        curl "$@"
        return
    fi
    [[ ${java_retry_catalog_transport:-0} != 1 ]] || attempts=2
    for ((attempt=1; attempt<=attempts; attempt++)); do
        if response="$(curl --write-out $'\n%{http_code}' --max-redirs 3 --proto '=https' --proto-redir '=https' --connect-timeout 5 --max-time 20 "$@")"; then
            [[ ${response##*$'\n'} == 200 ]] || return 5
            printf '%s\n' "${response%$'\n'*}"
            return
        else
            status=$?
        fi
        [[ ${status} == 22 && ${response##*$'\n'} == 404 ]] && return 3
        # NeoForge catalog reads may retry timeout and TLS handshake failures.
        # Certificate verification, HTTP and malformed-body failures never retry.
        case "${status}" in 28|35) ;; *) return 5 ;; esac
    done
    if (( status == 28 )); then return 4; fi
    return 5
}

java_publisher_catalog_response() {
    local response status
    response="$(java_exact_publisher_response "$@")" || {
        status=$?
        # A missing catalog says nothing about any exact release.
        if (( status == 3 )); then return 5; fi
        return "${status}"
    }
    printf '%s\n' "${response}"
}

java_release_catalog_member() {
    local catalog result
    catalog="$(jq -ces 'select(length == 1) | .[0]')" || return 6
    result="$(jq -er "$@" <<< "${catalog}")" || return 6
    [[ ${result} == true ]] && return 0
    # jq -e returns 1 on false, so the queries emit these as strings.
    [[ ${result} == false ]] && return 3
    return 6
}

# Print the release channel of a downloadable Paper server build.
paper_version_build_channel() {
    local version="$1" builds
    [[ ${version} =~ ^[0-9A-Za-z._-]+$ ]] || return 1
    builds="$(java_exact_publisher_response -fsSL -H "User-Agent: ${JV_PAPER_USER_AGENT}" \
        "https://fill.papermc.io/v3/projects/paper/versions/${version}/builds" 2>/dev/null)" || return "$?"
    if [[ ${classify_exact_release:-0} == 1 ]]; then
        builds="$(jq -ces '
            select(length == 1) | .[0] |
            select(type == "array" and all(.[]; type == "object" and
                (.channel | type == "string" and length > 0) and
                (.downloads | type == "object") and
                (if .downloads | has("server:default") then
                    .downloads."server:default".url | type == "string" and startswith("https://")
                 else true end)))' <<< "${builds}")" || return 6
        [[ ${builds} != '[]' ]] || return 3
    fi
    jq -er '[.[] | select((.downloads."server:default".url // "") != "")]
        | (map(select(.channel == "STABLE")) | .[0]) // .[0]
        | .channel | select(type == "string" and length > 0)' <<< "${builds}" 2>/dev/null || return 7
}

resolve_latest_available_paper_version() {
    local project versions version
    project="$(curl -fsSL -H "User-Agent: ${JV_PAPER_USER_AGENT}" https://fill.papermc.io/v3/projects/paper 2>/dev/null)" || return 1
    versions="$(jq -r '.versions[][]' <<< "${project}")"
    while IFS= read -r version; do
        [[ -n ${version} ]] || continue
        if paper_version_build_channel "${version}" >/dev/null; then
            printf '%s' "${version}"
            return 0
        fi
    done <<< "${versions}"
    return 1
}

resolve_latest_stable_paper_version() {
    local project versions version builds
    project="$(curl -fsSL -H "User-Agent: ${JV_PAPER_USER_AGENT}" https://fill.papermc.io/v3/projects/paper 2>/dev/null)" || return 1
    versions="$(jq -r '.versions[][]' <<< "${project}")"
    while IFS= read -r version; do
        [[ -n ${version} ]] || continue
        builds="$(curl -fsSL -H "User-Agent: ${JV_PAPER_USER_AGENT}" \
            "https://fill.papermc.io/v3/projects/paper/versions/${version}/builds" 2>/dev/null)" || continue
        if jq -e 'any(.[]; .channel == "STABLE" and (.downloads."server:default".url // "") != "")' <<< "${builds}" >/dev/null; then
            printf '%s' "${version}"
            return 0
        fi
    done <<< "${versions}"
    return 1
}

# Fixed Java implementations, with Paper compatibility for pre-existing configs.
validate_server_type() {
    case "$1" in paper|purpur|pufferfish|leaf|vanilla|spigot|forge|neoforge|fabric|quilt|bedrock) return 0 ;; *) return 1 ;; esac
}
# Preserve the deployed legacy migration/export integration boundary.
server_supports_plugins() { [[ ${1:-${MINECRAFT_SERVER_TYPE-paper}} == paper || ${1:-${MINECRAFT_SERVER_TYPE-paper}} == purpur ]]; }
server_supports_viaversion() {
    case "${1:-${MINECRAFT_SERVER_TYPE-paper}}" in paper|purpur|pufferfish|leaf|spigot) return 0 ;; *) return 1 ;; esac
}
server_supports_managed_crossplay() {
    case "${1:-${MINECRAFT_SERVER_TYPE-paper}}" in paper|purpur|pufferfish|leaf|spigot|fabric|neoforge) return 0 ;; *) return 1 ;; esac
}
server_supports_bukkit_plugins() {
    case "${1:-${MINECRAFT_SERVER_TYPE-paper}}" in paper|purpur|pufferfish|leaf|spigot) return 0 ;; *) return 1 ;; esac
}
server_is_mod_loader() {
    case "${1:-${MINECRAFT_SERVER_TYPE-paper}}" in forge|neoforge|fabric|quilt) return 0 ;; *) return 1 ;; esac
}
server_allows_in_place_switch() {
    validate_server_type "$1" && validate_server_type "$2" || return 1
    [[ $1 == "$2" ]] || { [[ $1 == paper || $1 == purpur || $1 == pufferfish || $1 == leaf ]] && [[ $2 == paper || $2 == purpur || $2 == pufferfish || $2 == leaf ]]; }
}
server_software_display_name() {
    case "${1:-${MINECRAFT_SERVER_TYPE-paper}}" in paper) printf Paper ;; purpur) printf Purpur ;; pufferfish) printf Pufferfish ;; leaf) printf Leaf ;; vanilla) printf Vanilla ;; spigot) printf Spigot ;; forge) printf Forge ;; neoforge) printf NeoForge ;; fabric) printf Fabric ;; quilt) printf Quilt ;; *) return 1 ;; esac
}
validate_server_capabilities() {
    validate_server_type "${MINECRAFT_SERVER_TYPE-paper}" || { echo 'ERROR: invalid Minecraft server type.' >&2; return 1; }
    if [[ ${BEDROCK_ENABLED:-no} == yes ]] && ! server_supports_managed_crossplay; then
        echo 'ERROR: managed Bedrock cross-play requires Paper, Purpur or Spigot.' >&2
        return 1
    fi
}
# Explicit pins use the Java generation for the game release. Availability of
# the exact game/loader pair must still be verified against upstream metadata.
validate_java_server_selection() {
    local software="$1" policy="$2" version="$3" loader="$4" image="$5" required_image
    validate_server_type "${software}" || return 1
    case "${software}" in
        paper|purpur|pufferfish|leaf|vanilla) [[ -z ${loader} ]]; return ;;
    esac
    [[ ${policy} == pinned ]] || return 1
    if [[ ${version} =~ ^1\.(20\.[56]|21(\.([0-9]|1[01]))?)$ ]]; then
        required_image=java21
    elif [[ ${version} =~ ^26\.[1-9][0-9]*(\.(0|[1-9][0-9]*))?$ ]]; then
        required_image=java25
    else
        return 1
    fi
    if [[ ${software} == spigot ]]; then
        [[ -z ${loader} && ${image} == "${required_image}-jdk" ]]
    else
        [[ ${image} == "${required_image}" && ${#loader} -le 64 && ${loader} =~ ^[0-9]+(\.[0-9]+){1,3}(-[A-Za-z0-9]+([.-][A-Za-z0-9]+)*)?$ ]]
    fi
}

pufferfish_job_branches() {
    local catalog
    catalog="$(java_publisher_catalog_response --connect-timeout 3 --max-time 12 -fsSL -H "User-Agent: ${JV_PAPER_USER_AGENT}" \
        'https://ci.pufferfish.host/api/json?tree=jobs%5Bname%5D')" || return "$?"
    jq -ces --arg exact "${classify_exact_release:-0}" '
        select(length == 1) | .[0].jobs | select(type == "array") |
        select($exact != "1" or all(.[]; .name | type == "string" and length > 0)) |
        [.[] | .name | strings | select(test("^Pufferfish-[0-9]+[.][0-9]+$")) |
            ltrimstr("Pufferfish-")] | unique |
        sort_by(split(".") | map(tonumber)) | reverse' <<< "${catalog}" || return 6
}

# A missing latest-build alias alone says nothing about historical artifacts.
# A validated catalog that lacks the branch can corroborate a branch 404.
pufferfish_missing_branch() {
    local branch="$1" branches
    branches="$(pufferfish_job_branches)" || return "$?"
    if jq -e --arg branch "${branch}" 'length > 0 and index($branch) == null' <<< "${branches}" >/dev/null; then
        return 3
    fi
    return 7
}

# Read only safe archived server identities from the last successful Jenkins build.
pufferfish_build_metadata() {
    local branch="$1" build="${2:-lastSuccessfulBuild}" response
    [[ ${build} == lastSuccessfulBuild || ${build} =~ ^[1-9][0-9]*$ ]] || return 1
    [[ ${branch} =~ ^[0-9]+[.][0-9]+$ ]] || return 1
    response="$(java_exact_publisher_response --connect-timeout 3 --max-time 12 -fsSL -H "User-Agent: ${JV_PAPER_USER_AGENT}" \
        "https://ci.pufferfish.host/job/Pufferfish-${branch}/${build}/api/json")" || return "$?"
    pufferfish_build_identity "${build}" <<< "${response}"
}

pufferfish_build_identity() {
    local build="$1"
    jq -esc --arg build "${build}" '
        select(length == 1) | .[0] | select(.artifacts | type == "array") |
        .number as $number | select($build == "lastSuccessfulBuild" or ($number | tostring) == $build) |
        select(.result == "SUCCESS" and
            (.number | type == "number" and . > 0 and . <= 2147483647 and floor == .)) |
        [.artifacts[] | select(.fileName | type == "string") |
            select(.fileName | test("^pufferfish-paperclip-[0-9]+[.][0-9]+([.][0-9]+)?-R0[.]1-SNAPSHOT-(mojmap|reobf)[.]jar$"))] |
        select(length > 0 and all(.[]; .fileName as $name |
            (.relativePath | type == "string" and
                test("^[A-Za-z0-9_.-]+(/[A-Za-z0-9_.-]+)*$") and
                (split("/") | all(.[]; . != ".." and . != ".")) and
                (endswith("/" + $name) or . == $name)))) |
        map(.fileName | capture("^pufferfish-paperclip-(?<version>[0-9]+[.][0-9]+([.][0-9]+)?)-R0").version) |
        unique | select(length == 1) | {game:.[0],pufferfish_build:$number}' || return 6
}

# Patches share a Jenkins job. A different game at lastSuccessfulBuild
# establishes neither support nor absence for the requested exact patch.
pufferfish_exact_build() {
    local branch="$1" game="$2" latest status history entries entry resolved selected='' incomplete=no
    latest="$(pufferfish_build_metadata "${branch}")" || {
        status=$?
        if [[ ${classify_exact_release:-0} == 1 ]] && (( status == 3 )); then
            pufferfish_missing_branch "${branch}" || status=$?
            (( status == 7 )) || return "${status}"
        else
            return "${status}"
        fi
    }
    if [[ -n ${latest} ]] && jq -e --arg game "${game}" '.game == $game' <<< "${latest}" >/dev/null; then
        printf '%s\n' "${latest}"
        return
    fi
    # The extra row distinguishes a complete retained inventory from a prefix.
    history="$(java_publisher_catalog_response --connect-timeout 3 --max-time 12 --max-filesize 1048576 -fsSG \
        --data-urlencode 'tree=name,builds[number,result,artifacts[fileName,relativePath]]{0,17}' \
        -H "User-Agent: ${JV_PAPER_USER_AGENT}" "https://ci.pufferfish.host/job/Pufferfish-${branch}/api/json")" || return "$?"
    entries="$(jq -ces --arg name "Pufferfish-${branch}" '
        select(length == 1) | .[0] | select(.name == $name) | .builds | select(type == "array" and
            all(.[]; type == "object" and
                (.number | type == "number" and . > 0 and . <= 2147483647 and floor == .) and
                has("result") and (.result == null or .result == "SUCCESS" or .result == "FAILURE" or
                    .result == "UNSTABLE" or .result == "ABORTED" or .result == "NOT_BUILT"))) |
        select((map(.number) | unique | length) == length)' <<< "${history}")" || return 6
    [[ $(jq -r length <<< "${entries}") -le 16 ]] || incomplete=yes
    if [[ -n ${latest} ]] && ! jq -e --argjson latest "${latest}" '
        any(.[]; .number == $latest.pufferfish_build and .result == "SUCCESS")' <<< "${entries}" >/dev/null; then
        incomplete=yes
    fi
    while IFS= read -r entry; do
        resolved="$(pufferfish_build_identity "$(jq -r .number <<< "${entry}")" <<< "${entry}")" || {
            incomplete=yes
            continue
        }
        if jq -e --arg game "${game}" '.game == $game' <<< "${resolved}" >/dev/null; then
            selected="${resolved}"
            break
        fi
    done < <(jq -c 'sort_by(.number) | reverse | .[:16][] | select(.result == "SUCCESS")' <<< "${entries}")
    if [[ -n ${selected} ]]; then
        printf '%s\n' "${selected}"
    elif [[ ${incomplete} == yes ]]; then
        return 7
    else
        return 8
    fi
}

setup_java_image() {
    if [[ $1 =~ ^1\.(20\.[56]|21(\.([0-9]|1[01]))?)$ ]]; then printf java21
    elif [[ $1 =~ ^26\.[1-9][0-9]*(\.(0|[1-9][0-9]*))?$ ]]; then printf java25
    else return 1; fi
}

# Validate the published asset before a version enters any policy or picker.
leaf_release_catalog() {
    local record page=1 response count releases='[]' games records game image exact_tag=no match
    games="$(vanilla_metadata)" || return "$?"
    games="$(jq -ce '[.versions[] | select(.type == "release") | .id]' <<< "${games}")" || return 6
    while :; do
        response="$(java_publisher_catalog_response --connect-timeout 3 --max-time 12 -fsS -H "User-Agent: ${JV_PAPER_USER_AGENT}" \
            "https://api.github.com/repos/Winds-Studio/Leaf/releases?per_page=100&page=${page}")" || return "$?"
        count="$(jq -ers 'select(length == 1) | .[0] | select(type == "array") | length' <<< "${response}")" || return 6
        if [[ ${classify_exact_release:-0} == 1 ]]; then
            match="$(jq -er --arg tag "ver-${specific}" '
                select(all(.[]; type == "object" and (.tag_name | type == "string" and length > 0)
                    and (.draft | type == "boolean"))) |
                any(.[]; .tag_name == $tag and .draft == false) | tostring' <<< "${response}")" || return 6
            [[ ${match} != true ]] || exact_tag=yes
        fi
        records="$(jq -c --argjson games "${games}" '
            [ .[] | try (
                select(.draft == false and (.assets | type == "array") and (.prerelease | type == "boolean") and (.body | type == "string")) |
                . as $release |
                (.tag_name | capture("^ver-(?<game>[1-9][0-9]*[.](0|[1-9][0-9]*)([.](0|[1-9][0-9]*))?)\\z").game) as $game |
                select($games | index($game)) |
                [.assets[] | select(.name | startswith("leaf-" + $game + "-") and endswith(".jar"))] |
                select(length == 1) | .[0] |
                select(.state == "uploaded" and (.digest | test("^sha256:[a-f0-9]{64}\\z"))) |
                select(.browser_download_url == ("https://github.com/Winds-Studio/Leaf/releases/download/" + $release.tag_name + "/" + .name)) |
                (.name | ltrimstr("leaf-" + $game + "-") | capture("^(?<build>[1-9][0-9]*)[.]jar\\z").build) as $build |
                ($release.body | split("Checksums") | select(length == 2) | last | ascii_downcase) as $notes |
                (.digest | ltrimstr("sha256:")) as $hash |
                # The release workflow publishes one table for the exact uploaded JAR.
                ($notes | split("\n") | map(select(startswith("|")) |
                    split("|") | map(gsub("[`*[:space:]]"; "")))) as $table |
                select(($table | first) == ["","",.name,""] and
                    ([$table[] | select(.[1] == "sha256")] == [["","sha256",$hash,""]])) |
                {game:$game,build:$build,digest:.digest,channel:(
                    if $notes | test("this version of leaf is unsupported|no longer supported") then "UNSUPPORTED"
                    elif $release.prerelease or ($notes | test("early dev|only for testing|experimental build|do not use in .*production")) then "TESTING"
                    elif ($notes | test("\\[!(warning|caution|important)\\]")) or
                         ($notes | split($hash) | last | test("[a-z0-9]")) then empty
                    else "PRODUCTION" end)}
            ) catch empty ]' <<< "${response}")" || return 1
        releases="$(jq -c --argjson more "${records}" '. + $more' <<< "${releases}")" || return 1
        (( count == 100 )) || break
        ((page++))
    done
    if [[ ${classify_exact_release:-0} == 1 && ${exact_tag} == no ]]; then
        return 3
    fi
    records="$(jq -cr 'group_by(.game) | map(select(length == 1) | first) |
        sort_by(.game | split(".") | map(tonumber)) | reverse[]' <<< "${releases}")" || return 1
    while IFS= read -r record; do
        [[ -n ${record} ]] || continue
        game="$(jq -r .game <<< "${record}")"
        image="$(setup_java_image "${game}")" || continue
        jq -c --arg image "${image}" '. + {image:$image,loader:""}' <<< "${record}"
    done <<< "${records}" | jq -sc '.'
}

leaf_select_release() {
    jq -ce --arg policy "$1" --arg version "$2" '
        map(select(if $policy == "recommended" then .channel == "PRODUCTION"
                   elif $policy == "latest" then true
                   elif $policy == "pinned" then .game == $version else false end)) | first // empty'
}

# Verify the artifact that the runtime will actually select. Paper compatibility
# never establishes availability of a Pufferfish or Leaf Minecraft release.
paper_fork_version_available() {
    local software="$1" version="$2" branch resolved
    [[ ${version} =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] || return 1
    case "${software}" in
        pufferfish)
            branch="$(cut -d. -f1,2 <<< "${version}")"
            resolved="$(pufferfish_build_metadata "${branch}" "${MINECRAFT_PUFFERFISH_BUILD:-}")" || return 1
            jq -e --arg game "${version}" '.game == $game' <<< "${resolved}" >/dev/null
            ;;
        leaf) leaf_release_catalog | leaf_select_release pinned "${version}" >/dev/null ;;
        *) return 1 ;;
    esac
}

# Verify an exact pair with the selected project's own metadata/artifact
# repository. No release synthesis, third-party jars, or alternate loader.
mojang_release_available() {
    local version="$1"
    if [[ ${classify_exact_release:-0} == 1 ]]; then
        local manifest
        manifest="$(vanilla_metadata)" || return "$?"
        java_release_catalog_member --arg v "${version}" '
            select(all(.versions[]; type == "object" and (.id | type == "string" and length > 0)
                and (.type | type == "string" and length > 0))) |
            any(.versions[]; .id == $v and .type == "release") | tostring' <<< "${manifest}"
        return
    fi
    curl --connect-timeout 3 --max-time 8 -fsSL https://piston-meta.mojang.com/mc/game/version_manifest_v2.json | jq -e --arg version "${version}" '
        any(.versions[]; .id == $version and .type == "release")' >/dev/null
}

java_server_version_available() {
    local software="$1" version="$2" loader="${3:-}" url coordinate prefix patch digest response java_class=65
    [[ ${version} =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] || return 1
    case "${software}" in
        paper|purpur|vanilla)
            [[ -z ${loader} ]] || return 1
            MINECRAFT_SERVER_TYPE="${software}" server_version_available "${version}"
            return
            ;;
    esac
    # Loader metadata can describe profiles independently of Mojang releases.
    # Require the exact game release as well as the exact implementation pair.
    if [[ ${software} == leaf ]]; then
        [[ -z ${loader} ]] && paper_fork_version_available leaf "${version}"
        return
    fi
    mojang_release_available "${version}" || return "$?"
    if [[ ${software} == pufferfish ]]; then
        [[ -z ${loader} ]] && paper_fork_version_available "${software}" "${version}"
        return
    fi
    if [[ ${software} == spigot ]]; then
        [[ -z ${loader} ]] || return 1
        [[ ${version} != 26.* ]] || java_class=69
        response="$(java_exact_publisher_response --connect-timeout 3 --max-time 12 -fsSL "https://hub.spigotmc.org/versions/${version}.json")" || return "$?"
        jq -e --argjson java_class "${java_class}" '
            (.javaVersions | type == "array" and index($java_class) != null) and
            ([.refs.BuildData,.refs.Bukkit,.refs.CraftBukkit,.refs.Spigot] | all(.[]; type == "string" and test("^[a-f0-9]{40}$")))' <<< "${response}" >/dev/null
        return
    fi
    [[ ${#loader} -le 64 && ${loader} =~ ^[0-9]+(\.[0-9]+){1,3}(-[A-Za-z0-9]+([.-][A-Za-z0-9]+)*)?$ ]] || return 1
    case "${software}" in
        forge)
            url="https://maven.minecraftforge.net/net/minecraftforge/forge/${version}-${loader}/forge-${version}-${loader}-installer.jar.sha1"
            ;;
        neoforge)
            # 26.x includes the full game version, with an omitted patch
            # represented by zero. Older releases omit Minecraft's leading 1.
            if [[ ${version} =~ ^26\.([1-9][0-9]*)(\.([0-9]+))?$ ]]; then
                prefix="26.${BASH_REMATCH[1]}.${BASH_REMATCH[3]:-0}."
            elif [[ ${version} =~ ^1\.(20|21)(\.([0-9]+))?$ ]]; then
                patch=${BASH_REMATCH[3]:-0}
                prefix="${BASH_REMATCH[1]}.${patch}."
            else
                return 1
            fi
            [[ ${loader} == "${prefix}"* ]] || return 1
            url="https://maven.neoforged.net/releases/net/neoforged/neoforge/${loader}/neoforge-${loader}-installer.jar.sha1"
            ;;
        fabric|quilt)
            if [[ ${software} == fabric ]]; then
                url="https://meta.fabricmc.net/v2/versions/loader/${version}/${loader}/server/json"
                coordinate="net.fabricmc:fabric-loader:${loader}"
            else
                url="https://meta.quiltmc.org/v3/versions/loader/${version}/${loader}/server/json"
                coordinate="org.quiltmc:quilt-loader:${loader}"
            fi
            response="$(java_exact_publisher_response --connect-timeout 3 --max-time 12 -fsSL "${url}")" || return "$?"
            jq -es --arg version "${version}" --arg loader "${coordinate}" '
                length == 1 and (.[0] | .inheritsFrom == $version and
                (.mainClass | type == "string" and endswith(".KnotServer")) and
                any(.libraries[]; .name == $loader))' <<< "${response}" >/dev/null || return 6
            return
            ;;
        *) return 1 ;;
    esac
    digest="$(java_exact_publisher_response --connect-timeout 3 --max-time 12 -fsSL "${url}")" || return "$?"
    [[ ${digest} =~ ^[a-fA-F0-9]{40}$ ]] || return 6
}

# Setup-time discovery only. Every selected pin is checked by the existing
# exact upstream verifier before it can leave this resolver.
resolve_setup_loader() {
    local software="$1" game="$2" metadata prefix
    case "${software}" in
        paper|purpur|vanilla|spigot|pufferfish|leaf) return 0 ;;
        fabric|quilt)
            local api=https://meta.fabricmc.net/v2
            [[ ${software} != quilt ]] || api=https://meta.quiltmc.org/v3
            metadata="$(java_exact_publisher_response --connect-timeout 3 --max-time 8 -fsSL "${api}/versions/loader/${game}")" || return "$?"
            if [[ ${classify_exact_release:-0} == 1 ]]; then
                metadata="$(jq -ces 'select(length == 1) | .[0] | select(type == "array" and
                    all(.[]; .loader | type == "object" and (.version | type == "string" and length > 0) and
                        (.stable == null or (.stable | type == "boolean"))))' <<< "${metadata}")" || return 6
                [[ ${metadata} != '[]' ]] || return 3
            fi
            jq -er --arg software "${software}" '
                [.[] | .loader
                    | select(if $software == "fabric" then .stable == true else .stable != false end)
                    | .version | select(test("^[0-9]+([.][0-9]+){1,3}$"))]
                | if $software == "quilt" then sort_by(split(".") | map(tonumber)) | last else first end
                // empty' <<< "${metadata}" || return 7
            ;;
        forge)
            metadata="$(java_publisher_catalog_response --connect-timeout 3 --max-time 8 -fsSL https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json)" || return "$?"
            jq -es 'length == 1 and (.[0].promos | type == "object" and length > 0)' <<< "${metadata}" >/dev/null || return 6
            jq -er --arg game "${game}" '.promos | .[$game + "-recommended"] // .[$game + "-latest"] | select(type == "string" and test("^[0-9]+([.][0-9]+){1,3}$"))' <<< "${metadata}" || return 7
            ;;
        neoforge)
            if [[ ${game} =~ ^26\.([1-9][0-9]*)(\.([0-9]+))?$ ]]; then
                prefix="26.${BASH_REMATCH[1]}.${BASH_REMATCH[3]:-0}."
            elif [[ ${game} =~ ^1\.(20|21)(\.([0-9]+))?$ ]]; then
                prefix="${BASH_REMATCH[1]}.${BASH_REMATCH[3]:-0}."
            else return 1; fi
            metadata="$(java_retry_catalog_transport=1 java_publisher_catalog_response --connect-timeout 3 --max-time 8 -fsSL https://maven.neoforged.net/releases/net/neoforged/neoforge/maven-metadata.xml)" || return "$?"
            python3 -c '
import re, sys, xml.etree.ElementTree as xml
raw = sys.stdin.read(4 * 1024 * 1024 + 1)
if len(raw) > 4 * 1024 * 1024 or "<!" in raw:
    sys.exit(6)
try:
    root = xml.fromstring(raw)
except xml.ParseError:
    sys.exit(6)
catalogs = root.findall("./versioning/versions")
if root.tag != "metadata" or len(catalogs) != 1 or not len(catalogs[0]):
    sys.exit(6)
matched = []
for element in catalogs[0]:
    if element.tag != "version" or len(element) or not (element.text or "").strip():
        sys.exit(6)
    if element.text.startswith(sys.argv[1]):
        matched.append(element.text)
if not matched:
    sys.exit(3)
stable = [version for version in matched if re.fullmatch(r"[0-9]+(?:[.][0-9]+){2,3}", version)]
if not stable:
    sys.exit(7)
stable.sort(key=lambda version: tuple(map(int, version.split("."))))
print(stable[-1])
' "${prefix}" <<< "${metadata}"
            ;;
        *) return 1 ;;
    esac
}

resolve_setup_java_runtime() {
    local software="$1" policy="$2" specific="$3" game loader image releases branches branch discovered metadata reason catalog outcome pufferfish="{}"
    if [[ ${software} == leaf ]]; then
        catalog="$(leaf_release_catalog)" || return "$?"
        leaf_select_release "${policy}" "${specific}" <<< "${catalog}" || return 7
        return
    fi
    case "${policy}" in recommended|pinned) ;; *) return 1 ;; esac
    if [[ ${policy} == pinned ]]; then
        [[ ${specific} =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] || return 1
        releases="${specific}"
    else
        case "${software}" in
            pufferfish)
                # Ask Jenkins which jobs actually exist. Probing every Mojang
                # release branch can exhaust the preview deadline on absent jobs.
                branches="$(pufferfish_job_branches)" || return 1
                branches="$(jq -er '.[0:4][]' <<< "${branches}")" || return 1
                releases=''
                while IFS= read -r branch; do
                    pufferfish="$(pufferfish_build_metadata "${branch}")" || continue
                    discovered="$(jq -er .game <<< "${pufferfish}")" || continue
                    [[ $(cut -d. -f1,2 <<< "${discovered}") == "${branch}" ]] || continue
                    mojang_release_available "${discovered}" || continue
                    releases="${discovered}"
                    break
                done <<< "${branches}"
                ;;
            fabric|neoforge)
                releases="$(vanilla_metadata | jq -er '.versions[] | select(.type == "release") | .id')" || return 1
                ;;
            *) releases="$(vanilla_metadata | jq -er '[.versions[] | select(.type == "release") | .id][0:8][]')" || return 1 ;;
        esac
    fi
    while IFS= read -r game; do
        image="$(setup_java_image "${game}")" || continue
        [[ ${software} != spigot ]] || image+=-jdk
        if [[ ${classify_exact_release:-0} == 1 ]]; then
            case "${software}" in
                paper|purpur|vanilla) ;;
                *) mojang_release_available "${game}" || return "$?" ;;
            esac
        fi
        loader="$(resolve_setup_loader "${software}" "${game}")" || {
            outcome=$?
            [[ ${classify_exact_release:-0} != 1 ]] || return "${outcome}"
            continue
        }
        validate_java_server_selection "${software}" pinned "${game}" "${loader}" "${image}" || continue
        if [[ ${software} == pufferfish ]]; then
            if [[ ${policy} == pinned ]]; then
                branch="$(cut -d. -f1,2 <<< "${game}")"
                pufferfish="$(pufferfish_exact_build "${branch}" "${game}")" || {
                    outcome=$?
                    if [[ ${classify_exact_release:-0} == 1 ]]; then
                        return "${outcome}"
                    fi
                    continue
                }
                if ! jq -e --arg game "${game}" '.game == $game' <<< "${pufferfish}" >/dev/null; then
                    [[ ${classify_exact_release:-0} != 1 ]] || return 7
                    continue
                fi
                if [[ ${classify_exact_release:-0} != 1 ]]; then mojang_release_available "${game}" || continue; fi
            fi
        else
            java_server_version_available "${software}" "${game}" "${loader}" || {
                outcome=$?
                if [[ ${classify_exact_release:-0} == 1 ]]; then
                    # A missing artifact for the selected loader does not
                    # establish that the game itself has no publisher release.
                    if [[ -n ${loader} && ${outcome} == 3 ]]; then return 7; fi
                    return "${outcome}"
                fi
                continue
            }
        fi
        reason=''
        jq -cn --arg game "${game}" --arg loader "${loader}" --arg image "${image}" --arg reason "${reason}" --argjson fork "${pufferfish}" '{game:$game,loader:$loader,image:$image,reason:$reason} + $fork'
        return
    done <<< "${releases}"
    return 1
}

# Machine-readable contract for the matrix. Ordinary Setup still receives the
# existing resolver result and ordinary Version keeps its boolean contract.
resolve_switch_matrix_java_release() {
    local classify_exact_release=1 resolved outcome reason
    if resolved="$(resolve_setup_java_runtime "$1" pinned "$2")"; then
        jq -cn --argjson runtime "${resolved}" '{outcome:"AVAILABLE",runtime:$runtime}'
    else
        outcome=$?
        case ${outcome} in
            3) printf '%s\n' '{"outcome":"UNSUPPORTED"}'; return ;;
            4) reason=publisher_request_timeout ;;
            5) reason=publisher_request_failed ;;
            6) reason=publisher_response_invalid ;;
            7) reason=publisher_selection_unverified ;;
            8) printf '%s\n' '{"outcome":"UNAVAILABLE"}'; return ;;
            *) reason=publisher_exact_release_inconclusive ;;
        esac
        jq -cn --arg verification "${reason}" '{outcome:"UNVERIFIED",verification:$verification}'
    fi
}

purpur_metadata() {
    local metadata
    metadata="$(java_publisher_catalog_response --connect-timeout 5 --max-time 20 -fsSL https://api.purpurmc.org/v2/purpur)" || return "$?"
    jq -e '
        select(.metadata.current | type == "string" and test("^[0-9]+([.][0-9]+){1,2}$"))
        | select(.versions | type == "array" and length > 0 and all(.[]; type == "string" and test("^[0-9]+([.][0-9]+){1,2}$")))
        | select(.metadata.current as $current | .versions | index($current) != null)' <<< "${metadata}" || return 6
}
vanilla_metadata() {
    local metadata
    metadata="$(java_publisher_catalog_response --connect-timeout 5 --max-time 20 -fsSL https://piston-meta.mojang.com/mc/game/version_manifest_v2.json)" || return "$?"
    jq -e '
        select(.latest.release | type == "string" and length > 0)
        | select(.versions | type == "array" and length > 0)
        | select(.latest.release as $release | any(.versions[]; .id == $release and .type == "release"))' <<< "${metadata}" || return 6
}
server_version_available() {
    local version="$1"
    [[ ${version} =~ ^[0-9A-Za-z._-]+$ ]] || return 1
    if [[ ${classify_exact_release:-0} == 1 ]]; then
        local catalog
        case "${MINECRAFT_SERVER_TYPE-paper}" in
            purpur)
                catalog="$(purpur_metadata)" || return "$?"
                java_release_catalog_member --arg v "${version}" '.versions | (index($v) != null) | tostring' <<< "${catalog}"
                return
                ;;
            vanilla) mojang_release_available "${version}"; return ;;
        esac
    fi
    case "${MINECRAFT_SERVER_TYPE-paper}" in
        paper) paper_version_build_channel "${version}" >/dev/null ;;
        purpur) purpur_metadata | jq -e --arg v "${version}" '.versions | index($v) != null' >/dev/null ;;
        vanilla) vanilla_metadata | jq -e --arg v "${version}" 'any(.versions[]; .id == $v and .type == "release")' >/dev/null ;;
        pufferfish|leaf|spigot|forge|neoforge|fabric|quilt) java_server_version_available "${MINECRAFT_SERVER_TYPE}" "${version}" "${MINECRAFT_LOADER_VERSION:-}" ;;
        *) return 1 ;;
    esac
}
resolve_recommended_server_version() {
    case "${MINECRAFT_SERVER_TYPE-paper}" in
        paper) resolve_latest_stable_paper_version ;;
        purpur) purpur_metadata | jq -er '.metadata.current' ;;
        vanilla) vanilla_metadata | jq -er '.latest.release' ;;
        pufferfish|leaf) resolve_setup_java_runtime "${MINECRAFT_SERVER_TYPE}" recommended '' | jq -er .game ;;
        *) return 1 ;;
    esac
}
resolve_latest_server_version() {
    case "${MINECRAFT_SERVER_TYPE-paper}" in
        paper) resolve_latest_available_paper_version ;;
        purpur) purpur_metadata | jq -er '.versions | sort_by(split(".") | map(tonumber)) | last' ;;
        leaf) resolve_setup_java_runtime leaf latest '' | jq -er .game ;;
        vanilla|pufferfish) resolve_recommended_server_version ;;
        *) return 1 ;;
    esac
}
server_version_channel() {
    if [[ ${MINECRAFT_SERVER_TYPE-paper} == leaf ]]; then resolve_setup_java_runtime leaf pinned "$1" | jq -er .channel
    elif [[ ${MINECRAFT_SERVER_TYPE-paper} == paper ]]; then paper_version_build_channel "$1"; else server_version_available "$1" && printf RELEASE; fi
}

render_runtime_files() {
    require_config
    case "${MINECRAFT_SERVER_TYPE-paper}" in paper|purpur|vanilla) ;; *)
        echo 'ERROR: this server software requires the per-instance runtime.' >&2
        return 1 ;;
    esac
    if declare -F jv_trace_run >/dev/null; then jv_trace_run install -d -m0700 -o root -g root "${JV_CONFIG_DIR}"; else install -d -m0700 -o root -g root "${JV_CONFIG_DIR}"; fi
    if declare -F jv_trace_run >/dev/null; then jv_trace_run install -d -m0755 -o root -g root /etc/containers/systemd; else install -d -m0755 -o root -g root /etc/containers/systemd; fi

    local rcon_password env_tmp quadlet_tmp backup_tmp service_tmp timer_tmp image_ref runtime_version paper_channel build_channel
    if [[ ${JUSTVOXEL_REGENERATE_RCON:-0} == 1 ]]; then
        rcon_password="$(openssl rand -hex 24)"
    elif [[ -r ${JV_MC_ENV} ]]; then
        rcon_password="$(sed -n 's/^RCON_PASSWORD=//p' "${JV_MC_ENV}" | head -n1)"
    else
        rcon_password="$(openssl rand -hex 24)"
    fi
    [[ -n ${rcon_password} ]] || rcon_password="$(openssl rand -hex 24)"

    runtime_version="${MINECRAFT_VERSION}"
    paper_channel=default
    if [[ ${MINECRAFT_SERVER_TYPE} != paper ]]; then
        if [[ ${MINECRAFT_VERSION_MODE} == latest ]]; then runtime_version=LATEST; fi
    elif [[ ${MINECRAFT_VERSION_MODE} == latest ]]; then
        runtime_version=LATEST
        paper_channel=experimental
    elif [[ ${MINECRAFT_VERSION_MODE} == pinned ]] &&
        { ! grep -Fqx "VERSION=${MINECRAFT_VERSION}" "${JV_MC_ENV}" 2>/dev/null || ! grep -Fqx TYPE=PAPER "${JV_MC_ENV}" 2>/dev/null; }; then
        build_channel="$(paper_version_build_channel "${MINECRAFT_VERSION}")" || return 1
        if [[ ${build_channel} != STABLE ]]; then
            paper_channel=experimental
        fi
    elif [[ ${MINECRAFT_VERSION_MODE} == pinned ]] && grep -Fqx 'PAPER_CHANNEL=experimental' "${JV_MC_ENV}" 2>/dev/null; then
        paper_channel=experimental
    fi
    env_tmp="$(mktemp "${JV_CONFIG_DIR}/.minecraft.env.XXXXXX")"
    cp "${JV_TEMPLATE_ROOT}/config/minecraft.env.in" "${env_tmp}"
    replace_token "${env_tmp}" MINECRAFT_SERVER_TYPE "${MINECRAFT_SERVER_TYPE^^}"
    replace_token "${env_tmp}" EULA TRUE
    replace_token "${env_tmp}" MINECRAFT_VERSION "${runtime_version}"
    replace_token "${env_tmp}" PAPER_CHANNEL "${paper_channel}"
    replace_token "${env_tmp}" MINECRAFT_UID "${MINECRAFT_UID}"
    replace_token "${env_tmp}" MINECRAFT_GID "${MINECRAFT_GID}"
    replace_token "${env_tmp}" JAVA_MEMORY "${JAVA_MEMORY^^}"
    replace_token "${env_tmp}" GAME_MODE "${GAME_MODE}"
    replace_token "${env_tmp}" DIFFICULTY "${DIFFICULTY}"
    replace_token "${env_tmp}" WHITELIST_ENABLED "$([[ ${WHITELIST_ENABLED} == yes ]] && printf TRUE || printf FALSE)"
    replace_token "${env_tmp}" ENFORCE_WHITELIST "$([[ ${ENFORCE_WHITELIST} == yes ]] && printf TRUE || printf FALSE)"
    replace_token "${env_tmp}" MAX_PLAYERS "${MAX_PLAYERS}"
    replace_token "${env_tmp}" RCON_PASSWORD "${rcon_password}"
    replace_token "${env_tmp}" MOTD "$(systemd_env_escape "${MOTD}")"
    # JustVoxel verifies these binaries; the container only owns ViaVersion.
    replace_token "${env_tmp}" PLUGINS_LINE '# Cross-play binaries are managed by JustVoxel.'
    if [[ ${MINECRAFT_SERVER_TYPE} != paper ]]; then sed -i '/^PAPER_CHANNEL=/d' "${env_tmp}"; fi
    if ! server_supports_viaversion; then sed -i '/^MODRINTH_PROJECTS=/d' "${env_tmp}"; fi
    if declare -F jv_trace_run >/dev/null; then jv_trace_run install -o root -g root -m0600 "${env_tmp}" "${JV_MC_ENV}"; else install -o root -g root -m0600 "${env_tmp}" "${JV_MC_ENV}"; fi
    rm -f "${env_tmp}"

    image_ref="$(minecraft_image_ref)"
    quadlet_tmp="$(mktemp /etc/containers/systemd/.minecraft.container.XXXXXX)"
    cp "${JV_TEMPLATE_ROOT}/quadlets/minecraft.container.in" "${quadlet_tmp}"
    if server_supports_managed_crossplay; then
        sed -i '/^ExecStartPre=.*validate-data-mount$/a ExecStartPre=/usr/libexec/justvoxel/mjust/managed-crossplay-start' "${quadlet_tmp}"
    fi
    replace_token "${quadlet_tmp}" MINECRAFT_IMAGE "${image_ref}"
    replace_token "${quadlet_tmp}" DATA_PATH "${DATA_PATH}"
    replace_token "${quadlet_tmp}" JAVA_PORT "${JAVA_PORT}"
    replace_token "${quadlet_tmp}" CONTAINER_MEMORY "${CONTAINER_MEMORY^^}"
    if [[ ${BEDROCK_ENABLED} == yes ]]; then
        replace_token "${quadlet_tmp}" BEDROCK_PUBLISH_PORT "PublishPort=${BEDROCK_PORT}:19132/tcp"
        sed -i "/^PublishPort=${BEDROCK_PORT}:19132\/tcp$/a PublishPort=${BEDROCK_PORT}:19132/udp" "${quadlet_tmp}"
    else
        replace_token "${quadlet_tmp}" BEDROCK_PUBLISH_PORT '# Bedrock port disabled.'
    fi
    if declare -F jv_trace_run >/dev/null; then jv_trace_run install -o root -g root -m0644 "${quadlet_tmp}" "${JV_QUADLET}"; else install -o root -g root -m0644 "${quadlet_tmp}" "${JV_QUADLET}"; fi
    rm -f "${quadlet_tmp}"

    backup_tmp="$(mktemp "${JV_CONFIG_DIR}/.minecraft-backup.env.XXXXXX")"
    cp "${JV_TEMPLATE_ROOT}/config/minecraft-backup.env.in" "${backup_tmp}"
    replace_token "${backup_tmp}" DATA_PATH "${DATA_PATH}"
    replace_token "${backup_tmp}" BACKUP_PATH "${BACKUP_PATH}"
    replace_token "${backup_tmp}" BACKUP_KEEP "${BACKUP_KEEP}"
    replace_token "${backup_tmp}" BACKUP_MOUNT_POINT "${BACKUP_MOUNT_POINT}"
    replace_token "${backup_tmp}" BACKUP_EXPECTED_UUID "${BACKUP_EXPECTED_UUID}"
    replace_token "${backup_tmp}" BACKUP_EXPECTED_SOURCE "${BACKUP_EXPECTED_SOURCE}"
    if declare -F jv_trace_run >/dev/null; then jv_trace_run install -o root -g root -m0600 "${backup_tmp}" "${JV_BACKUP_ENV}"; else install -o root -g root -m0600 "${backup_tmp}" "${JV_BACKUP_ENV}"; fi
    rm -f "${backup_tmp}"

    service_tmp="$(mktemp /etc/systemd/system/.minecraft-backup.service.XXXXXX)"
    cp "${JV_TEMPLATE_ROOT}/systemd/minecraft-backup.service.in" "${service_tmp}"
    replace_token "${service_tmp}" DATA_PATH "${DATA_PATH}"
    if [[ -n ${BACKUP_MOUNT_POINT} ]]; then
        replace_token "${service_tmp}" BACKUP_REQUIREMENT_PATH "${BACKUP_MOUNT_POINT}"
    else
        replace_token "${service_tmp}" BACKUP_REQUIREMENT_PATH "${BACKUP_PATH}"
    fi
    if declare -F jv_trace_run >/dev/null; then jv_trace_run install -o root -g root -m0644 "${service_tmp}" "${JV_BACKUP_SERVICE}"; else install -o root -g root -m0644 "${service_tmp}" "${JV_BACKUP_SERVICE}"; fi
    rm -f "${service_tmp}"

    timer_tmp="$(mktemp /etc/systemd/system/.minecraft-backup.timer.XXXXXX)"
    cp "${JV_TEMPLATE_ROOT}/systemd/minecraft-backup.timer.in" "${timer_tmp}"
    replace_token "${timer_tmp}" BACKUP_SCHEDULE "${BACKUP_SCHEDULE}"
    if declare -F jv_trace_run >/dev/null; then jv_trace_run install -o root -g root -m0644 "${timer_tmp}" "${JV_BACKUP_TIMER}"; else install -o root -g root -m0644 "${timer_tmp}" "${JV_BACKUP_TIMER}"; fi
    rm -f "${timer_tmp}"

    if declare -F jv_trace_run >/dev/null; then jv_trace_run systemctl daemon-reload; else systemctl daemon-reload; fi
}

activate_backup_timer() {
    if [[ ${BACKUP_TIMER_ENABLED} == yes ]]; then
        if declare -F jv_trace_run >/dev/null; then jv_trace_run systemctl enable minecraft-backup.timer >/dev/null; else systemctl enable minecraft-backup.timer >/dev/null; fi
        if declare -F jv_trace_run >/dev/null; then jv_trace_run systemctl restart minecraft-backup.timer; else systemctl restart minecraft-backup.timer; fi
    else
        if declare -F jv_trace_run >/dev/null; then jv_trace_run systemctl disable --now minecraft-backup.timer 2>/dev/null || true; else systemctl disable --now minecraft-backup.timer 2>/dev/null || true; fi
    fi
}

render_runtime() {
    render_runtime_files
    activate_backup_timer
}

# Legacy fixed-runtime callers have no instance firewall journal. Record only
# newly added Geyser TCP rules, including their zone, before changing firewalld.
# Missing evidence from older deployments never authorizes TCP removal.
geyser_tcp_firewall_add() {
    local port="$1" zone marker status
    validate_port "${port}" || return 1
    zone="$(firewall-cmd --get-default-zone)" || return 1
    [[ ${zone} =~ ^[A-Za-z0-9_-]+$ ]] || return 1
    marker="${JV_CONFIG_DIR}/geyser-tcp-firewall/${port}/${zone}"
    if firewall-cmd --zone="${zone}" --permanent --query-port="${port}/tcp" >/dev/null; then
        return 0
    else
        status=$?
        [[ ${status} == 1 ]] || return 1
    fi
    install -d -m0700 -o root -g root "$(dirname -- "${marker}")" || return 1
    [[ ! -L ${marker} ]] || return 1
    (umask 077; : > "${marker}") || return 1
    firewall-cmd --zone="${zone}" --permanent --add-port="${port}/tcp" >/dev/null || return 1
    firewall-cmd --zone="${zone}" --permanent --query-port="${port}/tcp" >/dev/null
}

geyser_tcp_firewall_remove() {
    local port="$1" marker zone status
    validate_port "${port}" || return 1
    for marker in "${JV_CONFIG_DIR}/geyser-tcp-firewall/${port}/"*; do
        [[ -e ${marker} ]] || continue
        [[ -f ${marker} && ! -L ${marker} && $(stat -c '%u:%g:%a' "${marker}") == 0:0:600 ]] || return 1
        zone="${marker##*/}"
        [[ ${zone} =~ ^[A-Za-z0-9_-]+$ ]] || return 1
        firewall-cmd --zone="${zone}" --permanent --remove-port="${port}/tcp" >/dev/null || return 1
        if firewall-cmd --zone="${zone}" --permanent --query-port="${port}/tcp" >/dev/null; then
            return 1
        else
            status=$?
            [[ ${status} == 1 ]] || return 1
        fi
        rm -f -- "${marker}" || return 1
    done
}

configure_firewall_initial() {
    firewall-cmd --permanent --add-port="${JAVA_PORT}/tcp" >/dev/null
    if [[ ${BEDROCK_ENABLED} == yes ]]; then
        geyser_tcp_firewall_add "${BEDROCK_PORT}" || return 1
        firewall-cmd --permanent --add-port="${BEDROCK_PORT}/udp" >/dev/null
    fi
    firewall-cmd --reload >/dev/null
}

update_firewall_ports() {
    local old_java="$1" old_bedrock_enabled="$2" old_bedrock="$3"
    if [[ ${old_java} != ${JAVA_PORT} ]]; then
        firewall-cmd --permanent --remove-port="${old_java}/tcp" >/dev/null 2>&1 || true
    fi
    if [[ ${old_bedrock_enabled} == yes && ( ${BEDROCK_ENABLED} != yes || ${old_bedrock} != ${BEDROCK_PORT} ) ]]; then
        geyser_tcp_firewall_remove "${old_bedrock}" || return 1
        firewall-cmd --permanent --remove-port="${old_bedrock}/udp" >/dev/null 2>&1 || true
    fi
    firewall-cmd --permanent --add-port="${JAVA_PORT}/tcp" >/dev/null
    if [[ ${BEDROCK_ENABLED} == yes ]]; then
        geyser_tcp_firewall_add "${BEDROCK_PORT}" || return 1
        firewall-cmd --permanent --add-port="${BEDROCK_PORT}/udp" >/dev/null
    fi
    firewall-cmd --reload >/dev/null
}

wait_for_rcon() {
    local timeout="${1:-900}" elapsed=0
    local container="${JV_INSTANCE_CONTAINER:-minecraft}" unit="${JV_INSTANCE_UNIT:-minecraft.service}"
    local properties key value loaded active sub job
    while (( elapsed < timeout )); do
        # A stopping container can still answer RCON; check the unit after the probe.
        if podman exec "${container}" rcon-cli list >/dev/null 2>&1 &&
            systemctl is-active --quiet "${unit}"; then
            return 0
        fi
        # A queued start or automatic restart can still make progress.
        if properties="$(systemctl show --all --property=LoadState,ActiveState,SubState,Job -- "${unit}" 2>/dev/null)"; then
            loaded= active= sub= job=unknown
            while IFS='=' read -r key value; do
                case "${key}" in
                    LoadState) loaded="${value}" ;;
                    ActiveState) active="${value}" ;;
                    SubState) sub="${value}" ;;
                    Job) job="${value}" ;;
                esac
            done <<< "${properties}"
            if [[ ${loaded} == loaded && ( -z ${job} || ${job} == 0 ) &&
                  ( ${active}:${sub} == failed:failed || ${active}:${sub} == inactive:dead ) ]]; then
                return 2
            fi
        fi
        sleep 5
        elapsed=$((elapsed + 5))
    done
    return 1
}

configure_geyser_floodgate() {
    [[ ${BEDROCK_ENABLED} == yes ]] || return 0
    local geyser_config="${DATA_PATH}/plugins/Geyser-Spigot/config.yml"
    if [[ ! -f ${geyser_config} ]]; then
        echo "ERROR: Geyser configuration was not generated: ${geyser_config}" >&2
        return 1
    fi
    if grep -Eq '^[[:space:]]*auth-type:[[:space:]]*floodgate[[:space:]]*$' "${geyser_config}"; then
        return 0
    fi
    echo 'Configuring Geyser to use Floodgate authentication.'
    systemctl stop minecraft.service
    sed -i -E 's/^([[:space:]]*auth-type:).*/\1 floodgate/' "${geyser_config}"
    chown "${MINECRAFT_UID}:${MINECRAFT_GID}" "${geyser_config}"
    restorecon -F "${geyser_config}" 2>/dev/null || true
    grep -Eq '^[[:space:]]*auth-type:[[:space:]]*floodgate[[:space:]]*$' "${geyser_config}"
    systemctl start minecraft.service
    wait_for_rcon 900
}

# The official Linux release entry is the sole authority. Never use preview or LATEST.
resolve_bedrock_release() {
    curl --connect-timeout 3 --max-time 8 -fsSL https://net.web.minecraft-services.net/api/v1.0/download/links |
      jq -er '[.result.links[] | select(.downloadType == "serverBedrockLinux") | .downloadUrl |
        capture("^https://www[.]minecraft[.]net/bedrockdedicatedserver/bin-linux/bedrock-server-(?<version>[0-9]+[.][0-9]+[.][0-9]+([.][0-9]+)?)[.]zip$").version] |
        unique | select(length == 1) | .[0]'
}
