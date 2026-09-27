# Oracle Cloud: an API signing key for a user in the tenancy (Identity > Users > API keys).
variable "tenancy_ocid" { type = string }
variable "user_ocid" { type = string }
variable "fingerprint" { type = string }
variable "private_key_path" { type = string }
variable "region" {
  type        = string
  description = "e.g. eu-frankfurt-1; Always Free ARM capacity varies by region."
}
variable "compartment_ocid" {
  type        = string
  description = "Where to create everything; the tenancy's root compartment works."
}

# The VM. The Always Free allowance is 4 OCPUs and 24 GB of memory on Ampere A1, and 200 GB of
# block storage in total.
variable "ocpus" {
  type    = number
  default = 4
}
variable "memory_gb" {
  type    = number
  default = 24
}
variable "boot_volume_gb" {
  type    = number
  default = 100
}
variable "ssh_public_key" {
  type        = string
  description = "Your SSH public key, for the ubuntu user."
}
variable "admin_cidr" {
  type        = string
  description = "The only address allowed to SSH in, e.g. 203.0.113.7/32. Nothing else is open inbound."
  validation {
    condition     = can(cidrhost(var.admin_cidr, 0)) && var.admin_cidr != "0.0.0.0/0"
    error_message = "admin_cidr must be a CIDR block, and not the whole internet."
  }
}
variable "k3s_version" {
  type    = string
  default = "v1.37.0+k3s1"
}

# Cloudflare: the zone (domain) the sites live on, and the account that owns the tunnel.
variable "cloudflare_account_id" { type = string }
variable "cloudflare_zone_id" { type = string }
variable "domain" {
  type        = string
  description = "The zone's name, e.g. omwenga.dev."
}
variable "sites" {
  type = map(object({
    hostname = string # "" for the apex
    service  = string # where cloudflared sends the traffic, inside the cluster
  }))
  description = "Public hostnames and the in-cluster services behind them."
  default = {
    lighthouse = { hostname = "", service = "http://lighthouse.lighthouse.svc.cluster.local:8080" }
    redacted   = { hostname = "redacted", service = "http://redacted.demos.svc.cluster.local:3000" }
    ghostchat  = { hostname = "ghostchat", service = "http://ghostchat.demos.svc.cluster.local:5000" }
  }
}
