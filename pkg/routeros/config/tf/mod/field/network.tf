locals {
  fms = jsondecode(file("${path.root}/fms.json"))
}

data "routeros_interfaces" "ether1" {
  filter = {
    name = "ether1"
  }
}

resource "routeros_interface_bridge" "br0" {
  name              = "br0"
  frame_types       = "admit-only-vlan-tagged"
  vlan_filtering    = true
  ingress_filtering = true
  auto_mac          = false
  admin_mac         = data.routeros_interfaces.ether1.interfaces[0].mac_address
}

resource "routeros_interface_vlan" "fms" {
  name      = "fms0"
  interface = routeros_interface_bridge.br0.name
  vlan_id   = 10
  comment   = "FMS Network"
}

resource "routeros_interface_vlan" "dump" {
  name      = "dump0"
  interface = routeros_interface_bridge.br0.name
  vlan_id   = 450
  comment   = "Empty dump network"
}

resource "routeros_interface_bridge_vlan" "fms" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = routeros_interface_vlan.fms.vlan_id
  untagged = ["ether1"]
  tagged   = [routeros_interface_bridge.br0.name]
  comment  = "Uplink"
}

resource "routeros_interface_bridge_vlan" "team" {
  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = join(",", sort([for t in local.fms.Teams : t.VLAN]))
  tagged   = ["ether1", routeros_interface_bridge.br0.name]
  comment  = "Team access VLAN"
}

resource "routeros_ip_dhcp_client" "upstream" {
  interface         = routeros_interface_vlan.fms.name
  add_default_route = "yes"
  use_peer_ntp      = false
  use_peer_dns      = false
  script            = "{/ip/dhcp-client/set 0 disabled=yes}"
  comment           = "Internal Upstream"
}
