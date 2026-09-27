# The Cloudflare Tunnel: cloudflared (running in the cluster) dials out to Cloudflare, which sends
# each public hostname to a service inside the cluster. No inbound port is ever opened.

resource "random_bytes" "tunnel_secret" {
  length = 32
}

resource "cloudflare_zero_trust_tunnel_cloudflared" "main" {
  account_id    = var.cloudflare_account_id
  name          = "lighthouse"
  config_src    = "cloudflare"
  tunnel_secret = random_bytes.tunnel_secret.base64
}

locals {
  fqdn = { for key, site in var.sites : key => site.hostname == "" ? var.domain : "${site.hostname}.${var.domain}" }
}

resource "cloudflare_zero_trust_tunnel_cloudflared_config" "main" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.main.id
  config = {
    ingress = concat(
      [for key, site in var.sites : { hostname = local.fqdn[key], service = site.service }],
      [{ service = "http_status:404" }], # anything else
    )
  }
}

resource "cloudflare_dns_record" "site" {
  for_each = var.sites
  zone_id  = var.cloudflare_zone_id
  name     = local.fqdn[each.key]
  type     = "CNAME"
  content  = "${cloudflare_zero_trust_tunnel_cloudflared.main.id}.cfargotunnel.com"
  proxied  = true
  ttl      = 1 # automatic
}

data "cloudflare_zero_trust_tunnel_cloudflared_token" "main" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.main.id
}
