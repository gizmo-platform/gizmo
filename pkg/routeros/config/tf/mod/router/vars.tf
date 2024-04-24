variable "bootstrap" {
  type        = bool
  default     = false
  description = "Bootstrap mode, affects some resource creation"
}

variable "fms_mac" {
  type        = string
  description = "MAC address of the FMS machine."
}
