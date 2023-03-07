terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
    # https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 3.31.0"
    }
    # https://registry.terraform.io/providers/grafana/grafana/latest/docs
    grafana = {
      source  = "grafana/grafana"
      version = "~> 1.27.0"
    }
    postgresql = {
      source  = "cyrilgdn/postgresql"
      version = "1.18.0"
    }
  }
}

locals {
  name = "dev-instance"
  env  = "dev"
}

provider "aws" {
  profile = "default"
  region  = var.aws_region
}

provider "grafana" {
  url  = "https://grafana.hrry.local"
  auth = ""
}

resource "aws_vpc" "vpc" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true
  tags = {
    Name        = "${local.name}-vpc"
    Environment = local.env
  }
}

resource "aws_internet_gateway" "ig" {
  vpc_id = aws_vpc.vpc.id
  tags = {
    Name        = "${local.name}-ig"
    Environment = local.env
  }
}

resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.vpc.id
  cidr_block              = "10.0.10.0/24"
  map_public_ip_on_launch = true
  tags = {
    Name        = "${local.name}-public-subnet"
    Environment = local.env
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.vpc.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.ig.id
  }
  route {
    ipv6_cidr_block = "::/0"
    gateway_id      = aws_internet_gateway.ig.id
  }
  tags = {
    Name        = "${local.name}-public-route-table"
    Environment = local.env
  }
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

data "http" "myip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  myip = chomp(data.http.myip.response_body)
}

resource "aws_security_group" "sg" {
  name        = "kubernetes"
  description = "Allow inbound access to a Kuernetes control plane."

  vpc_id = aws_vpc.vpc.id
  tags = {
    Name        = "${local.name}-k8s-security-group"
    Environment = local.env
  }

  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"
    # cidr_blocks = ["${local.myip}/32"]
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    # Kubernetes control plane
    from_port = 6443
    to_port   = 6443
    protocol  = "tcp"
    cidr_blocks = [
      "${local.myip}/32" # only traffic from my public IP
    ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

module "ubuntu" {
  source = "../../modules/aws/ubuntu_ami"
}

module "key" {
  source   = "../../modules/ssh-key"
  keys_dir = "./keys"
  name     = "aws_dev_instance"
}

resource "local_file" "pubkey" {
  filename        = pathexpand("~/.ssh/aws_dev_instance.pub")
  file_permission = "0640"
  content         = module.key.public_key
}

resource "local_file" "prvkey" {
  filename        = pathexpand("~/.ssh/aws_dev_instance")
  file_permission = "0600"
  content         = module.key.private_key
}

resource "aws_key_pair" "key" {
  key_name   = "development-key"
  public_key = trimspace(module.key.public_key)
}

resource "aws_instance" "server" {
  ami                    = module.ubuntu.ami
  instance_type          = "t3a.nano"
  subnet_id              = aws_subnet.public.id
  key_name               = aws_key_pair.key.key_name
  vpc_security_group_ids = [aws_security_group.sg.id]
  tags = {
    Name        = "${local.name}"
    Environment = local.env
  }
}

resource "aws_eip" "ip" {
  vpc = true
  tags = {
    Name        = "${local.name}-eip"
    Environment = local.env
  }
}

resource "aws_eip_association" "vpn_eip_assoc" {
  instance_id   = aws_instance.server.id
  allocation_id = aws_eip.ip.id
}

output "ip" {
  value = aws_eip.ip.public_ip
}
