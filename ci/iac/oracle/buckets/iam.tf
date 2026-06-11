resource "oci_identity_user" "service_user" {
  compartment_id = var.tenancy_ocid
  name           = var.service_user_name
  description    = var.service_user_description
  email          = var.service_user_email
}

resource "oci_identity_group" "service_group" {
  compartment_id = var.tenancy_ocid
  name           = "${var.service_user_name}-group"
  description    = "Group for ${var.service_user_name}"
}

resource "oci_identity_user_group_membership" "service_membership" {
  user_id  = oci_identity_user.service_user.id
  group_id = oci_identity_group.service_group.id
}

resource "oci_identity_policy" "bucket_write_policy" {
  compartment_id = var.compartment_ocid
  name           = "${var.service_user_name}-bucket-write"
  description    = "Allow ${var.service_user_name} to manage objects in designated buckets"

  statements = [
    for bucket in keys(var.buckets) :
    "Allow group ${oci_identity_group.service_group.name} to manage objects in compartment id ${var.compartment_ocid} where all {target.bucket.name='${bucket}'}"
  ]
}

resource "oci_identity_customer_secret_key" "service_user_s3_key" {
  display_name = "${var.service_user_name}-s3-compat-key"
  user_id      = oci_identity_user.service_user.id
}
