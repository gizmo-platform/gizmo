// This works out what internal interfaces exist, and allows this same
// module to scale seamlessly between devices with different numbers of
// ports.
data "routeros_interfaces" "ether" {
  filter = {
    type = "ether"
  }
}

locals {
  reserved_ifaces = formatlist("ether%d", [1, 2])
  internal_enet = [ for iface in data.routeros_interfaces.ether.interfaces : iface.name if !(contains(local.reserved_ifaces, iface.name) || startswith(iface.name, "sfp")) ]
  trunk_sfp = [ for iface in data.routeros_interfaces.ether.interfaces : iface.name if startswith(iface.name, "sfp") ]
}

resource "routeros_interface_bridge_port" "fms" {
  disabled = var.bootstrap

  bridge    = routeros_interface_bridge.br0.name
  interface = "ether2"
  pvid      = routeros_interface_vlan.vlan_infra["fms0"].vlan_id
}

resource "routeros_interface_bridge_port" "internal" {
  for_each = toset(local.internal_enet)

  bridge    = routeros_interface_bridge.br0.name
  interface = each.key
  pvid      = routeros_interface_vlan.vlan_infra["fms0"].vlan_id
}

resource "routeros_interface_bridge_vlan" "internal" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = [tostring(routeros_interface_vlan.vlan_infra["fms0"].vlan_id)]
  tagged   = flatten([routeros_interface_bridge.br0.name, local.trunk_sfp])
  untagged = flatten([routeros_interface_bridge_port.fms.interface, local.internal_enet])
}

resource "routeros_interface_bridge_vlan" "team" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = sort([for vlan in routeros_interface_vlan.vlan_team : vlan.vlan_id])
  tagged = flatten([
    [routeros_interface_bridge.br0.name],
    local.internal_enet
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
  tagged   = flatten([routeros_interface_bridge.br0.name, local.trunk_sfp])
  untagged = ["ether1"]
}

resource "routeros_interface_ethernet" "poe_ports" {
  for_each = toset(formatlist("ether%d", [3, 4, 5]))

  factory_name = each.key
  name         = each.key
  poe_out      = "auto-on"
}

resource "routeros_interface_bridge_port" "trunk_sfp" {
  for_each = toset(local.trunk_sfp)

  bridge    = routeros_interface_bridge.br0.name
  interface = each.key
  pvid      = 1
}
