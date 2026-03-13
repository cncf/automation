resource "oci_objectstorage_bucket" "replica_buckets" {
  provider = oci.replica
  for_each = var.buckets

  compartment_id = var.compartment_ocid
  namespace      = var.namespace
  name           = each.key
  access_type    = each.value.access_type
  storage_tier   = each.value.storage_tier
  versioning     = each.value.versioning
}

resource "oci_identity_policy" "replication_policy" {
  compartment_id = var.compartment_ocid
  name           = "object-storage-replication-policy"
  description    = "Allow Object Storage service to manage objects for replication"

  statements = [
    "Allow service objectstorage-${var.region} to manage object-family in compartment id ${var.compartment_ocid}",
    "Allow service objectstorage-${var.replica_region} to manage object-family in compartment id ${var.compartment_ocid}",
  ]
}

resource "oci_objectstorage_replication_policy" "replication" {
  for_each = var.buckets

  namespace               = var.namespace
  bucket                  = oci_objectstorage_bucket.buckets[each.key].name
  name                    = "${each.key}-replication-to-ashburn"
  destination_region_name = var.replica_region
  destination_bucket_name = oci_objectstorage_bucket.replica_buckets[each.key].name

  depends_on = [oci_identity_policy.replication_policy]
}
