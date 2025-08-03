#!/bin/sh

[ -r ./conf ] && . ./conf
cd /var/lib/gizmo || exit 1
[ -r fms.json ] || echo '{}' | chpst -u _gizmo tee fms.json > /dev/null
[ -f .htpasswd ] || touch .htpasswd
[ -f .htgroup ] || touch .htgroup
export USER=_gizmo
export HOME=/var/lib/gizmo
exec 2>&1
exec chpst -u _gizmo:wheel:dialout /usr/bin/gizmo fms run
