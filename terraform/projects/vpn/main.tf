terraform {
  backend "s3" {
    bucket         = "hrryhomelab"
    key            = "infra/projects/vpn.tfstate"
    region         = "us-west-2"
    dynamodb_table = "infra"
  }
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.34.0"
    }
  }
}

provider "aws" {
  profile = "terraform"
  region  = lookup(local.region_mapping, local.region_name)
}

provider "cloudflare" {
  api_token = var.cloudflare_token
}

locals {
  name              = "personal-vpn"
  ovpn_storage_path = "./openvpn"
  region_mapping = tomap({
    oregon     = "us-west-2" # Pretty cheap
    california = "us-west-1" # more expensive
    london     = "eu-west-2"
  })
  region_name = "oregon"
  # https://instances.vantage.sh/?region=us-west-2&selected=t4g.micro
  instance_type = "t4g.micro" # 1GiB, 2vcpu burst, 5 Gib net
  # instance_type = "t3a.nano"
  #ami = "ami-008fe2fc65df48dac"
  ami = "ami-0dca369228f3b2ce7" # Ubuntu Server 20.04 LTS (HVM), SSD Volume Type
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
  users               = ["harry-ovpn"]
  storage_path        = local.ovpn_storage_path
  vpc_id              = aws_vpc.vpc.id
  public_subnet_id    = aws_subnet.public.id
  ami                 = local.ami
  instance_type       = local.instance_type
  key_name            = aws_key_pair.key.key_name
  ssh_user            = "ubuntu"
  public_key_openssh  = module.key.public_key
  private_key_openssh = module.key.private_key
  ipv6                = true
}

data "cloudflare_zone" "hrry_dev" {
  name = "hrry.dev"
}

resource "cloudflare_record" "vpn" {
  zone_id = data.cloudflare_zone.hrry_dev.id
  type    = "A"
  name    = "vpn"
  value   = module.vpn.public_ip
  proxied = false
  ttl     = 60
}

output "ip" {
  value = module.vpn.public_ip
}

output "config" {
  value = module.vpn.config_file
}
