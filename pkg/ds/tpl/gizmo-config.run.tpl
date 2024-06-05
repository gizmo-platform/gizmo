#!/bin/sh

exec 2>&1
exec /usr/local/bin/gizmo ds config-server /boot/gsscfg.json
