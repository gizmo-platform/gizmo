#!/bin/sh

exec 2>&1
exec /usr/bin/gizmo ds run /boot/gsscfg.json
