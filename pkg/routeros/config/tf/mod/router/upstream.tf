resource "routeros_ip_dhcp_client" "upstream" {
  interface         = routeros_interface_vlan.vlan_infra["wan0"].name
  add_default_route = "yes"
  use_peer_ntp      = false
  use_peer_dns      = false
  comment           = "External Upstream"
}
