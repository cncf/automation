# GitHub Actions Runner VM Image Builder

This tool generates and uploads GitHub Actions runner VM images to Oracle Cloud Infrastructure (OCI).

## Features

- **Build**: Generate and upload a new GHA runner image to OCI
- **List Capabilities**: Display the virtual machine capabilities defined for the GHA runner images

## Usage

### List VM Capabilities

To view the virtual machine capabilities configured for the GitHub Action runners:

```bash
go run main.go list-capabilities
```

Or if you have built the binary:

```bash
./gha-runner-vm list-capabilities
```

This command displays:
- Compute capabilities (firmware, launch mode, secure boot, etc.)
- Network capabilities (attachment type, IPv6 support)
- Storage capabilities (volume types, encryption, naming conventions)
- Configuration details (compartment ID, schema version)

### Build a New Runner Image

To build and upload a new runner image:

```bash
go run main.go build [flags]
```

Available flags:
- `--os`: Operating System (default: "ubuntu")
- `--os-version`: Operating System Version (default: "24.04")
- `--arch`: Architecture - "x86" or "arm64" (default: "x86")
- `--bucketName`: OCI bucket name
- `--compartmentId`: OCI compartment ID
- `--namespace`: OCI namespace
- `--isoURL`: ISO URL for Packer to use
- `--isoChecksum`: ISO Checksum for Packer to use
- `--debug`: Enable debug logging

## VM Capabilities

The VM capabilities are defined in `capability-update.json` and include settings for:

### Compute
- Firmware support (BIOS, UEFI_64)
- Launch mode (PARAVIRTUALIZED)
- AMD Secure Encrypted Virtualization
- Secure Boot

### Network
- Attachment type (PARAVIRTUALIZED)
- IPv6-only support

### Storage
- Boot, local, and remote data volume types (ISCSI, PARAVIRTUALIZED)
- Consistent volume naming
- ISCSI multipath device support
- Paravirtualization encryption in transit
- Paravirtualization attachment version

## Building

```bash
go build -o gha-runner-vm
```

## Requirements

- Go 1.24.2 or later
- Packer (for building images)
- OCI CLI (for uploading to Oracle Cloud)
- Appropriate OCI credentials configured
