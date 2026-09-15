#!/usr/bin/bash

jui_is_interactive() {
    [[ -t 0 && -t 1 && ${TERM:-dumb} != dumb ]]
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
    if ! jui_is_interactive; then
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
            gum choose --header "${prompt}" "${options[@]}"
            ;;
        fzf)
            printf '%s\n' "${options[@]}" \
                | fzf --height='~45%' --layout=reverse --border \
                    --prompt="${prompt} > " --no-multi
            ;;
        bash)
            printf '%s\n' "${prompt}" >&2
            for index in "${!options[@]}"; do
                printf '  %d. %s\n' "$((index + 1))" "${options[index]}" >&2
            done
            while true; do
                read -r -p 'Selection: ' choice
                if [[ ${choice} =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#options[@]} )); then
                    selected="${options[choice - 1]}"
                    printf '%s' "${selected}"
                    return 0
                fi
                echo 'Please enter a valid number.' >&2
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
            gum confirm "${prompt}"
            ;;
        fzf)
            selected="$(printf 'No\nYes\n' | fzf --height='~20%' --layout=reverse --border --prompt="${prompt} > " --no-multi || true)"
            [[ ${selected} == Yes ]]
            ;;
        bash)
            read -r -p "${prompt} [y/N]: " answer
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
            gum input --prompt "${prompt}: " --value "${default}"
            ;;
        fzf|bash)
            read -r -p "${prompt}${default:+ [${default}]}: " value
            printf '%s' "${value:-${default}}"
            ;;
        *)
            return 2
            ;;
    esac
}

jui_pause() {
    jui_is_interactive || return 0
    echo
    read -r -p 'Press Enter to return to the JustVoxel menu...' _
}
