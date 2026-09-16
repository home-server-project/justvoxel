#!/usr/bin/bash

jv_player_list_raw() {
    podman exec minecraft rcon-cli 'list' 2>/dev/null
}

jv_player_check_before_interrupt() {
    local action="${1:-interrupt the server}" players mode

    if ! systemctl is-active --quiet minecraft.service 2>/dev/null; then
        return 0
    fi

    players="$(jv_player_list_raw 2>/dev/null || true)"
    if [[ -z ${players} ]]; then
        echo 'ERROR: could not confirm player status through RCON. Refusing to interrupt Minecraft.' >&2
        return 1
    fi

    echo "${players}"
    if grep -q 'There are 0 of' <<< "${players}"; then
        return 0
    fi

    mode="${JV_INTERRUPT_CONFIRMATION_MODE:-interactive}"
    case "${mode}" in
        interactive)
            confirm "Players appear to be online. ${action} anyway?"
            ;;
        required)
            return 10
            ;;
        confirmed)
            return 0
            ;;
        *)
            echo "ERROR: invalid interruption confirmation mode: ${mode}" >&2
            return 2
            ;;
    esac
}

jv_stop_minecraft_for_system_action() {
    local action="${1:-continue}"
    if ! systemctl is-active --quiet minecraft.service 2>/dev/null; then
        return 0
    fi

    jv_player_check_before_interrupt "${action}"
    echo 'Stopping Minecraft through its configured graceful shutdown path.'
    systemctl stop minecraft.service
    if systemctl is-active --quiet minecraft.service 2>/dev/null; then
        echo 'ERROR: Minecraft is still running. Refusing the system power action.' >&2
        return 1
    fi
}
