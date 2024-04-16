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
  depends_on = [module.router]

  source = "./mod/field"

  providers = {
    routeros = routeros.field{{.ID}}
  }

  bootstrap = {{$top.Bootstrap}}
  field_id = {{.ID}}
}
{{- end }}
{{- end }}
