variable "bootstrap" {
  type        = bool
  default     = false
  description = "Bootstrap mode, affects some resource creation"
}

variable "field_id" {
  type        = number
  description = "Number associated with this field"
}
