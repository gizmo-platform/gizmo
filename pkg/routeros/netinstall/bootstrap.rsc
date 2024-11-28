{{.network}}
/user/set admin password={{.AdminPass}}
/user/group/add name=readonly policy=read,ssh,web
/user/add name={{.AutoUser}} group=full password={{.AutoPass}}
/user/add name={{.ViewUser}} group=readonly password={{.ViewPass}}
/ip service
set www disabled=no
