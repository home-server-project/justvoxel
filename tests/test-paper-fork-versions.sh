#!/usr/bin/bash
# Offline fixtures for exact official metadata and bounded recommendation search.
# This script is written for separate verification; it does not access an appliance.
set -euo pipefail
repo_root="$(cd -- "$(dirname -- "$0")/.." && pwd)"
source "${repo_root}/mjust/libexec/common.sh"
reject() { if "$@"; then echo 'Unsupported fork version accepted' >&2; exit 1; fi; }
fixture=valid
curl() {
    [[ ${fixture} != offline ]] || return 22
    case "${*: -1}" in
        https://piston-meta.mojang.com/mc/game/version_manifest_v2.json)
            printf '%s' '{"latest":{"release":"26.3"},"versions":[{"id":"26.3","type":"release"},{"id":"1.21.10","type":"release"},{"id":"1.21.11","type":"release"},{"id":"1.21.8","type":"release"},{"id":"1.21.8-pre1","type":"snapshot"}]}' ;;
        'https://ci.pufferfish.host/api/json?tree=jobs%5Bname%5D')
            # Only published jobs may be queried; no invented 26.x job probes.
            if [[ ${fixture} == no-jobs ]]; then
                printf '%s' '{"jobs":[]}'
            else
                printf '%s' '{"jobs":[{"name":"Pufferfish-1.21"},{"name":"Pufferfish-1.20"},{"name":"Pufferfish-26.3/unsafe"},{"name":"Other"}]}'
            fi ;;
        https://ci.pufferfish.host/job/Pufferfish-1.21/lastSuccessfulBuild/api/json|https://ci.pufferfish.host/job/Pufferfish-1.21/39/api/json)
            case "${fixture}" in
                reobf|both|ambiguous|nonrelease|malformed|zero|negative|string|fractional|absolute|backslash)
                    jq -cn --arg fixture "${fixture}" '
                        {result:"SUCCESS",number:39,artifacts:[{fileName:"pufferfish-paperclip-1.21.10-R0.1-SNAPSHOT-mojmap.jar",relativePath:"build/libs/pufferfish-paperclip-1.21.10-R0.1-SNAPSHOT-mojmap.jar"}]} |
                        if $fixture == "reobf" then .artifacts |= map(.fileName |= sub("mojmap";"reobf") | .relativePath |= sub("mojmap";"reobf"))
                        elif $fixture == "both" then .artifacts += [(.artifacts[0] | .fileName |= sub("mojmap";"reobf") | .relativePath |= sub("mojmap";"reobf"))]
                        elif $fixture == "ambiguous" then .artifacts += [{fileName:"pufferfish-paperclip-1.21.8-R0.1-SNAPSHOT-reobf.jar",relativePath:"pufferfish-paperclip-1.21.8-R0.1-SNAPSHOT-reobf.jar"}]
                        elif $fixture == "nonrelease" then .artifacts |= map(.fileName |= sub("1.21.10";"1.21.9") | .relativePath |= sub("1.21.10";"1.21.9"))
                        elif $fixture == "malformed" then .artifacts[0].fileName="pufferfish-paperclip-latest.jar"
                        elif $fixture == "negative" then .number=-1
                        elif $fixture == "string" then .number="39"
                        elif $fixture == "zero" then .number=0
                        elif $fixture == "fractional" then .number=39.5
                        elif $fixture == "absolute" then .artifacts[0].relativePath="/" + .artifacts[0].relativePath
                        elif $fixture == "backslash" then .artifacts[0].relativePath="..\\" + .artifacts[0].fileName else . end' ;;
                empty) printf '%s' '{"result":"SUCCESS","number":39,"artifacts":[]}' ;;
                failed) printf '%s' '{"result":"FAILURE","number":39,"artifacts":[{"fileName":"pufferfish-paperclip-1.21.10-R0.1-SNAPSHOT-mojmap.jar","relativePath":"build/libs/pufferfish-paperclip-1.21.10-R0.1-SNAPSHOT-mojmap.jar"}]}' ;;
                unsafe) printf '%s' '{"result":"SUCCESS","number":39,"artifacts":[{"fileName":"pufferfish-paperclip-1.21.10-R0.1-SNAPSHOT-mojmap.jar","relativePath":"../pufferfish-paperclip-1.21.10-R0.1-SNAPSHOT-mojmap.jar"}]}' ;;
                *) printf '%s' '{"result":"SUCCESS","number":39,"artifacts":[{"fileName":"pufferfish-paperclip-1.21.10-R0.1-SNAPSHOT-mojmap.jar","relativePath":"build/libs/pufferfish-paperclip-1.21.10-R0.1-SNAPSHOT-mojmap.jar"}]}' ;;
            esac ;;
        'https://api.github.com/repos/Winds-Studio/Leaf/releases?per_page=100&page=1')
            jq -cn --arg fixture "${fixture}" '
                ["1.21.8","1.21.11","26.3","1.21.10"] | map(. as $game |
                    ("leaf-" + $game + "-17.jar") as $jar |
                    ("a"*64) as $hash |
                    {tag_name:("ver-"+$game), draft:false, prerelease:false,
                     assets:[{name:$jar,state:"uploaded",digest:("sha256:"+$hash),
                         browser_download_url:("https://github.com/Winds-Studio/Leaf/releases/download/ver-"+$game+"/"+$jar)}],
                     body:("### 🔒 Checksums\n| | " + $jar + " |\n|---|---|\n| SHA256 | " + $hash + " |" +
                         (if $game == "26.3" then "\n> [!WARNING]\n> This is the early dev build, only for testing usage."
                          elif $game == "1.21.10" then "\n> [!CAUTION]\n> This version of Leaf is unsupported."
                          else "" end))} |
                    if $fixture == "empty" then empty
                    elif $fixture == "failed" then .draft=true
                    elif $fixture == "unsafe" then .assets[0].digest="invalid"
                    elif $fixture == "mismatch" then .tag_name += "/unsafe"
                    elif $fixture == "wrong-artifact" then .assets[0].name="other.jar"
                    elif $fixture == "duplicate-download" then .assets += .assets
                    elif $fixture == "wrong-project" then .assets[0].browser_download_url="https://example.invalid/file"
                    elif $fixture == "missing-hash" then del(.assets[0].digest)
                    elif $fixture == "wrong-hash" then .assets[0].digest=("sha256:"+("b"*64))
                    elif $fixture == "unknown-channel" then .body |= (split("\n>") | first) + "\n> [!WARNING]\n> Unknown"
                    elif $fixture == "no-production" then .prerelease=true else . end)' ;;
        *) return 22 ;; # No unlisted upstream/version may acquire invented builds.
    esac
}
for software in pufferfish leaf; do
    version=1.21.11 image=java21
    [[ ${software} != pufferfish ]] || { version=1.21.10; image=java21; }
    validate_server_type "${software}"
    server_supports_bukkit_plugins "${software}"
    server_supports_viaversion "${software}"
    server_supports_managed_crossplay "${software}"
    reject server_is_mod_loader "${software}"
    validate_java_server_selection "${software}" pinned "${version}" '' "${image}"
    reject validate_java_server_selection "${software}" pinned "${version}" 0.1.0 "${image}"
    java_server_version_available "${software}" "${version}"
    reject java_server_version_available "${software}" "${version}" 0.1.0
    reject java_server_version_available "${software}" 26.4
    reject java_server_version_available "${software}" 1.21.8-pre1
    reject java_server_version_available "${software}" 1.21.9
    for fixture in empty failed unsafe offline; do
        reject java_server_version_available "${software}" "${version}"
        reject resolve_setup_java_runtime "${software}" recommended ''
    done
    fixture=valid
    selected="$(resolve_setup_java_runtime "${software}" recommended '')"
    jq -e --arg version "${version}" --arg image "${image}" '.game == $version and .loader == "" and .image == $image' <<< "${selected}" >/dev/null
    reject resolve_setup_java_runtime "${software}" pinned 1.21.9
    MINECRAFT_SERVER_TYPE=${software}
    [[ $(resolve_recommended_server_version) == "${version}" ]]
    if [[ ${software} == leaf ]]; then
        [[ $(resolve_latest_server_version) == 26.3 && $(server_version_channel "${version}") == PRODUCTION ]]
    else
        [[ $(resolve_latest_server_version) == "${version}" && $(server_version_channel "${version}") == RELEASE ]]
    fi
    for other in paper purpur pufferfish leaf; do
        server_allows_in_place_switch "${software}" "${other}"
        server_allows_in_place_switch "${other}" "${software}"
    done
    for other in spigot vanilla fabric forge neoforge quilt bedrock; do
        reject server_allows_in_place_switch "${software}" "${other}"
        reject server_allows_in_place_switch "${other}" "${software}"
        server_allows_in_place_switch "${other}" "${other}"
    done
    # Paper forks retain Paper's exact Geyser version compatibility check.
    server_crossplay_version_compatible "${version}" "${version}"
    reject server_crossplay_version_compatible "${version}" 26.4
