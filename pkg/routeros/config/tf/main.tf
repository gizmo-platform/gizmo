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
{{- end }}
{{- end }}
