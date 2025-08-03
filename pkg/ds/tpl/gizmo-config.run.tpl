#!/bin/sh

exec 2>&1
exec /usr/bin/gizmo ds config-server /boot/gsscfg.json
