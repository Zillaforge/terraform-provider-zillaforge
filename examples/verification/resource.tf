resource "zillaforge_security_group" "ssh" {
  name = "ssh-sg"

  ingress_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_cidr = "0.0.0.0/0"
  }

  egress_rule {
    protocol         = "any"
    port_range       = "all"
    destination_cidr = "0.0.0.0/0"
  }
}

resource "zillaforge_keypair" "sandbox" {
  name = "sandbox-keypair"
}


resource "zillaforge_floating_ip" "sandbox" {
  count = 3
  name  = "sandbox-floating-ip-${count.index + 1}"
}

resource "zillaforge_server" "sandbox" {
  count     = 3
  name      = "sandbox-0${count.index + 1}"
  flavor_id = data.zillaforge_flavors.small_falvor.flavors[0].id
  image_id  = data.zillaforge_images.cirros.images[0].id
  keypair   = zillaforge_keypair.sandbox.id

  network_attachment {
    network_id     = data.zillaforge_networks.default.networks[0].id
    floating_ip_id = zillaforge_floating_ip.sandbox[count.index].id
    security_group_ids = [
      zillaforge_security_group.ssh.id
    ]
  }
}