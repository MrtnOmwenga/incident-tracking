# Deploying Lighthouse and the demos

Everything runs on one Oracle Cloud Always Free ARM VM (4 OCPUs, 24 GB) with k3s. Visitors arrive
through a Cloudflare Tunnel, and SSH through a second one behind Cloudflare Access, so the VM has
no open inbound ports at all. Terraform builds the machine and the
tunnel; Flux keeps the cluster in step with this repository.

```
visitors ──▶ Cloudflare (TLS, DNS) ──tunnel──▶ cloudflared ─┬─▶ lighthouse   (namespace lighthouse, + PostgreSQL)
                                                            ├─▶ redacted     (namespace demos, + PostgreSQL)
                                                            └─▶ ghostchat    (namespace demos, + MongoDB, Redis)
GitHub Actions ──▶ GHCR (multi-arch images, signed) ──▶ Flux (commits new tags, applies deploy/k8s/production)
```

| Folder | What |
|---|---|
| `terraform/` | The OCI network (no inbound rules) and VM (cloud-init installs k3s from a checksum-verified binary, and cloudflared for SSH), the two Cloudflare tunnels, their DNS records, the Access policy in front of SSH, and the backups bucket with its 30-day expiry (and the IAM policy that expiry needs) |
| `k8s/base/` | The workloads. Every pod: non-root, read-only filesystem, no capabilities, seccomp; "restricted" Pod Security enforced per namespace; network policies deny ingress by default |
| `k8s/production/` | The settings that differ per deployment (`lighthouse.env`, `ghostchat.env`) and the image tags Flux updates |
| `k8s/flux/` | What Flux applies, and the image automation |
| `k8s/scripts/create-secrets.sh` | Creates the secrets (passwords generated on the spot, never written to disk) |
| `site/` | The portfolio's content, baked into the Lighthouse image |

CI renders the manifests and validates them (kubeconform), checks every container is locked down,
and runs `terraform validate`.

## Status

| | |
|---|---|
| Domain | `martinomwenga.com`, on Cloudflare |
| Region | Oracle `eu-stockholm-1` |
| Network, tunnels, DNS, Access, backups bucket | Created (29 September 2026) |
| VM | Waiting for Arm capacity ("Out of host capacity"); retried every 10 minutes |
| Cluster, secrets, Flux | Next, once the VM is up: steps 2 to 7 below |
| Manifests | Proven end to end on a local kind cluster: every pod ran under restricted Pod Security, and the network policies blocked what they should |
| Email alerts | Off for now (`ALERT_TO` empty) |

## What you need first

1. **Oracle Cloud:** an account, upgraded to *Pay As You Go* so the Always Free VM isn't reclaimed
   as idle, with a **budget alert at $1** (Billing > Budgets). An API signing key for your user
   (Identity > Users > API keys) and a Customer Secret Key (for Terraform state and backups in
   Object Storage). Create one bucket by hand, `lighthouse-terraform`, for Terraform's state;
   Terraform creates the backups bucket itself.

   Without the Pay As You Go upgrade, everything still works, but Always Free Arm capacity is
   scarce and the VM may fail to create with "Out of host capacity" (see below).
2. **A domain on Cloudflare** (free plan), its zone ID, your account ID, and an API token with
   *Zone:DNS:Edit* on that zone, *Account:Cloudflare Tunnel:Edit* and *Account:Access: Apps and
   Policies:Edit*. Open **Zero Trust** in the dashboard once and pick the Free plan and a team
   name, so Access is enabled on the account.
3. **A GitHub OAuth app** for the owner sign-in: callback `https://<domain>/auth/github/callback`.
4. **Optional, for email alerts:** SMTP credentials (for example Brevo's or Resend's free tier).

## Steps

```sh
# 1. The machine and the tunnel.
cd deploy/terraform
cp terraform.tfvars.example terraform.tfvars   # fill in
cp backend.hcl.example backend.hcl             # fill in
export CLOUDFLARE_API_TOKEN=...  AWS_ACCESS_KEY_ID=...  AWS_SECRET_ACCESS_KEY=...   # the last two: the Customer Secret Key
# OCI's S3 API rejects the chunked uploads newer AWS SDKs send by default (the state lock fails).
export AWS_REQUEST_CHECKSUM_CALCULATION=when_required AWS_RESPONSE_CHECKSUM_VALIDATION=when_required
terraform init -backend-config=backend.hcl
terraform apply

# 2. SSH, through the admin tunnel. Install cloudflared locally, then add to ~/.ssh/config:
#
#      Host lighthouse
#        HostName ssh.<domain>
#        User ubuntu
#        IdentityFile ~/.ssh/lighthouse
#        ProxyCommand cloudflared access ssh --hostname %h
#
#    The first connection opens a browser for Cloudflare Access (a code sent to admin_email).
#    cloud-init takes a few minutes after `apply` before SSH answers.
ssh lighthouse sudo cat /etc/rancher/k3s/k3s.yaml > ~/.kube/lighthouse.yaml && chmod 600 ~/.kube/lighthouse.yaml
# kubectl reaches the API server through SSH: keep this running in another terminal.
ssh -N -L 6443:127.0.0.1:6443 lighthouse
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

## What the first real apply found

`terraform validate` and the local kind cluster can't catch what only a real cloud account
rejects. The first `apply` found three things:

- **The state lock failed:** `NotImplemented: AWS chunked encoding not supported`. Newer AWS SDKs
  (which Terraform's S3 backend uses) stream uploads with chunked checksums; OCI's S3-compatible
  API doesn't accept them. Fix: `AWS_REQUEST_CHECKSUM_CALCULATION=when_required` and
  `AWS_RESPONSE_CHECKSUM_VALIDATION=when_required` (step 1). `skip_s3_checksum` in the backend
  config isn't enough on its own for the lock file.
- **The backups' expiry rule was refused:** `InsufficientServicePermissions`. Object Storage deletes
  expired objects as a service principal, which needs an IAM policy even inside your own tenancy.
  `storage.tf` now creates one, scoped to the backups bucket.
- **The VM: `Out of host capacity`.** Free Arm capacity comes and goes, and free-tier accounts are
  served last. Nothing is wrong with the configuration: everything else is created, and the VM can
  be retried on its own until it succeeds. Each retry plans the VM alone and applies only if that
  plan is exactly one resource to add:

  ```sh
  terraform plan -target=oci_core_instance.host -out=vm.tfplan   # expect: 1 to add, 0 to change, 0 to destroy
  terraform apply vm.tfplan
  ```

  Upgrading the account to Pay As You Go (still free for this usage) makes capacity much easier
  to get.

Also worth knowing: Cloudflare's provider can't delete a tunnel's configuration, so `terraform
destroy` leaves it behind (it warns about this); delete it in the dashboard if you ever tear down.

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
