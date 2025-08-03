#!/bin/sh

msg "Fix permissions on Gizmo Tools"
setcap cap_net_admin+ep /usr/bin/gizmo

msg "Configuring system for Gizmo"
/usr/bin/gizmo fms system-setup
