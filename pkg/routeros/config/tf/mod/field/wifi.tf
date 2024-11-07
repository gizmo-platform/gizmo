resource "routeros_interface_wireless_cap" "settings" {
  enabled              = true
  caps_man_addresses   = ["100.64.0.2"]
  bridge               = routeros_interface_bridge.br0.name
  interfaces           = ["wlan1", "wlan2"]
  discovery_interfaces = [routeros_interface_vlan.fms.name]
}
