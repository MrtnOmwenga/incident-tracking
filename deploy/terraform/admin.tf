# SSH, through its own tunnel. cloudflared runs on the host itself (installed by cloud-init), not in
# the cluster: SSH is how the cluster gets set up, so it can't depend on the cluster. The VM has no
# inbound port at all. Cloudflare Access sits in front: only admin_email can reach the SSH port,
# after a one-time code sent to that address, and the SSH key is still required after that.
#
#   ssh lighthouse   # with the ~/.ssh/config entry from the deploy README

resource "random_bytes" "admin_tunnel_secret" {
  length = 32
}

resource "cloudflare_zero_trust_tunnel_cloudflared" "admin" {
  account_id    = var.cloudflare_account_id
  name          = "lighthouse-admin"
  config_src    = "cloudflare"
  tunnel_secret = random_bytes.admin_tunnel_secret.base64
}

locals {
  ssh_hostname = "ssh.${var.domain}"
}

resource "cloudflare_zero_trust_tunnel_cloudflared_config" "admin" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.admin.id
  config = {
    ingress = [
      { hostname = local.ssh_hostname, service = "ssh://localhost:22" },
      { service = "http_status:404" },
    ]
  }
}

resource "cloudflare_dns_record" "ssh" {
  zone_id = var.cloudflare_zone_id
  name    = local.ssh_hostname
  type    = "CNAME"
  content = "${cloudflare_zero_trust_tunnel_cloudflared.admin.id}.cfargotunnel.com"
  proxied = true
  ttl     = 1
}

data "cloudflare_zero_trust_tunnel_cloudflared_token" "admin" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.admin.id
}

resource "cloudflare_zero_trust_access_policy" "admin" {
  account_id       = var.cloudflare_account_id
  name             = "lighthouse admin"
  decision         = "allow"
  include          = [{ email = { email = var.admin_email } }]
  session_duration = "24h"
}

resource "cloudflare_zero_trust_access_application" "ssh" {
  account_id       = var.cloudflare_account_id
  name             = "lighthouse ssh"
  type             = "self_hosted"
  domain           = local.ssh_hostname
  session_duration = "24h"
  policies         = [{ id = cloudflare_zero_trust_access_policy.admin.id, precedence = 1 }]
}
