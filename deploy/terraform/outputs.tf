output "ssh_hostname" {
  description = "SSH goes through Cloudflare Access; see the ~/.ssh/config entry in the deploy README."
  value       = local.ssh_hostname
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
