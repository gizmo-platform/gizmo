#!/bin/bash

# dialog colors
BLACK="\Z0"
RED="\Z1"
GREEN="\Z2"
YELLOW="\Z3"
BLUE="\Z4"
MAGENTA="\Z5"
CYAN="\Z6"
WHITE="\Z7"
BOLD="\Zb"
REVERSE="\Zr"
UNDERLINE="\Zu"
RESET="\Zn"

# Properties shared per widget.
MENULABEL="${BOLD}Use UP and DOWN keys to navigate \
menus. Use TAB to switch between buttons and ENTER to select.${RESET}"
INPUTSIZE="8 60"
INFOSIZE="6 60"
BACKTITLE="${BOLD}${WHITE}Gizmo Platform -- https://gizmoplatform.org ({{.Version}})${RESET}"

# global variables
USER="$(id -u -n)"
URL="http://localhost:8080/"
ANSWER=$(mktemp -t gizmo-welcome-XXXXXXXX || exit 1)

trap "DIE" INT TERM QUIT

DIALOG() {
    rm -f "$ANSWER"
    dialog --colors --keep-tite --no-shadow --no-mouse \
        --backtitle "$BACKTITLE" \
        --cancel-label "Back" --aspect 20 "$@" 2>"$ANSWER"
    return $?
}

INFOBOX() {
    dialog --colors --no-shadow --no-mouse \
        --backtitle "$BACKTITLE" \
        --title "${TITLE}" --aspect 20 --infobox "$@"
}

DIE() {
    rval=$1
    [ -z "$rval" ] && rval=0
    clear
    rm -r "$ANSWER"
    exit "$rval"
}

change_password() {
    local rv _firstpass _secondpass _again _desc

    while true; do
        if [ -z "${_firstpass}" ]; then
            _desc="Enter a new password for user $USER"
        else
            _again=" again"
        fi
        # shellcheck disable=SC2086
        DIALOG --insecure --passwordbox "${_desc}${_again}" ${INPUTSIZE}
        rv="$?"
        if [ "$rv" -eq 0 ]; then
            if [ -z "${_firstpass}" ]; then
                _firstpass="$(cat "$ANSWER")"
            else
                _secondpass="$(cat "$ANSWER")"
            fi
            if [ -n "${_firstpass}" ] && [ -n "${_secondpass}" ]; then
                if [ "${_firstpass}" != "${_secondpass}" ]; then
                    # shellcheck disable=SC2086
                    INFOBOX "Passwords do not match! Please enter again." ${INFOSIZE}
                    unset _firstpass _secondpass _again
                    sleep 2 && clear && continue
                fi
                echo "$USER:${_firstpass}" | sudo chpasswd -c SHA512
                sudo htpasswd -cb /var/lib/gizmo/.htpasswd "$USER" "${_firstpass}"
                sudo sv reload gizmo-fms
                # shellcheck disable=SC2086
                INFOBOX "Password updated for user $USER." ${INFOSIZE}
                sleep 2 && clear && break
            fi
        else
            return
        fi
    done
}

update_system() {
    local rv
    stdbuf -oL sudo bash -c "xbps-install -Sy xbps && xbps-install -Syu" 2>&1 | \
        DIALOG --title "Updating system packages..." \
        --programbox 24 90
    rv="$?"
    if [ "$rv" -ne 0 ]; then
        INFOBOX "Failed to update system packages (Code: $rv)"
        sleep 2 && clear
    fi
}

configure_wlan() {
    # shellcheck disable=SC2086
    INFOBOX "Launching Wi-Fi configuration menu..." ${INFOSIZE}
    iwgtk >/dev/null 2>&1 &
    disown
    sleep 2 && clear
}

launch_site() {
    # shellcheck disable=SC2086
    INFOBOX "Opening '$URL' in Firefox..." ${INFOSIZE}
    firefox "$URL" >/dev/null 2>&1 &
    disown
    sleep 15 && clear
}

reload_fms() {
    # shellcheck disable=SC2086
    INFOBOX "Restarting Gizmo FMS supervisory processes..." ${INFOSIZE}
    sudo sv restart gizmo-fms
    sleep 10 && clear
}

menu() {
    DIALOG --default-item "Password" \
        --title " Welcome to Gizmo " \
        --menu "$MENULABEL" 10 70 0 \
        "Password" "Change user password" \
        "Wi-Fi" "Connect to Wi-Fi" \
        "Update" "Update system packages" \
        "Launch" "Launch Gizmo Platform" \
        "Reload" "Reload Gizmo FMS Processes" \
        "Exit" "Exit Welcome"

    case "$(cat "$ANSWER")" in
        Password) change_password ;;
        Wi-Fi) configure_wlan ;;
        Update) update_system ;;
        Launch) launch_site ;;
        Reload) reload_fms ;;
        Exit) DIE 0 ;;
    esac
}

while true; do
    menu
done
