controlgroup wheel
hostname
vendorclassid
option domain_name_servers, domain_name, domain_search
option classless_static_routes
option interface_mtu
option rapid_commit
require dhcp_server_identifier
nodelay
noipv6

allowinterfaces br0

interface br0
fallback local
profile local
  static ip_address={{ .Team|ip4prefix }}.1/24
  noipv6
  release
