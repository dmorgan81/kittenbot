terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.16.2"
    }
  }

  required_version = ">= 1.5.0"

  cloud {
    organization = "kittenbot"
    workspaces {
      name = "main"
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
