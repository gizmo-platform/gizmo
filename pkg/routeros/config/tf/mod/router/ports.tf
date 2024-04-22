resource "routeros_interface_bridge_port" "internal" {
  for_each = toset(formatlist("ether%d", [2, 3, 4, 5]))

  bridge    = routeros_interface_bridge.br0.name
  interface = each.key
  pvid      = routeros_interface_vlan.vlan_infra["fms0"].vlan_id
}

resource "routeros_interface_bridge_vlan" "trunks" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = join(",", sort([for vlan in routeros_interface_vlan.vlan_team : tostring(vlan.vlan_id)]))
  tagged   = formatlist("ether%d", [3, 4, 5])
}

resource "routeros_interface_bridge_port" "wan" {
  bridge    = routeros_interface_bridge.br0.name
  interface = "ether1"
  pvid      = routeros_interface_vlan.vlan_infra["wan0"].vlan_id
}

resource "routeros_interface_bridge_vlan" "wan" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = tostring(routeros_interface_vlan.vlan_infra["wan0"].vlan_id)
  untagged = ["ether1"]
}

resource "routeros_interface_bridge_vlan" "peer" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = tostring(routeros_interface_vlan.vlan_infra["peer0"].vlan_id)
  tagged   = ["ether1"]
}

resource "routeros_interface_ethernet" "poe_ports" {
  for_each = toset(formatlist("ether%d", [3, 4, 5]))

  factory_name = each.key
  name         = each.key
  poe_out      = "auto-on"
}
