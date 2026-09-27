#!/usr/bin/env bash
# Creates the cluster's secrets. Passwords are generated here and never written to disk; secrets
# that already exist are left alone, so this is safe to run again.
#
# Needs: kubectl pointed at the cluster, and in the environment:
#   TUNNEL_TOKEN          terraform -chdir=deploy/terraform output -raw tunnel_token
#   GITHUB_CLIENT_SECRET  the owner sign-in's OAuth app
#   BACKUP_ENDPOINT BACKUP_BUCKET BACKUP_ACCESS_KEY_ID BACKUP_SECRET_ACCESS_KEY
#                         Object Storage for the nightly database backup
#   SMTP_USERNAME SMTP_PASSWORD   (optional) for email alerts
set -euo pipefail

need() { [[ -n "${!1:-}" ]] || { echo "set $1 (see the comments at the top of $0)" >&2; exit 1; }; }
for v in TUNNEL_TOKEN GITHUB_CLIENT_SECRET BACKUP_ENDPOINT BACKUP_BUCKET BACKUP_ACCESS_KEY_ID BACKUP_SECRET_ACCESS_KEY; do need "$v"; done

random() { openssl rand -hex 24; }

create() { # namespace name key=value...
  local ns=$1 name=$2; shift 2
  if kubectl -n "$ns" get secret "$name" >/dev/null 2>&1; then
    echo "exists: $ns/$name (left as it is)"
    return
  fi
  local args=()
  for kv in "$@"; do args+=(--from-literal="$kv"); done
  kubectl -n "$ns" create secret generic "$name" "${args[@]}" >/dev/null
  echo "created: $ns/$name"
}

for ns in lighthouse demos cloudflared; do
  kubectl get namespace "$ns" >/dev/null 2>&1 || kubectl create namespace "$ns" >/dev/null
done

create lighthouse lighthouse-db owner-password="$(random)" app-password="$(random)"
create lighthouse lighthouse-app github-client-secret="$GITHUB_CLIENT_SECRET" \
  smtp-username="${SMTP_USERNAME:-}" smtp-password="${SMTP_PASSWORD:-}"
create lighthouse lighthouse-backup endpoint="$BACKUP_ENDPOINT" bucket="$BACKUP_BUCKET" \
  access-key-id="$BACKUP_ACCESS_KEY_ID" secret-access-key="$BACKUP_SECRET_ACCESS_KEY"
create demos redacted-db owner-password="$(random)" app-password="$(random)"
create demos redacted-app jwt-secret="$(openssl rand -hex 32)"
create demos ghostchat-app jwt-secret="$(openssl rand -hex 32)"
create cloudflared cloudflared token="$TUNNEL_TOKEN"
