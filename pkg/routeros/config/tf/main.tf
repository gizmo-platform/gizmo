terraform {
  required_providers {
    routeros = {
      source = "terraform-routeros/routeros"
    }
  }
}

// Main FMS Router
provider "routeros" {
  hosturl  = "https://{{.RouterAddr}}"
  alias    = "router"
  insecure = true
  username = "{{.FMS.AutoUser}}"
  password = "{{.FMS.AutoPass}}"
}

module "router" {
  source = "./mod/router"

  providers = {
    routeros = routeros.router
  }

  bootstrap = {{.Bootstrap}}
  fms_mac   = "{{.FMS.FMSMac}}"
}

{{- with $top := . }}
{{- range .FMS.Fields }}

// Field {{.ID}}
provider "routeros" {
  hosturl  = "https://{{.IP}}"
  alias    = "field{{.ID}}"
  insecure = true
  username = "{{$top.FMS.AutoUser}}"
  password = "{{$top.FMS.AutoPass}}"
}

module "field{{.ID}}" {
  source = "./mod/field"

  providers = {
    routeros = routeros.field{{.ID}}
  }

  bootstrap     = {{$top.Bootstrap}}
  field_id      = {{.ID}}
  infra_visible = {{$top.FMS.InfrastructureVisible}}
  infra_ssid    = "{{$top.FMS.InfrastructureSSID}}"
  infra_psk     = "{{$top.FMS.InfrastructurePSK}}"
}
{{- end }}
{{- end }}
