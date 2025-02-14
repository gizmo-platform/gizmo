terraform {
  required_providers {
    routeros = {
      source = "terraform-routeros/routeros"
      version = "1.75.0"
    }
  }
}

// Main FMS Router
provider "routeros" {
  hosturl  = "http://{{.RouterAddr}}"
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

  bootstrap = {{.RouterBootstrap}}
  fms_mac   = "{{.FMS.FMSMac}}"
}

{{- with $top := . }}
{{- range .FMS.Fields }}

// Field {{.ID}}
provider "routeros" {
  hosturl  = "http://{{.IP}}"
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

  bootstrap     = {{$top.FieldBootstrap}}
  field_id      = {{.ID}}
}
{{- end }}
{{- end }}
