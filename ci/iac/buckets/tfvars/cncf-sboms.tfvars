region    = "us-sanjose-1"
namespace = "axtwf1hkrwcy"

service_user_name        = "cncf-sboms-writer"
service_user_description = "Service user for writing SBOM data to Object Storage buckets"
service_user_email       = "projects@cncf.io"

buckets = {
  "cncf-project-sboms" = {
    access_type  = "ObjectRead"
    storage_tier = "Standard"
    versioning   = "Disabled"w
  }
  "cncf-subproject-sboms" = {
    access_type  = "ObjectRead"
    storage_tier = "Standard"
    versioning   = "Disabled"
  }
}
