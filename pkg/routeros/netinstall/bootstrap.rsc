{{.network}}
/certificate
add name=ca common-name=local_ca key-usage=key-cert-sign
add name=self common-name=localhost
sign ca
sign self
/ip service
set www disabled=no
set www-ssl certificate=self disabled=no
/user/set admin password={{.AdminPass}}
/user/group/add name=readonly policy=read,ssh,web
/user/add name={{.AutoUser}} group=full password={{.AutoPass}}
/user/add name={{.ViewUser}} group=readonly password={{.ViewPass}}
