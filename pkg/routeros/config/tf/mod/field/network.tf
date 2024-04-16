locals {
  fms = jsondecode(file("${path.root}/fms.json"))
}

data "routeros_interfaces" "ether1" {
  filter = {
    name = "ether1"
  }
}

resource "routeros_interface_bridge" "br0" {
  name           = "br0"
  vlan_filtering = true
  frame_types    = "admit-only-vlan-tagged"
  auto_mac       = false
  admin_mac      = data.routeros_interfaces.ether1.interfaces[0].mac_address
}

resource "routeros_interface_vlan" "vlan_team" {
  for_each = local.fms.Teams

  name      = format("team%d", each.key)
  interface = routeros_interface_bridge.br0.name
  vlan_id   = each.value.VLAN
  comment   = each.value.Name
}

resource "routeros_interface_vlan" "fms" {
  name      = "fms0"
  interface = routeros_interface_bridge.br0.name
  vlan_id   = 10
  comment   = "FMS Network"
}

resource "routeros_interface_bridge_vlan" "team" {
  bridge = routeros_interface_bridge.br0.name
  vlan_ids = join(",", sort(flatten([
    [for vlan in routeros_interface_vlan.vlan_team : vlan.vlan_id],
    [routeros_interface_vlan.fms.vlan_id],
  ])))
  tagged  = [routeros_interface_bridge.br0.name]
  comment = "Bridge Networks"
}

resource "routeros_ip_dhcp_client" "upstream" {
  interface         = routeros_interface_vlan.fms.name
  add_default_route = "yes"
  use_peer_ntp      = false
  use_peer_dns      = false
  comment           = "Internal Upstream"
}
