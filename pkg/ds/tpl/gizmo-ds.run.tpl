#!/bin/sh

/usr/bin/sv check dhcpcd || exit 1

exec 2>&1
exec /usr/local/bin/gizmo ds run /boot/gsscfg.json
