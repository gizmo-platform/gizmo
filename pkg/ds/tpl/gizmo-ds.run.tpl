#!/bin/sh

exec 2>&1
exec /usr/local/bin/gizmo ds run /boot/gsscfg.json
