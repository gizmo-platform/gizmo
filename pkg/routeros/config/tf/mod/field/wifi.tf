resource "routeros_capsman_manager" "mgr" {
  enabled        = true
  upgrade_policy = "require-same-version"
}

resource "routeros_interface_wireless_cap" "settings" {
  discovery_interfaces = [routeros_interface_bridge.br0.name]
  bridge               = routeros_interface_bridge.br0.name
  enabled              = true
  interfaces           = ["wlan1", "wlan2"]
}

resource "routeros_capsman_channel" "channel" {
  for_each = {
    gizmo-2ghz = { band = "2ghz-g/n" }
    gizmo-5ghz = { band = "5ghz-n/ac" }
  }
  name = each.key
  band = each.value.band
}

resource "routeros_capsman_security" "gizmo" {
  name                 = "gizmo"
  authentication_types = ["wpa2-psk"]
  passphrase           = var.infra_psk
  encryption           = ["tkip", "aes-ccm"]
}

resource "routeros_capsman_datapath" "gizmo" {
  name    = "gizmo"
  vlan_id = routeros_interface_vlan.fms.vlan_id
  bridge  = routeros_interface_bridge.br0.name

  local_forwarding = true
  vlan_mode        = "use-tag"
}

resource "routeros_capsman_configuration" "gizmo" {
  for_each = routeros_capsman_channel.channel

  name      = each.key
  ssid      = var.infra_ssid
  hide_ssid = !var.infra_visible
  mode      = "ap"
  country   = "united states3"

  channel = {
    config = routeros_capsman_channel.channel[each.key].name
  }

  security = {
    config = routeros_capsman_security.gizmo.name
  }

  datapath = {
    config = routeros_capsman_datapath.gizmo.name
  }
}

resource "routeros_capsman_security" "team" {
  for_each = local.fms.Teams

  name                 = format("team%d", each.key)
  authentication_types = ["wpa2-psk"]
  passphrase           = each.value.PSK
  encryption           = ["tkip", "aes-ccm"]
}

resource "routeros_capsman_datapath" "team" {
  for_each = local.fms.Teams

  name    = format("team%d", each.key)
  vlan_id = each.value.VLAN
  bridge  = routeros_interface_bridge.br0.name

  local_forwarding = true
  vlan_mode        = "use-tag"
}

resource "routeros_capsman_configuration" "team" {
  for_each = local.fms.Teams

  name      = format("team%d", each.key)
  ssid      = each.value.SSID
  hide_ssid = true
  mode      = "ap"
  country   = "united states3"

  channel = {
    config = routeros_capsman_channel.channel["gizmo-2ghz"].name
  }

  security = {
    config = routeros_capsman_security.team[each.key].name
  }

  datapath = {
    config = routeros_capsman_datapath.team[each.key].name
  }
}

resource "routeros_capsman_provisioning" "gizmo_5ghz" {
  comment = "gizmo-5ghz"

  master_configuration = routeros_capsman_configuration.gizmo["gizmo-5ghz"].name
  action               = "create-dynamic-enabled"
  hw_supported_modes   = "ac"
}

resource "routeros_capsman_provisioning" "gizmo_2ghz" {
  comment = "gizmo-2ghz"

  master_configuration = routeros_capsman_configuration.gizmo["gizmo-2ghz"].name
  slave_configurations = join(",", [for cfg in routeros_capsman_configuration.team :
    cfg.name if contains(values(local.fmap), routeros_capsman_datapath.team[replace(cfg.name, "team", "")].vlan_id)
  ])
  action             = "create-dynamic-enabled"
  hw_supported_modes = "gn"
}
