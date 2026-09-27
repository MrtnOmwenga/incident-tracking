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

resource "oci_objectstorage_object_lifecycle_policy" "backups" {
  namespace = data.oci_objectstorage_namespace.ns.namespace
  bucket    = oci_objectstorage_bucket.backups.name
  rules {
    name        = "delete-after-30-days"
    action      = "DELETE"
    is_enabled  = true
    time_amount = 30
    time_unit   = "DAYS"
    target      = "objects"
  }
}
