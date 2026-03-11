#!/bin/bash

PACKER_VERSION=1.14.3

# Read credentials from env file transferred via SCP
if [ -f ~/oci_credentials.env ]; then
  source ~/oci_credentials.env
  shred -u ~/oci_credentials.env
fi

OCI_CONFIG_FILE="/home/ubuntu/.oci/config"
OCI_KEY_FILE="/home/ubuntu/.oci/oci_api_key.pem"
mkdir -p /home/ubuntu/.oci

cat > ${OCI_CONFIG_FILE} << EOF
[DEFAULT]
user=${OCI_CLI_USER}
fingerprint=${OCI_CLI_FINGERPRINT}
tenancy=${OCI_CLI_TENANCY}
region=${OCI_CLI_REGION}
key_file=${OCI_KEY_FILE}
EOF

echo "${OCI_CLI_KEY_CONTENT}" | base64 -d > ${OCI_KEY_FILE}

chmod 600 ${OCI_CONFIG_FILE}
chmod 600 ${OCI_KEY_FILE}

echo "Waiting for apt lock..."
while sudo fuser /var/lib/apt/lists/lock >/dev/null 2>&1; do
  sleep 3
done

while sudo fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do
  sleep 3
done

sudo apt-get update
sudo apt-get install -y xorriso qemu-system-arm qemu-efi-aarch64 git golang zip pipx

echo 'KERNEL=="kvm", GROUP="kvm", MODE="0666", OPTIONS+="static_node=kvm"' | sudo tee /etc/udev/rules.d/99-kvm4all.rules
sudo udevadm control --reload-rules
sudo udevadm trigger --name-match=kvm
sudo kvm-ok

curl -LO https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_arm64.zip
unzip packer_${PACKER_VERSION}_linux_arm64.zip
sudo mv packer /usr/local/bin/
rm packer_${PACKER_VERSION}_linux_arm64.zip
packer plugins install github.com/hashicorp/oracle
packer plugins install github.com/hashicorp/qemu

pipx install oci-cli
export PATH="$PATH:$HOME/.local/bin"

oci compute image list \
  --compartment-id ocid1.compartment.oc1..aaaaaaaa22icap66vxktktubjlhf6oxvfhev6n7udgje2chahyrtq65ga63a \
  --operating-system runner-images \
  --operating-system-version 123456

git clone https://github.com/cncf/automation
cd automation/ci/gha-runner-vm

PACKER_LOG=1 GITHUB_PERIODIC=true go run main.go \
  --isoURL https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-arm64.img \
  --arch arm64
