keep-in-foreground
log-facility=-
port=53
listen-address={{.Team|ip4prefix}}.1
dhcp-option=option:T1,3600
dhcp-option=option:T2,3700
dhcp-option=option:netmask,255.255.0.0
dhcp-range={{.Team|ip4prefix}}.2,{{.Team|ip4prefix}}.10
host-record=gizmo-ds,{{.Team|ip4prefix}}.1
host-record=ds.gizmo,{{.Team|ip4prefix}}.1
