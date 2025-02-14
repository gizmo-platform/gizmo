resource "routeros_ip_address" "peer" {
  count     = local.fms.AdvancedBGPAS != 0 ? 1 : 0
  address   = local.fms.AdvancedBGPIP
  interface = routeros_interface_vlan.vlan_infra["peer0"].name
}

resource "routeros_interface_bridge_vlan" "peer" {
  count = local.fms.AdvancedBGPAS != 0 ? 1 : 0

  bridge   = routeros_interface_bridge.br0.name
  vlan_ids = [tostring(routeros_interface_vlan.vlan_infra["peer0"].vlan_id)]
  tagged   = flatten([routeros_interface_bridge.br0.name, local.trunk_sfp])
}

resource "routeros_routing_bgp_connection" "peer" {
  count = local.fms.AdvancedBGPAS != 0 ? 1 : 0

  as   = local.fms.AdvancedBGPAS
  name = "Peer"

  connect = true
  listen  = true

  router_id = "100.64.0.1"

  hold_time      = "30s"
  keepalive_time = "10s"

  local {
    role    = "ibgp"
    address = split("/", local.fms.AdvancedBGPIP)[0]
  }

  remote {
    address = local.fms.AdvancedBGPPeerIP
  }

  output {
    network = "nat_sources"
  }
}
