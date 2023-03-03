terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
  }
}

locals {
  name              = "personal-vpn"
  ovpn_storage_path = "./openvpn"
  region_name       = "oregon"
  region_mapping = tomap({
    oregon     = "us-west-2" # Pretty cheap
    california = "us-west-1" # more expensive
    london     = "eu-west-2"
  })
}

provider "aws" {
  profile = "default"
  region  = lookup(local.region_mapping, local.region_name)
}

module "ubuntu" {
  source = "../../modules/aws/ubuntu_ami"
}

module "key" {
  source   = "../../modules/ssh-key"
  keys_dir = "./keys"
  name     = "vpn"
}

resource "aws_key_pair" "key" {
  key_name   = "vpn-key"
  public_key = trimspace(module.key.public_key)
}

module "vpn" {
  source              = "../../modules/aws/vpn"
  project_name        = local.name
  storage_path        = local.ovpn_storage_path
  vpc_id              = aws_vpc.vpc.id
  public_subnet_id    = aws_subnet.public.id
  ami                 = module.ubuntu.ami
  instance_type       = "t3a.nano"
  key_name            = aws_key_pair.key.key_name
  ssh_user            = "ubuntu"
  public_key_openssh  = module.key.public_key
  private_key_openssh = module.key.private_key
}

output "ip" {
  value = module.vpn.public_ip
}

output "config" {
  value = module.vpn.config_file
}
