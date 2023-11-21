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

  dezgo_key            = var.dezgo_key
  domain               = var.domain
  image_tag            = var.image_tag
  prompts              = var.prompts
  reddit_client_id     = var.reddit_client_id
  reddit_client_secret = var.reddit_client_secret
  reddit_username      = var.reddit_username
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

variable "reddit_client_id" {
  type        = string
  description = "Reddit API client ID"
  sensitive   = true
}

variable "reddit_client_secret" {
  type        = string
  description = "Reddit API client secret"
  sensitive   = true
}

variable "reddit_username" {
  type        = string
  description = "Reddit API username"
  sensitive   = true
}
