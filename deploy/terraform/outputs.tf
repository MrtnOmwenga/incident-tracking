output "host_public_ip" {
  description = "For SSH (from admin_cidr only): ssh ubuntu@<ip>"
  value       = oci_core_instance.host.public_ip
}

output "tunnel_token" {
  description = "cloudflared's credential; goes into the cluster as a secret (deploy/k8s README)."
  value       = data.cloudflare_zero_trust_tunnel_cloudflared_token.main.token
  sensitive   = true
}

output "urls" {
  value = { for key, name in local.fqdn : key => "https://${name}" }
}

output "backup_endpoint" {
  description = "BACKUP_ENDPOINT for create-secrets.sh (the S3-compatible API)."
  value       = "https://${data.oci_objectstorage_namespace.ns.namespace}.compat.objectstorage.${var.region}.oraclecloud.com"
}
