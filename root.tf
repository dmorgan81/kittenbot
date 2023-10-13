terraform {
  required_version = ">= 1.5.0"

  cloud {
    organization = "kittenbot"
    workspaces {
      name = "main"
    }
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.16.2"
    }
  }
}

provider "aws" {
  default_tags {
    tags = {
      project = "KittenBot"
    }
  }
}

module "infra" {
  source = "./infra"

  dezgo_key = var.dezgo_key
  domain    = var.domain
  image_tag = var.image_tag
  prompts   = var.prompts
}

variable "dezgo_key" {
  type        = string
  description = "Dezgo.com API key"
  validation {
    condition     = length(var.dezgo_key) > 6 && substr(var.dezgo_key, 0, 6) == "DEZGO-"
    error_message = "Must start with 'DEZGO-'"
  }
  sensitive = true
}

variable "domain" {
  type        = string
  description = "Domain name"
}

variable "image_tag" {
  type        = string
  description = "Image tag for lambda"
  default     = "latest"
}

variable "prompts" {
  type = list(object({
    model  = string
    prompt = string
  }))
  description = "Models and prompts"
  validation {
    condition     = length(var.prompts) > 1
    error_message = "List of models/prompts must not be empty"
  }
}