resource "routeros_ip_pool" "pool_infra" {
  name    = "fms"
  ranges  = ["100.64.0.1-100.64.0.30"]
  comment = "FMS Default IP Pool"
}

resource "routeros_ip_dhcp_server" "server_infra" {
  interface          = routeros_interface_vlan.vlan_infra["fms0"].name
  name               = "FMS"
  address_pool       = routeros_ip_pool.pool_infra.name
  comment            = "FMS Default DHCP Server"
  conflict_detection = true
  lease_time         = "1h"
}

resource "routeros_ip_dhcp_server_network" "network_infra" {
  address    = "100.64.0.0/24"
  comment    = "Options for FMS"
  gateway    = "100.64.0.1"
  domain     = "gizmo"
  dns_server = ["100.64.0.1"]
}

resource "routeros_ip_dhcp_server_lease" "field" {
  for_each = local.fms.Fields

  address     = each.value.IP
  mac_address = each.value.MAC
  comment     = format("Field %d", each.value.ID)
  server      = routeros_ip_dhcp_server.server_infra.name
}

resource "routeros_ip_dhcp_server_lease" "fms" {
  address     = "100.64.0.2"
  mac_address = var.fms_mac
  comment     = "FMS"
  server      = routeros_ip_dhcp_server.server_infra.name
}

resource "routeros_ip_dns_record" "fms" {
  name    = "fms.gizmo"
  address = "100.64.0.2"
  type    = "A"
}
