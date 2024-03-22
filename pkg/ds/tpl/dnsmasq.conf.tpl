keep-in-foreground
log-facility=-
port=0
listen-address={{.Team|ip4prefix}}.1
dhcp-option=option:T1,300
dhcp-option=option:T2,525
dhcp-option=option:netmask,255.255.0.0
dhcp-range={{.Team|ip4prefix}}.2,{{.Team|ip4prefix}}.10
