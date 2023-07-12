locals {
  tags = {
    Name = "${var.project_name}-vpn"
  }
  template_path      = "/var/tmp/vpn"
  admin_user         = "openvpn"
  install_script     = format("%s/%s", local.template_path, "install.sh")
  update_user_script = format("%s/%s", local.template_path, "update_user.sh")
}

resource "aws_eip" "vpn_ip" {
  vpc  = true
  tags = local.tags
}

resource "aws_eip_association" "vpn_eip_assoc" {
  instance_id   = aws_instance.vpn.id
  allocation_id = aws_eip.vpn_ip.id
}

resource "aws_security_group" "vpn" {
  name        = "openvpn"
  description = "Allow inbound UDP access to OpenVPN and unrestricted egress"

  vpc_id = var.vpc_id
  tags   = local.tags

  ingress {
    from_port   = 1194
    to_port     = 1194
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = var.ssh_port
    to_port     = var.ssh_port
    protocol    = "tcp"
    cidr_blocks = [var.ssh_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "vpn" {
  ami                    = var.ami
  instance_type          = var.instance_type
  subnet_id              = var.public_subnet_id
  key_name               = var.key_name
  vpc_security_group_ids = concat([aws_security_group.vpn.id], var.vpc_security_group_ids)
  # We want to allow the destination address to not match the instance since
  # this is a VPN so we disable the `source_dest_check` attribute.
  source_dest_check = false
  tags              = local.tags

  root_block_device {
    encrypted = true
    tags      = local.tags
    # TODO figure this out
    #kms_key_id = var.kms_key_id != null ? var.kms_key_id : "aws/ebs"
  }
}

resource "null_resource" "provision_openvpn" {
  triggers = {
    user        = var.ssh_user
    port        = format("%d", var.ssh_port)
    private_key = var.private_key_openssh
    host        = aws_eip.vpn_ip.public_ip
  }

  depends_on = [aws_instance.vpn, aws_eip.vpn_ip]

  connection {
    type        = "ssh"
    user        = self.triggers.user
    port        = self.triggers.port
    private_key = self.triggers.private_key
    host        = self.triggers.host
  }

  provisioner "remote-exec" {
    inline = [
      "sudo apt-get -y update",
      "sudo apt-get install -y curl vim git libltdl7 python3 python3-pip python software-properties-common unattended-upgrades",
      "sudo hostnamectl set-hostname ${aws_instance.vpn.tags["Name"]}",
      "rm -rf ${local.template_path}",
      "mkdir -p ${local.template_path}",
    ]
  }
}

resource "null_resource" "openvpn_install" {
  triggers = {
    user        = var.ssh_user
    port        = format("%d", var.ssh_port)
    private_key = var.private_key_openssh
    host        = aws_eip.vpn_ip.public_ip
  }

  depends_on = [null_resource.provision_openvpn]

  connection {
    type        = "ssh"
    user        = self.triggers.user
    port        = self.triggers.port
    private_key = self.triggers.private_key
    host        = self.triggers.host
  }

  provisioner "file" {
    destination = local.install_script
    content = templatefile(
      format("%s/%s", path.module, "templates/install.sh.tpl"),
      {
        public_ip = aws_eip.vpn_ip.public_ip
        client    = local.admin_user
      }
    )
  }

  provisioner "remote-exec" {
    inline = [
      format("%s %s", "sudo chmod a+x", local.install_script),
      format("%s %s", "sudo ", local.install_script),
    ]
  }
}

resource "null_resource" "openvpn_adduser" {
  triggers = {
    ssh_user    = var.ssh_user
    port        = format("%d", var.ssh_port)
    private_key = var.private_key_openssh
    host        = aws_eip.vpn_ip.public_ip
    # users       = [
    #   local.admin_user,
    # ]
    user = local.admin_user
  }

  depends_on = [null_resource.openvpn_install]

  connection {
    type        = "ssh"
    user        = self.triggers.ssh_user
    port        = self.triggers.port
    private_key = self.triggers.private_key
    host        = self.triggers.host
  }

  provisioner "file" {
    destination = local.update_user_script
    content = templatefile(
      # TODO add support for multiple users
      format("%s/%s", path.module, "templates/update_user.sh.tpl"),
      {
        client = local.admin_user
      }
    )
  }

  provisioner "remote-exec" {
    inline = [
      format("%s %s", "sudo chmod a+x", local.update_user_script),
      # TODO add support for multiple users
      format("%s %s", "sudo ", local.update_user_script),
      format(
        "sudo cp %s /home/${var.ssh_user}/",
        #join(" ", [for u in self.triggers.users : "/root/${u}.ovpn" ])
        "/root/${self.triggers.user}.ovpn"
      )
    ]
  }
}

data "aws_region" "current" {}

resource "local_file" "private_key_file" {
  filename             = "/tmp/vpn-ssh-key"
  content              = var.private_key_openssh
  file_permission      = "0600"
  directory_permission = "0777"
}

locals {
  ovpn_config_path = "${var.storage_path}/${data.aws_region.current.name}/${aws_eip.vpn_ip.public_ip}"
}

resource "null_resource" "openvpn_download_configurations" {
  triggers = {
    user        = var.ssh_user
    port        = format("%d", var.ssh_port)
    private_key = local_file.private_key_file.filename
    host        = aws_eip.vpn_ip.public_ip
  }

  depends_on = [
    null_resource.openvpn_adduser,
    aws_eip.vpn_ip,
    local_file.private_key_file,
  ]

  provisioner "local-exec" {
    command = <<EOT
    mkdir -p ${local.ovpn_config_path}/;
    scp -o StrictHostKeyChecking=no \
      -o UserKnownHostsFile=/dev/null \
      -i ${self.triggers.private_key} ${self.triggers.user}@${self.triggers.host}:/home/${self.triggers.user}/*.ovpn ${local.ovpn_config_path}/;
EOT
  }
}
