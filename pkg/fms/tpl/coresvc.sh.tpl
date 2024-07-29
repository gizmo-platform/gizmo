#!/bin/sh

msg "Fix permissions on Gizmo Tools"
setcap cap_net_admin+ep /usr/local/bin/gizmo

msg "Configuring system for Gizmo"
/usr/local/bin/gizmo fms system-setup
