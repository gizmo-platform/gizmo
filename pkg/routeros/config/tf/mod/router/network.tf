locals {
  fms = jsondecode(file("${path.root}/fms.json"))
  tlm = jsondecode(file("${path.root}/tlm.json"))

  bgp_vlan = local.fms.AdvancedBGPVLAN != 0 ? local.fms.AdvancedBGPVLAN : 410
}

resource "routeros_interface_bridge" "br0" {
  name           = "br0"
  vlan_filtering = true
  frame_types    = "admit-only-vlan-tagged"
}

resource "routeros_interface_list" "teams" {
  name = "teams"
}

resource "routeros_interface_list_member" "teams" {
  for_each = routeros_interface_vlan.vlan_team

  interface = each.value.name
  list      = routeros_interface_list.teams.name
}

resource "routeros_interface_vlan" "vlan_team" {
  for_each = local.fms.Teams

  name      = format("team%d", each.key)
  interface = routeros_interface_bridge.br0.name
  vlan_id   = each.value.VLAN
  comment   = each.value.Name
}

resource "routeros_interface_vlan" "vlan_infra" {
  for_each = {
    fms0  = { id = 400, description = "FMS Network" }
    wan0  = { id = 405, description = "Upstream Networks" }
    peer0 = { id = local.bgp_vlan, description = "Peer Networks" }
  }

  interface = routeros_interface_bridge.br0.name
  name      = each.key
  comment   = each.value.description
  vlan_id   = each.value.id
}

resource "routeros_ip_address" "team" {
  for_each = local.fms.Teams

  address   = format("%s/%d", cidrhost(each.value.CIDR, 1), split("/", each.value.CIDR)[1])
  interface = routeros_interface_vlan.vlan_team[each.key].name
}

resource "routeros_ip_address" "fms" {
  address   = "100.64.0.1/24"
  interface = routeros_interface_vlan.vlan_infra["fms0"].name
}
