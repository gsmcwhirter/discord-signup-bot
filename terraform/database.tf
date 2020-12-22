resource "digitalocean_database_cluster" "signup-bot-pg" {
    name = "signup-bot-pg"
    engine = "pg"
    version = 12
    size = "db-s-1vcpu-1gb"
    region = "sfo2"
    node_count = 1

    # maintenance_window {
    #     day = "Sunday"
    #     hour = "08:00"
    # }
}

resource "digitalocean_database_db" "signup-bot-pg" {
    cluster_id = digitalocean_database_cluster.signup-bot-pg.id
    name       = "signup_bot"
}

resource "digitalocean_database_firewall" "signup-bot-pg" {
    cluster_id = digitalocean_database_cluster.signup-bot-pg.id
    rule {
        type = "ip_addr"
        value = "192.168.1.1"  # yes, this is localhost -- will configure manually in UI
    }

    # rule {
    #     type = "droplet"
    #     value = digitalocean_droplet.app.id
    # }
}