done
fixture=valid
reject java_server_version_available pufferfish 1.21.8 # artifact proves 1.21.10 only
for fixture in mismatch wrong-artifact duplicate-download wrong-project missing-hash wrong-hash unknown-channel; do
    reject java_server_version_available leaf 26.3
    reject resolve_setup_java_runtime leaf recommended ''
done
fixture=valid
catalog="$(leaf_release_catalog)"
jq -e 'map(.game) == ["26.3","1.21.11","1.21.10","1.21.8"]' <<< "${catalog}" >/dev/null
jq -e '.game == "1.21.11" and .channel == "PRODUCTION"' <<< "$(resolve_setup_java_runtime leaf recommended '')" >/dev/null
jq -e '.game == "26.3" and .channel == "TESTING" and .image == "java25"' <<< "$(resolve_setup_java_runtime leaf latest '')" >/dev/null
jq -e '.channel == "UNSUPPORTED"' <<< "$(resolve_setup_java_runtime leaf pinned 1.21.10)" >/dev/null
fixture=no-production
reject resolve_setup_java_runtime leaf recommended ''
fixture=valid
specific="$(resolve_setup_java_runtime pufferfish pinned 1.21.10)"
jq -e '.game == "1.21.10" and .pufferfish_build == 39' <<< "${specific}" >/dev/null
MINECRAFT_PUFFERFISH_BUILD=39 java_server_version_available pufferfish 1.21.10
MINECRAFT_PUFFERFISH_BUILD=40 reject java_server_version_available pufferfish 1.21.10
reject java_server_version_available pufferfish 26.3
reject resolve_setup_java_runtime pufferfish pinned 26.3
# A single usable identity, official release, safe path and positive integer
# build are mandatory even when another archived JAR looks plausible.
for fixture in ambiguous nonrelease malformed zero negative string fractional absolute backslash; do
    reject resolve_setup_java_runtime pufferfish recommended ''
    reject java_server_version_available pufferfish 1.21.10
done

for fixture in reobf both; do
    java_server_version_available pufferfish 1.21.10
    selected="$(resolve_setup_java_runtime pufferfish recommended '')"
    jq -e '.game == "1.21.10" and .pufferfish_build == 39' <<< "${selected}" >/dev/null
done

fixture=no-jobs
reject resolve_setup_java_runtime pufferfish recommended ''
