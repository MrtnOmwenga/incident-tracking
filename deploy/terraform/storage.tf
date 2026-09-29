# The nightly database backups: a private bucket whose objects are deleted after 30 days. (The
# Terraform state bucket is created by hand, before the first `terraform init`.)

data "oci_objectstorage_namespace" "ns" {
  compartment_id = var.tenancy_ocid
}

resource "oci_objectstorage_bucket" "backups" {
  compartment_id = var.compartment_ocid
  namespace      = data.oci_objectstorage_namespace.ns.namespace
  name           = "lighthouse-backups"
  access_type    = "NoPublicAccess"
  versioning     = "Disabled"
}

# Object Storage deletes expired objects as a service principal, which needs permission to, even
# in your own tenancy. Scoped to the backups bucket.
resource "oci_identity_policy" "backup_lifecycle" {
  compartment_id = var.tenancy_ocid
  name           = "lighthouse-backup-lifecycle"
  description    = "Lets Object Storage delete Lighthouse backups older than 30 days"
  statements = [
    "Allow service objectstorage-${var.region} to manage object-family in tenancy where target.bucket.name = '${oci_objectstorage_bucket.backups.name}'",
  ]
}

resource "oci_objectstorage_object_lifecycle_policy" "backups" {
  depends_on = [oci_identity_policy.backup_lifecycle]
  namespace  = data.oci_objectstorage_namespace.ns.namespace
  bucket     = oci_objectstorage_bucket.backups.name
  rules {
    name        = "delete-after-30-days"
    action      = "DELETE"
    is_enabled  = true
    time_amount = 30
    time_unit   = "DAYS"
    target      = "objects"
  }
}
