

## Oracle cloud set up

```
COMPARTMENT_ID=...

# List availability-domains:
oci iam availability-domain list
AVAILABILITY_DOMAIN=mfcO:US-ASHBURN-AD-1

# Pick the correct subnet for the AD
oci network vcn list --compartment-id ${COMPARTMENT_ID}
oci network vcn create --compartment-id ${COMPARTMENT_ID} --cidr-blocks '["10.0.0.0/16"]'

oci network subnet list --compartment-id ${COMPARTMENT_ID}
oci network subnet create --compartment-id ${COMPARTMENT_ID} --cidr-block 10.0.0.0/16 --vcn-id ${VCN_ID}
oci network subnet get --subnet-id ${SUBNET_ID}

oci network route-table get --rt-id ${ROUTE_TABLE_ID}

oci network internet-gateway create --vcn-id ${VCN_ID} --compartment-id ${COMPARTMENT_ID} --is-enabled true

ROUTE_RULES=$(cat <<EOF
[
  {
    "cidrBlock": "0.0.0.0/0",
    "networkEntityId": "$INTERNET_GATEWAY_ID"
  }
]
EOF
)
oci network route-table update --rt-id ${ROUTE_TABLE_ID}  --route-rules "${ROUTE_RULES}" --force

# Allow SSH
SECURITY_LIST_ID=$(oci network subnet get  --subnet-id ${SUBNET_ID} | jq -r '.data."security-list-ids"[0]')
oci network security-list update \
    --security-list-id ${SECURITY_LIST_ID} \
    --ingress-security-rules '[{"source": "0.0.0.0/0", "protocol": "6", "tcpOptions": {"destinationPortRange": {"min": 22, "max": 22}}}]'
```


## Create OCI disk image

```
IMAGE_TAG=$(date +%Y%m%d-%H%M%S)

BASE_IMAGE_AMD64=$(oci compute image list --compartment-id ${COMPARTMENT_ID} --display-name Canonical-Ubuntu-24.04-Minimal-2025.01.31-1 --query "data[0].id" --raw-output)
echo "Using BASE_IMAGE_AMD64 ${BASE_IMAGE_AMD64}"
go run ./tools/gha-imagebuilder-oci/  --base-image ${BASE_IMAGE_AMD64} --create-image-name gha-${IMAGE_TAG}-amd64 --availability-domain ${AVAILABILITY_DOMAIN} --subnet ${SUBNET_ID} --shape VM.Standard.E5.Flex
AMD64_DISK_ID=$(oci compute image list --display-name gha-${IMAGE_TAG}-amd64 --compartment-id ${COMPARTMENT_ID} --query "data[0].id" --raw-output)
echo "AMD64_DISK_ID is ${AMD64_DISK_ID}"

BASE_IMAGE_ARM64=$(oci compute image list --compartment-id ${COMPARTMENT_ID} --display-name Canonical-Ubuntu-24.04-Minimal-aarch64-2025.01.31-1 --query "data[0].id" --raw-output)
echo "Using BASE_IMAGE_ARM64 ${BASE_IMAGE_ARM64}"
go run ./tools/gha-imagebuilder-oci/  --base-image ${BASE_IMAGE_ARM64} --create-image-name gha-${IMAGE_TAG}-arm64 --availability-domain ${AVAILABILITY_DOMAIN} --subnet ${SUBNET_ID} --shape VM.Standard.A1.Flex
ARM64_DISK_ID=$(oci compute image list --display-name gha-${IMAGE_TAG}-amd64 --compartment-id ${COMPARTMENT_ID} --query "data[0].id" --raw-output)
echo "ARM64_DISK_ID is ${ARM64_DISK_ID}"
```

## Configuring github-actions-runner

# Upload runner image to gcr
# TODO: docker buildx build --push -t gcr.io/${PROJECT_ID}/gha-cloudrunners-gcp:latest .

# TODO: Instructions for hooking up to github-actions-runner
