variable "dezgo_key" {
  type        = string
  description = "Dezgo.com API key"
  validation {
    condition     = length(var.dezgo_key) > 6 && substr(var.dezgo_key, 0, 6) == "DEZGO-"
    error_message = "Must start with 'DEZGO-'"
  }
  sensitive = true
}

resource "aws_ssm_parameter" "dezgo_key" {
  name            = "/kittenbot/dezgo-key"
  type            = "SecureString"
  tier            = "Standard"
  allowed_pattern = "^DEZGO-.+$"
  value           = var.dezgo_key
}
