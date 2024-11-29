resource "routeros_interface_bridge_port" "internal" {
  for_each = toset(formatlist("ether%d", [2, 3, 4, 5]))

  bridge    = routeros_interface_bridge.br0.name
  interface = each.key
  pvid      = routeros_interface_vlan.vlan_infra["fms0"].vlan_id
}

resource "routeros_interface_bridge_vlan" "fms" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = [tostring(routeros_interface_vlan.vlan_infra["fms0"].vlan_id)]
  tagged   = flatten([routeros_interface_bridge.br0.name, data.routeros_interfaces.sfp1.interfaces[*].name])
  untagged = formatlist("ether%d", [2, 3, 4, 5])
}

resource "routeros_interface_bridge_vlan" "team" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = sort([for vlan in routeros_interface_vlan.vlan_team : vlan.vlan_id])
  tagged = flatten([
    [routeros_interface_bridge.br0.name],
    formatlist("ether%d", [3, 4, 5]),
  ])
}

resource "routeros_interface_bridge_port" "wan" {
  bridge    = routeros_interface_bridge.br0.name
  interface = "ether1"
  pvid      = routeros_interface_vlan.vlan_infra["wan0"].vlan_id
}

resource "routeros_interface_bridge_vlan" "wan" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = [tostring(routeros_interface_vlan.vlan_infra["wan0"].vlan_id)]
  tagged   = flatten([routeros_interface_bridge.br0.name, data.routeros_interfaces.sfp1.interfaces[*].name])
  untagged = ["ether1"]
}

resource "routeros_interface_ethernet" "poe_ports" {
  for_each = toset(formatlist("ether%d", [3, 4, 5]))

  factory_name = each.key
  name         = each.key
  poe_out      = "auto-on"
}

# This is all related to fairly advanced usage where we want the FMS
# network to be part of a much larger routed fabric.  This really only
# makes sense at a handful of very large hubs and championships.
data "routeros_interfaces" "sfp1" {
  filter = {
    name = "sfp1"
  }
}

resource "routeros_interface_bridge_port" "sfp1" {
  count = length(data.routeros_interfaces.sfp1.interfaces)

  bridge    = routeros_interface_bridge.br0.name
  interface = data.routeros_interfaces.sfp1.interfaces[count.index].name
  pvid      = 1
}
