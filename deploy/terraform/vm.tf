data "oci_identity_availability_domains" "all" {
  compartment_id = var.tenancy_ocid
}

# The newest Canonical Ubuntu 24.04 image for Ampere A1.
data "oci_core_images" "ubuntu" {
  compartment_id           = var.compartment_ocid
  operating_system         = "Canonical Ubuntu"
  operating_system_version = "24.04"
  shape                    = "VM.Standard.A1.Flex"
  sort_by                  = "TIMECREATED"
  sort_order               = "DESC"
}

resource "oci_core_instance" "host" {
  compartment_id      = var.compartment_ocid
  availability_domain = data.oci_identity_availability_domains.all.availability_domains[0].name
  display_name        = "lighthouse"
  shape               = "VM.Standard.A1.Flex"

  shape_config {
    ocpus         = var.ocpus
    memory_in_gbs = var.memory_gb
  }

  source_details {
    source_type             = "image"
    source_id               = data.oci_core_images.ubuntu.images[0].id
    boot_volume_size_in_gbs = var.boot_volume_gb
  }

  create_vnic_details {
    subnet_id        = oci_core_subnet.public.id
    assign_public_ip = true
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
    user_data = base64encode(templatefile("${path.module}/cloud-init.yaml", {
      k3s_version        = var.k3s_version
      admin_tunnel_token = data.cloudflare_zero_trust_tunnel_cloudflared_token.admin.token
    }))
  }

  # A new image or cloud-init shouldn't silently replace a running host; do that on purpose.
  lifecycle {
    ignore_changes = [source_details[0].source_id, metadata["user_data"]]
  }
}
