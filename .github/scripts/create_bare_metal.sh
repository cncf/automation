#!/bin/bash

set -euo pipefail

# ----- CONFIGURATION -----
COMPARTMENT_OCID="ocid1.compartment.oc1..aaaaaaaa22icap66vxktktubjlhf6oxvfhev6n7udgje2chahyrtq65ga63a"
SUBNET_OCID="ocid1.subnet.oc1.us-sanjose-1.aaaaaaaahgdslvujnywu3hvhqbvgz23souseseozvypng7ehnxgcotislubq"
AVAILABILITY_DOMAIN="bzBe:US-SANJOSE-1-AD-1"
IMAGE_OCID="ocid1.image.oc1.us-sanjose-1.aaaaaaaa43mwu75532lsj655xqgl4flkmlzbpin54ccoddrkpoyygzh4pvmq" # Canonical-Ubuntu-24.04-aarch64-2025.05.20-0
SHAPE="BM.Standard.A1.160"
SSH_PUBLIC_KEY_PATH="./id_rsa.pub"
SSH_PRIVATE_KEY_PATH="./id_rsa"
INSTANCE_NAME="gha-arm-image-builder-$(date +%s)"
# --------------------------

ssh-keygen -t rsa -f id_rsa -q -N ""

echo "Creating Bare Metal instance: $INSTANCE_NAME"

INSTANCE_OCID=$(/home/runner/bin/oci compute instance launch \
  --compartment-id "$COMPARTMENT_OCID" \
  --availability-domain "$AVAILABILITY_DOMAIN" \
  --shape "$SHAPE" \
  --subnet-id "$SUBNET_OCID" \
  --image-id "$IMAGE_OCID" \
  --display-name "$INSTANCE_NAME" \
  --ssh-authorized-keys-file "$SSH_PUBLIC_KEY_PATH" \
  --query "data.id" --raw-output)

echo "Instance OCID: $INSTANCE_OCID"
echo "export INSTANCE_OCID=$INSTANCE_OCID" >> $GITHUB_ENV

echo "Waiting for instance to become RUNNING..."
/home/runner/bin/oci compute instance wait-for-state "$INSTANCE_OCID" --state RUNNING

echo "Fetching public IP..."
PUBLIC_IP=""
while [ -z "$PUBLIC_IP" ]; do
  PUBLIC_IP=$(/home/runner/bin/oci compute instance list-vnics --instance-id "$INSTANCE_OCID" \
    --query "data[0].\"public-ip\"" --raw-output)
  [ -z "$PUBLIC_IP" ] && echo "Waiting for public IP..." && sleep 10
done
echo "Instance Public IP: $PUBLIC_IP"
echo "export PUBLIC_IP=$PUBLIC_IP" >> $GITHUB_ENV

echo "Waiting for SSH to become available..."
until ssh -o StrictHostKeyChecking=no -i "$SSH_PRIVATE_KEY_PATH" ubuntu@"$PUBLIC_IP" "echo SSH is ready"; do
  sleep 5
done

echo "Your instance is ready."
