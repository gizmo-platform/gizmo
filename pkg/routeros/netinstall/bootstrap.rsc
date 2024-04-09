/ip dhcp-client
add comment=defconf interface=ether1
{{.network}}
/ip/dns
set servers=8.8.8.8
/system clock
set time-zone-name=America/Chicago
/system note
set show-at-login=no
/system/ntp/client
set enabled=yes
/system/ntp/client/servers
add address=pool.ntp.org
/certificate
add name=ca common-name=local_ca key-usage=key-cert-sign
add name=self common-name=localhost
sign ca
sign self
/ip service
set www disabled=no
set www-ssl certificate=self disabled=no
/user/group/add name=automation policy=read,write,rest-api,ssh
/user/group/add name=readonly policy=read,ssh,web
/user/add name={{.AutoUser}} group=automation password={{.AutoPass}}
/user/add name={{.ViewUser}} group=readonly password={{.ViewPass}}
