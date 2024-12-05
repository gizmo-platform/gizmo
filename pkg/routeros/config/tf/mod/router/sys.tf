resource "routeros_system_identity" "identity" {
  name = "gizmo-edge"
}

resource "routeros_ip_service" "disabled" {
  for_each = {
    api-ssl = 8729
    api     = 8278
    ftp     = 21
    telnet  = 21
    winbox  = 8291
    www-ssl = 443
  }

  numbers  = each.key
  port     = each.value
  disabled = true
}

resource "routeros_dns" "dns" {
  allow_remote_requests = true
  servers               = lookup(local.fms, "FixedDNS", [])
}
