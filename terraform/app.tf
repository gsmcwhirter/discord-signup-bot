resource "digitalocean_droplet" "app" {
    count = 0
    image = var.latest_image
    name = "app-${count.index}"
    region = "sfo2"
    size = "s-1vcpu-1gb"
    private_networking = true
    user_data = var.app_user_data

    ssh_keys = [
      data.digitalocean_ssh_key.terraform.id
    ]
}