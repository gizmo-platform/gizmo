# Input Section - Traffic from Internet

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

resource "routeros_ip_firewall_nat" "dstnat" {
  chain        = "dstnat"
  action       = "dst-nat"
  in_interface = routeros_interface_vlan.vlan_infra["wan0"].name
  to_addresses = "100.64.0.2"
  protocol     = "tcp"
  dst_port     = 22
}
