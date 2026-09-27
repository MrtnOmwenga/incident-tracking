terraform {
  required_version = ">= 1.10"
  required_providers {
    oci        = { source = "oracle/oci", version = "~> 9.3" }
    cloudflare = { source = "cloudflare/cloudflare", version = "~> 5.26" }
    random     = { source = "hashicorp/random", version = "~> 3.9" }
  }

  # State lives in OCI Object Storage through its S3-compatible API (free tier: 20 GB). The bucket
  # and its credentials are passed at init time, so nothing account-specific is committed:
  #   terraform init -backend-config=backend.hcl
  # (see backend.hcl.example).
  backend "s3" {}
}

provider "oci" {
  tenancy_ocid     = var.tenancy_ocid
  user_ocid        = var.user_ocid
  fingerprint      = var.fingerprint
  private_key_path = var.private_key_path
  region           = var.region
}

# Reads CLOUDFLARE_API_TOKEN from the environment: a token scoped to this zone's DNS and the
# account's tunnels, nothing else.
provider "cloudflare" {}
