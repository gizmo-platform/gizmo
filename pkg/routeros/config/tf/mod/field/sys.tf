resource "routeros_system_identity" "identity" {
  name = "gizmo-field-${var.field_id}"
}

resource "routeros_interface_list" "internal" {
  name = "internal"
}

resource "routeros_interface_list_member" "ether1" {
  interface = "ether1"
  list      = routeros_interface_list.internal.name
}

resource "routeros_tool_mac_server" "mac_server" {
  allowed_interface_list = routeros_interface_list.internal.name
}

resource "routeros_ip_service" "disabled" {
  for_each = {
    api-ssl = 8729
    api     = 8278
    ftp     = 21
    telnet  = 21
    winbox  = 8291
    www     = 80
  }

  numbers  = each.key
  port     = each.value
  disabled = true
}
