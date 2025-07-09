#!/bin/bash

# oci raw-request command requires key_file to be set
# so, manually creating the OCI config files
OCI_CONFIG_FILE="/home/ubuntu/.oci/config"
OCI_KEY_FILE="/home/ubuntu/.oci/oci_api_key.pem"
mkdir /home/ubuntu/.oci

cat > ${OCI_CONFIG_FILE} << EOF
[DEFAULT]
user=$1
fingerprint=$3
tenancy=$2
region=$5
key_file=${OCI_KEY_FILE}
EOF

echo $4 | base64 -d > ${OCI_KEY_FILE}

chmod 600 ${OCI_CONFIG_FILE}
chmod 600 ${OCI_KEY_FILE}

sudo apt update
sudo apt install -y xorriso qemu-system-arm qemu-efi-aarch64 git golang zip

echo 'KERNEL=="kvm", GROUP="kvm", MODE="0666", OPTIONS+="static_node=kvm"' | sudo tee /etc/udev/rules.d/99-kvm4all.rules
sudo udevadm control --reload-rules
sudo udevadm trigger --name-match=kvm
sudo kvm-ok

curl -LO https://releases.hashicorp.com/packer/1.12.0/packer_1.12.0_linux_arm64.zip
unzip packer_1.12.0_linux_arm64.zip
sudo mv packer /usr/local/bin/
rm packer_1.12.0_linux_arm64.zip
packer plugin install github.com/hashicorp/oracle
packer plugins install github.com/hashicorp/qemu
packer plugins install github.com/hashicorp/azure

curl -L -O https://raw.githubusercontent.com/oracle/oci-cli/master/scripts/install/install.sh
chmod +x install.sh
./install.sh --accept-all-defaults
export PATH=$PATH:$HOME/bin

oci compute image list \
  --compartment-id ocid1.compartment.oc1..aaaaaaaa22icap66vxktktubjlhf6oxvfhev6n7udgje2chahyrtq65ga63a \
  --operating-system runner-images \
  --operating-system-version 123456

git clone https://github.com/cncf/automation
cd automation/ci/gha-runner-vm

PACKER_LOG=1 go run main.go \
  --isoURL https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-arm64.img \
  --arch arm64
