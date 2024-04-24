locals {
  tlm  = jsondecode(file("${path.root}/tlm.json"))
  fmap = lookup(local.tlm, var.field_id, {})
}

resource "routeros_interface_bridge_port" "trunk" {
  bridge    = routeros_interface_bridge.br0.name
  interface = "ether1"
  pvid      = 10
  disabled  = var.bootstrap
}

resource "routeros_interface_bridge_port" "access" {
  for_each = toset(formatlist("ether%d", [2, 3, 4, 5]))

  bridge    = routeros_interface_bridge.br0.name
  interface = each.key
  pvid      = lookup(local.fmap, each.key, routeros_interface_vlan.dump.vlan_id)
}
