output "buckets" {
  value = {
    for name, bucket in oci_objectstorage_bucket.buckets : name => {
      id        = bucket.id
      namespace = bucket.namespace
      name      = bucket.name
    }
  }
  description = "Map of created bucket names to their details."
}

output "replica_buckets" {
  value = {
    for name, bucket in oci_objectstorage_bucket.replica_buckets : name => {
      id        = bucket.id
      namespace = bucket.namespace
      name      = bucket.name
    }
  }
  description = "Map of replica bucket names to their details."
}

output "service_user" {
  value = {
    id   = oci_identity_user.service_user.id
    name = oci_identity_user.service_user.name
  }
  description = "Service user details."
}

output "service_group" {
  value = {
    id   = oci_identity_group.service_group.id
    name = oci_identity_group.service_group.name
  }
  description = "Service group details."
}

output "s3_compatible_access_key" {
  value       = oci_identity_customer_secret_key.service_user_s3_key.id
  description = "S3-compatible access key ID for the service user."
}

output "s3_compatible_secret_key" {
  value       = oci_identity_customer_secret_key.service_user_s3_key.key
  sensitive   = true
  description = "S3-compatible secret key for the service user."
}
