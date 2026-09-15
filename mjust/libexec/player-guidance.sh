#!/usr/bin/bash

physical_ram_gib() {
    local mem_kib mem_mib
    mem_kib="$(awk '/^MemTotal:/ {print $2}' /proc/meminfo 2>/dev/null || echo 0)"
    [[ ${mem_kib} =~ ^[0-9]+$ ]] || mem_kib=0
    mem_mib=$((mem_kib / 1024))
    # Round to the nearest nominal GiB so a normal 8 GiB machine is not shown
    # as "7 GiB" only because firmware/kernel reservations reduce MemTotal.
    printf '%d' $(((mem_mib + 512) / 1024))
}

player_limit_bucket() {
    local value="$1" number
    [[ ${value} =~ ^[1-9][0-9]*$ ]] || return 1
    if (( ${#value} > 2 )); then
        printf 'high'
        return 0
    fi
    number=$((10#${value}))
    if (( number <= 5 )); then
        printf 'five'
    elif (( number <= 10 )); then
        printf 'ten'
    elif (( number <= 20 )); then
        printf 'twenty'
    else
        printf 'high'
    fi
}

print_player_ram_guidance() {
    cat <<'EOF'
General JustVoxel guidance:
  Up to 5 players   - 8 GiB RAM can work; 12 GiB is more comfortable
  Up to 10 players  - 12-16 GiB RAM recommended
  Up to 20 players  - 16 GiB+ RAM recommended
  More than 20      - workload dependent

World size, plugins, Geyser/Bedrock and view/simulation distance can increase
resource requirements. This is guidance only; JustVoxel does not restrict the
maximum-player value you choose.
EOF
}

print_player_ram_notice() {
    local max_players="$1" ram_gib="$2" bucket
    bucket="$(player_limit_bucket "${max_players}")" || return 1

    echo "Requested maximum players: ${max_players}"
    echo "Physical RAM: ${ram_gib} GiB"
    echo

    case "${bucket}" in
        five)
            if (( ram_gib < 8 )); then
                echo 'Notice: this is below the usual 8 GiB guidance for up to 5 players.'
            elif (( ram_gib < 12 )); then
                echo 'RAM guidance: this can work; around 12 GiB is more comfortable.'
            else
                echo 'RAM guidance: suitable for this player range.'
            fi
            ;;
        ten)
            if (( ram_gib < 12 )); then
                echo 'Notice: JustVoxel normally recommends 12-16 GiB RAM for up to 10 players.'
            else
                echo 'RAM guidance: within the normal range for up to 10 players.'
            fi
            ;;
        twenty)
            if (( ram_gib < 16 )); then
                echo 'Notice: JustVoxel normally recommends 16 GiB or more for up to 20 players.'
            else
                echo 'RAM guidance: meets the normal baseline for this player range.'
            fi
            ;;
        high)
            echo 'Notice: this is a high player limit. Performance becomes increasingly dependent'
            echo 'on CPU performance, world activity, plugins, view distance and server tuning.'
            ;;
    esac
}
