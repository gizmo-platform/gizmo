#!/bin/sh

[ -r ./conf ] && . ./conf
cd /var/lib/gizmo || exit 1
[ -r fms.json ] || echo '{}' | chpst -u _gizmo tee fms.json > /dev/null
exec chpst -u _gizmo /usr/local/bin/gizmo fms run
