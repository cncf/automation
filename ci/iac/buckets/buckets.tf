resource "oci_objectstorage_bucket" "buckets" {
  for_each = var.buckets

  compartment_id = var.compartment_ocid
  namespace      = var.namespace
  name           = each.key
  access_type    = each.value.access_type
  storage_tier   = each.value.storage_tier
  versioning     = each.value.versioning
}
