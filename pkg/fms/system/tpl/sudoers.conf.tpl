%wheel ALL=(ALL:ALL) ALL

# This allows the gizmo binary to invoke certain commands in
# non-interactive mode when called by the webserver, which is
# necessary to avoid a complex prompting scheme to pass the user
# credentials inwards.
_gizmo ALL=(root) NOPASSWD:/usr/bin/tzupdate
_gizmo ALL=(root) NOPASSWD:/usr/bin/setcap cap_net_raw\,cap_net_bind_service+ep /var/lib/gizmo/bin/netinstall-cli

# This allows the 'admin' user to transparently gain authority, as
# most users of the Gizmo platform are not expected to be seasoned
# administrators with understanding of what to do when greeted by a
# sudo prompt.
admin ALL=(root) NOPASSWD:ALL
