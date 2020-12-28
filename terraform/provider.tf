terraform {
  required_providers {
    digitalocean = {
      source = "digitalocean/digitalocean"
    }
  }
  required_version = ">= 0.13"

  backend "s3" {
    skip_credentials_validation = true
    skip_metadata_api_check = true
    endpoint = "https://sfo2.digitaloceanspaces.com"
    region = "us-east-1"
    bucket = "discord-bot-terraform"
    key = "production/terraform.tfstate"
  }
}

provider "digitalocean" {
    token = var.do_token
}