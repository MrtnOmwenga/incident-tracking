# Deploying Lighthouse and the demos

Everything runs on one Oracle Cloud Always Free ARM VM (4 OCPUs, 24 GB) with k3s. Visitors arrive
through a Cloudflare Tunnel, so the VM has no open web ports. Terraform builds the machine and the
tunnel; Flux keeps the cluster in step with this repository.

```
visitors ──▶ Cloudflare (TLS, DNS) ──tunnel──▶ cloudflared ─┬─▶ lighthouse   (namespace lighthouse, + PostgreSQL)
                                                            ├─▶ redacted     (namespace demos, + PostgreSQL)
                                                            └─▶ ghostchat    (namespace demos, + MongoDB, Redis)
GitHub Actions ──▶ GHCR (multi-arch images, signed) ──▶ Flux (commits new tags, applies deploy/k8s/production)
```

| Folder | What |
|---|---|
| `terraform/` | The OCI network and VM (cloud-init installs k3s from a checksum-verified binary), the Cloudflare tunnel, its routes and DNS records |
| `k8s/base/` | The workloads. Every pod: non-root, read-only filesystem, no capabilities, seccomp; "restricted" Pod Security enforced per namespace; network policies deny ingress by default |
| `k8s/production/` | The settings that differ per deployment (`lighthouse.env`, `ghostchat.env`) and the image tags Flux updates |
| `k8s/flux/` | What Flux applies, and the image automation |
| `k8s/scripts/create-secrets.sh` | Creates the secrets (passwords generated on the spot, never written to disk) |
| `site/` | The portfolio's content, baked into the Lighthouse image |

CI renders the manifests and validates them (kubeconform), checks every container is locked down,
and runs `terraform validate`.

## What you need first

1. **Oracle Cloud:** an account, upgraded to *Pay As You Go* so the Always Free VM isn't reclaimed
   as idle, with a **budget alert at $1** (Billing > Budgets). An API signing key for your user
   (Identity > Users > API keys) and a Customer Secret Key (for Terraform state and backups in
   Object Storage). Create one bucket by hand, `lighthouse-terraform`, for Terraform's state;
   Terraform creates the backups bucket itself.
2. **A domain on Cloudflare** (free plan), its zone ID, your account ID, and an API token with
   *Zone:DNS:Edit* on that zone and *Account:Cloudflare Tunnel:Edit*.
3. **A GitHub OAuth app** for the owner sign-in: callback `https://<domain>/auth/github/callback`.
4. **Optional, for email alerts:** SMTP credentials (for example Brevo's or Resend's free tier).

## Steps

```sh
# 1. The machine and the tunnel.
cd deploy/terraform
cp terraform.tfvars.example terraform.tfvars   # fill in
cp backend.hcl.example backend.hcl             # fill in
export CLOUDFLARE_API_TOKEN=...  AWS_ACCESS_KEY_ID=...  AWS_SECRET_ACCESS_KEY=...   # the last two: the Customer Secret Key
terraform init -backend-config=backend.hcl
terraform apply

# 2. A kubeconfig, over SSH (only your admin_cidr can reach port 22).
ssh ubuntu@$(terraform output -raw host_public_ip) sudo cat /etc/rancher/k3s/k3s.yaml > ~/.kube/lighthouse.yaml
# Point its server at localhost through a tunnel: ssh -L 6443:127.0.0.1:6443 ubuntu@<ip>
export KUBECONFIG=~/.kube/lighthouse.yaml

# 3. Settings: your domain in deploy/k8s/production/lighthouse.env (PUBLIC_URL, GITHUB_CLIENT_ID,
#    alerts) and ghostchat.env (CORS_ORIGINS). Commit them.

# 4. Namespaces, then secrets.
kubectl apply -f deploy/k8s/base/namespaces.yaml
TUNNEL_TOKEN=$(terraform -chdir=deploy/terraform output -raw tunnel_token) \
GITHUB_CLIENT_SECRET=... \
BACKUP_ENDPOINT=$(terraform -chdir=deploy/terraform output -raw backup_endpoint) BACKUP_BUCKET=lighthouse-backups \
BACKUP_ACCESS_KEY_ID=... BACKUP_SECRET_ACCESS_KEY=... \
SMTP_USERNAME=... SMTP_PASSWORD=... \
  deploy/k8s/scripts/create-secrets.sh

# 5. Flux: installs itself, then applies deploy/k8s/production and keeps it applied.
flux bootstrap github --owner=MrtnOmwenga --repository=lighthouse --branch=main \
  --path=deploy/k8s/flux --personal --components-extra=image-reflector-controller,image-automation-controller
```

6. **Images:** the release workflows in Lighthouse, RBAC-API and GhostChat publish to GHCR on every
   push to `main`. Make each package public once (GitHub > Packages > Package settings), or give
   the cluster a pull secret.
7. **Monitors:** sign in to `https://<domain>/console/` with GitHub and add HTTP monitors for each
   demo's in-cluster health address (`http://redacted.demos.svc.cluster.local:3000/health/ready`,
   `http://ghostchat.demos.svc.cluster.local:5000/health`), with *Allow private network* on, and for
   the public addresses too. Give each the slug its project uses in `site/site.yaml`.

## Operating it

- **Deploying:** push to `main`. The image is built, signed and published; Flux notices the new
  tag within ten minutes, commits it to `production/kustomization.yaml` and rolls it out.
- **Watching:** `flux get kustomizations`, `kubectl get pods -A`, and Lighthouse itself.
- **Backups:** Lighthouse's database is dumped nightly to the backups bucket (Terraform's lifecycle
  rule deletes them after 30 days). Restore: `pg_restore -h <postgres> -U lighthouse_owner -d lighthouse
  --clean <file>.dump` from a pod in the `lighthouse` namespace. The demos' data is disposable by
  design and not backed up.
- **Rotating a secret:** delete it (`kubectl -n <ns> delete secret <name>`), run the script again,
  and restart the pods that use it. The database passwords are also stored in the databases, so
  rotate those with `ALTER ROLE` first.
- **Cost:** everything used is in Oracle's and Cloudflare's free tiers. The budget alert is the
  safety net.
