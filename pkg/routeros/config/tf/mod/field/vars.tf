variable "bootstrap" {
  type        = bool
  default     = false
  description = "Bootstrap mode, affects some resource creation"
}

variable "field_id" {
  type        = number
  description = "Number associated with this field"
}

variable "infra_visible" {
  type        = bool
  default     = true
  description = "Make infrastructure network visible"
}

variable "infra_ssid" {
  type        = string
  default     = "gizmo"
  description = "Infrastructure SSID"
}

variable "infra_psk" {
  type        = string
  description = "PSK for the infrastructure SSID"
}

variable "gizmo_channel" {
  type        = string
  description = "Channel for the Gizmo network"
}
