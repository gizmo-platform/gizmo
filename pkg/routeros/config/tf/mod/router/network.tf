locals {
  fms = jsondecode(file("${path.root}/fms.json"))
}

resource "routeros_interface_bridge" "br0" {
  name           = "br0"
  vlan_filtering = true
  frame_types    = "admit-only-vlan-tagged"
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
    fms0  = { id = 10, description = "FMS Network" }
    wan0  = { id = 20, description = "Upstream Networks" }
    peer0 = { id = 30, description = "Peer Networks" }
  }

  interface = routeros_interface_bridge.br0.name
  name      = each.key
  comment   = each.value.description
  vlan_id   = each.value.id
}

resource "routeros_interface_bridge_vlan" "br_vlan" {
  bridge = routeros_interface_bridge.br0.name
  vlan_ids = join(",", [for s in sort(formatlist("%03d", flatten([
    [for vlan in routeros_interface_vlan.vlan_team : vlan.vlan_id],
    [for vlan in routeros_interface_vlan.vlan_infra : vlan.vlan_id],
    ]))) : tonumber(s)]
  )
  tagged  = [routeros_interface_bridge.br0.name]
  comment = "Bridge Networks"
}

resource "routeros_interface_bridge_port" "vlan_team" {
  for_each = routeros_interface_vlan.vlan_team

  interface = each.value.name
  pvid      = each.value.vlan_id
  bridge    = routeros_interface_bridge.br0.name
  comment   = each.value.comment
}

resource "routeros_interface_bridge_port" "vlan_infra" {
  for_each = routeros_interface_vlan.vlan_infra

  interface = each.value.name
  pvid      = each.value.vlan_id
  bridge    = routeros_interface_bridge.br0.name
  comment   = each.value.comment
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

resource "routeros_ip_address" "peer" {
  count     = local.fms.AdvancedBGPAS != 0 ? 0 : 1
  address   = local.fms.AdvancedBGPIP
  interface = routeros_interface_vlan.vlan_infra["peer0"].name
}
