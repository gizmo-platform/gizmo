#!/bin/sh

[ -r ./conf ] && . ./conf
cd /var/lib/gizmo || exit 1
[ -r fms.json ] || echo '{}' | chpst -u _gizmo tee fms.json > /dev/null
export USER=_gizmo
export HOME=/var/lib/gizmo
exec 2>&1
exec chpst -u _gizmo:wheel:dialout /usr/local/bin/gizmo fms run
