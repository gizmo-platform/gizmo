resource "routeros_interface_bridge_port" "fms" {
  bridge    = routeros_interface_bridge.br0.name
  interface = "ether2"
  pvid      = routeros_interface_vlan.vlan_infra["fms0"].vlan_id
}

resource "routeros_interface_bridge_vlan" "fms" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = tostring(routeros_interface_vlan.vlan_infra["fms0"].vlan_id)
  untagged = formatlist("ether%d", [2, 3, 4, 5])
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

resource "routeros_interface_bridge_vlan" "field_teams" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = join(",", [for vlan in routeros_interface_vlan.vlan_team : vlan.vlan_id])
  tagged   = formatlist("ether%d", [3, 4, 5])
}
