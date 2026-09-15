#!/usr/bin/bash

jui_is_interactive() {
    [[ -t 0 && -t 1 && ${TERM:-dumb} != dumb ]]
}

# Nested menus commonly run inside command substitution so their stdout can
# return exactly one selected value. In that case stdout is a pipe even though
# the administrator still has a real controlling terminal. Use /dev/tty for
# nested keyboard/display interaction instead of treating captured stdout as
# noninteractive.
jui_has_tty() {
    [[ ${TERM:-dumb} != dumb ]] || return 1
    { : </dev/tty >/dev/tty; } 2>/dev/null
}

jui_terminal_cols() {
    local cols
    cols="$(tput cols 2>/dev/null || true)"
    [[ ${cols} =~ ^[0-9]+$ ]] || cols=80
    printf '%s' "${cols}"
}

jui_terminal_lines() {
    local lines
    lines="$(tput lines 2>/dev/null || true)"
    [[ ${lines} =~ ^[0-9]+$ ]] || lines=24
    printf '%s' "${lines}"
}

jui_rich_supported() {
    jui_is_interactive || return 1
    command -v fzf >/dev/null 2>&1 || return 1
    (( $(jui_terminal_cols) >= 90 && $(jui_terminal_lines) >= 20 ))
}

jui_backend() {
    if ! jui_has_tty; then
        printf 'none'
    elif command -v gum >/dev/null 2>&1; then
        printf 'gum'
    elif command -v fzf >/dev/null 2>&1; then
        printf 'fzf'
    else
        printf 'bash'
    fi
}

jui_choose() {
    local prompt="$1"
    shift
    local -a options=("$@")
    local backend selected index choice

    (( ${#options[@]} > 0 )) || return 2
    backend="$(jui_backend)"

    case "${backend}" in
        gum)
            gum choose --header "${prompt}" "${options[@]}" </dev/tty
            ;;
        fzf)
            printf '%s\n' "${options[@]}" \
                | fzf --height='~45%' --layout=reverse --border \
                    --prompt="${prompt} > " --no-multi
            ;;
        bash)
            printf '%s\n' "${prompt}" >/dev/tty
            for index in "${!options[@]}"; do
                printf '  %d. %s\n' "$((index + 1))" "${options[index]}" >/dev/tty
            done
            while true; do
                printf 'Selection: ' >/dev/tty
                IFS= read -r choice </dev/tty
                if [[ ${choice} =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#options[@]} )); then
                    selected="${options[choice - 1]}"
                    printf '%s' "${selected}"
                    return 0
                fi
                echo 'Please enter a valid number.' >/dev/tty
            done
            ;;
        *)
            return 2
            ;;
    esac
}

jui_confirm() {
    local prompt="$1" backend answer selected
    backend="$(jui_backend)"
    case "${backend}" in
        gum)
            gum confirm "${prompt}" </dev/tty
            ;;
        fzf)
            selected="$(printf 'No\nYes\n' | fzf --height='~20%' --layout=reverse --border --prompt="${prompt} > " --no-multi || true)"
            [[ ${selected} == Yes ]]
            ;;
        bash)
            printf '%s [y/N]: ' "${prompt}" >/dev/tty
            IFS= read -r answer </dev/tty
            [[ ${answer,,} == y || ${answer,,} == yes ]]
            ;;
        *)
            return 1
            ;;
    esac
}

jui_input() {
    local prompt="$1" default="${2:-}" backend value
    backend="$(jui_backend)"
    case "${backend}" in
        gum)
            gum input --prompt "${prompt}: " --value "${default}" </dev/tty
            ;;
        fzf|bash)
            printf '%s%s: ' "${prompt}" "${default:+ [${default}]}" >/dev/tty
            IFS= read -r value </dev/tty
            printf '%s' "${value:-${default}}"
            ;;
        *)
            return 2
            ;;
    esac
}

jui_pause() {
    jui_has_tty || return 0
    echo >/dev/tty
    printf 'Press Enter to return to the JustVoxel menu...' >/dev/tty
    IFS= read -r _ </dev/tty
}
