# Input Section - Traffic from Internet

resource "routeros_ip_firewall_addr_list" "bogons_v4" {
  for_each = {
    "0.0.0.0/8"       = "'This' network"
    "10.0.0.0/8"      = "RFC1918"
    "100.64.0.0/10"   = "CG-NAT"
    "127.0.0.0/8"     = "Loopback"
    "127.0.53.53"     = "Name collision occurence"
    "169.254.0.0/16"  = "Link Local"
    "172.16.0.0/12"   = "RFC1918"
    "192.0.0.0/24"    = "IETF Protocol Assignments"
    "192.0.2.0/24"    = "TEST-NET-1"
    "192.168.0.0/16"  = "RFC1918"
    "198.18.0.0/15"   = "Network interconect device benchmark testing"
    "198.51.100.0/24" = "TEST-NET-2"
    "203.0.113.0/24"  = "TEST-NET-3"
    "224.0.0.0/4"     = "Multicast"
    "240.0.0.0/4"     = "Reserved for future use"
    "255.255.255.255" = "Limited Broadcast"
  }

  address = each.key
  list    = "bogons_v4"
  comment = each.value
}

resource "routeros_ip_firewall_addr_list" "nat_sources" {
  list    = "nat_sources"
  address = "100.64.0.0/24"
  comment = "NAT Source Pool"
}

resource "routeros_ip_firewall_filter" "accept_established" {
  chain            = "input"
  action           = "accept"
  connection_state = "established,related,untracked"
  comment          = "accept-established"
  place_before     = routeros_ip_firewall_filter.default_drop.id
}

resource "routeros_ip_firewall_filter" "no_wan_bogons" {
  chain            = "input"
  action           = "drop"
  in_interface     = routeros_interface_vlan.vlan_infra["wan0"].name
  src_address_list = "bogons_v4"
  comment          = "deny-bogons-to-self"
  place_before     = routeros_ip_firewall_filter.default_drop.id
}

resource "routeros_ip_firewall_filter" "no_invalid" {
  chain            = "input"
  action           = "drop"
  connection_state = "invalid"
  comment          = "drop-invalid"
  place_before     = routeros_ip_firewall_filter.default_drop.id
}

resource "routeros_ip_firewall_filter" "accept_icmp" {
  chain        = "input"
  action       = "accept"
  protocol     = "icmp"
  place_before = routeros_ip_firewall_filter.default_drop.id
}

resource "routeros_ip_firewall_filter" "accept_peer" {
  chain        = "input"
  action       = "accept"
  in_interface = routeros_interface_vlan.vlan_infra["peer0"].name
  place_before = routeros_ip_firewall_filter.default_drop.id
}

resource "routeros_ip_firewall_filter" "accept_dns" {
  chain             = "input"
  action            = "accept"
  comment           = "accept-team-dns"
  dst_port          = 53
  protocol          = "udp"
  in_interface_list = routeros_interface_list.teams.name
  place_before      = routeros_ip_firewall_filter.default_drop.id
}

resource "routeros_ip_firewall_filter" "prevent_team_to_team" {
  chain              = "forward"
  action             = "drop"
  comment            = "prevent-team-to-team"
  in_interface_list  = routeros_interface_list.teams.name
  out_interface_list = routeros_interface_list.teams.name
  place_before       = routeros_ip_firewall_filter.default_drop.id
}

resource "routeros_ip_firewall_filter" "default_drop" {
  chain        = "input"
  action       = "drop"
  comment      = "default-deny"
  in_interface = "!${routeros_interface_vlan.vlan_infra["fms0"].name}"
  disabled     = var.bootstrap
}

resource "routeros_ip_firewall_nat" "srcnat" {
  chain            = "srcnat"
  action           = "masquerade"
  out_interface    = routeros_interface_vlan.vlan_infra["wan0"].name
  src_address_list = "nat_sources"
  comment          = "nat-masquerade"
}
