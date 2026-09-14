# JustVoxel interactive Bash identity.
# Bright green is intentionally close to Minecraft's grass/creeper visual cue
# without changing terminal background or non-interactive shell output.
if [[ -n "${BASH_VERSION:-}" && $- == *i* ]]; then
    case "${TERM:-}" in
        dumb) ;;
        *) PS1='\[\e[38;5;82m\]\u@\h\[\e[0m\] \w \[\e[38;5;82m\]\$\[\e[0m\] ' ;;
    esac
fi
