/interface/vlan/add comment="Bootstrap Interface" interface=ether2 name=bootstrap0 vlan-id=2
/ip/address/add address=100.64.1.1/24 interface=bootstrap0
/certificate
add name=ca common-name=local_ca key-usage=key-cert-sign
add name=self common-name=localhost
sign ca
sign self
/ip service
set www disabled=no
set www-ssl certificate=self disabled=no
/user/group/add name=readonly policy=read,ssh,web
/user/add name={{.AutoUser}} group=full password={{.AutoPass}}
/user/add name={{.ViewUser}} group=readonly password={{.ViewPass}}
