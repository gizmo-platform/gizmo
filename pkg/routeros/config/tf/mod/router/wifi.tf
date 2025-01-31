locals {
  frequencies = {
    "Auto" = [],
    "1"    = [2412],
    "6"    = [2437],
    "11"   = [2462],
  }
}

resource "routeros_capsman_manager" "mgr" {
  enabled        = true
  upgrade_policy = "require-same-version"
  certificate    = "auto"
  ca_certificate = "auto"
}

resource "routeros_capsman_channel" "gizmo" {
  name = "gizmo-2ghz"
  band = "2ghz-g/n"
}

resource "routeros_capsman_channel" "aux" {
  name = "gizmo-5ghz"
  band = "5ghz-n/ac"
}

resource "routeros_capsman_security" "gizmo" {
  name                 = "gizmo"
  authentication_types = ["wpa2-psk"]
  passphrase           = local.fms.InfrastructurePSK
  encryption           = ["tkip", "aes-ccm"]
}

resource "routeros_capsman_datapath" "gizmo" {
  name    = "gizmo"
  vlan_id = routeros_interface_vlan.vlan_infra["fms0"].vlan_id
  bridge  = routeros_interface_bridge.br0.name

  local_forwarding = true
  vlan_mode        = "use-tag"
}

resource "routeros_capsman_configuration" "gizmo" {
  for_each = {
    gizmo-2ghz = routeros_capsman_channel.gizmo
    gizmo-5ghz = routeros_capsman_channel.aux
  }

  name             = each.key
  ssid             = local.fms.InfrastructureSSID
  hide_ssid        = !local.fms.InfrastructureVisible
  mode             = "ap"
  country          = "united states3"
  installation     = "indoor"
  distance         = "indoors"
  keepalive_frames = "enabled"

  channel = {
    config = each.value.name
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

  local_forwarding            = true
  client_to_client_forwarding = true
  vlan_mode                   = "use-tag"
}

resource "routeros_capsman_configuration" "team" {
  for_each = local.fms.Teams

  name             = format("team%d", each.key)
  ssid             = each.value.SSID
  hide_ssid        = true
  mode             = "ap"
  country          = "united states3"
  installation     = "indoor"
  distance         = "indoors"
  keepalive_frames = "enabled"

  channel = {
    config = routeros_capsman_channel.gizmo.name
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
  hw_supported_modes   = ["ac"]
}

resource "routeros_capsman_provisioning" "gizmo_2ghz" {
  for_each = local.fms.Fields

  comment         = "gizmo-field-${each.value.ID}"
  identity_regexp = "gizmo-field-${each.value.ID}"

  disabled             = (local.fms.RadioMode != "FIELD")
  master_configuration = routeros_capsman_configuration.gizmo["gizmo-2ghz"].name
  slave_configurations = [for cfg in routeros_capsman_configuration.team :
    cfg.name if contains(values(lookup(local.tlm, each.value.ID, {})), routeros_capsman_datapath.team[replace(cfg.name, "team", "")].vlan_id)
  ]
  action             = "create-dynamic-enabled"
  hw_supported_modes = ["gn"]
}
