package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v71/github"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/spf13/cobra"
)

var replacements = make(map[string]string)
var selectedRelease *github.RepositoryRelease
var Cmd = &cobra.Command{
	Use:  "gha-runner-vm",
	Long: "Generate and upload a new GHA runner image to OCI (Oracle Cloud Infrastructure)",
	RunE: run,
}
var args struct {
	debug         bool
	os            string
	osVersion     string
	arch          string
	bucketName    string
	compartmentId string
	namespace     string
	isoURL        string
	isoChecksum   string
	accelerator   string
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)

	if err := Cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(cmd *cobra.Command, argv []string) error {
	filepath := "images/ubuntu/templates/"
	sourceFile := fmt.Sprintf("%s%s-%s.pkr.hcl", filepath, args.os, args.osVersion)
	imageFile := fmt.Sprintf("build/image.raw")
	filename := ""
	imageName := fmt.Sprintf("%s-%s-%s-gha-image", args.os, args.osVersion, args.arch)

	githubClient := github.NewClient(nil)
	releases, _, err := githubClient.Repositories.ListReleases(context.Background(), "actions", "runner-images", nil)
	if err != nil {
		log.Fatalf("Failed to list releases: %s\n", err)
	}
	for _, release := range releases {
		if *release.Prerelease {
			continue
		}
		if strings.Contains(strings.ToLower(release.GetName()), strings.ToLower(fmt.Sprintf("%s %s", args.os, args.osVersion))) {
			log.Printf("Found %s %s release: %s\n", args.os, args.osVersion, release.GetTagName())
			downloadURL := release.GetTarballURL()

			if exists, _ := imageExists(imageName, release.GetTagName()); exists {
				log.Println("Image already exists.")
				return nil
			}

			log.Printf("Download URL: %s\n", downloadURL)

			filename, err = extractPackerFileFromURL(downloadURL, sourceFile)
			if err != nil {
				log.Fatalf("Failed to extract packer file: %s\n", err)
			}
			selectedRelease = release
			break
		}
	}

	file, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %s\n", err)
	}

	for key, value := range replacements {
		log.Printf("Replacing %s with %s\n", key, value)
		file = bytes.ReplaceAll(file, []byte(key), []byte(value))
	}

	newFile := fmt.Sprintf("%s-replaced.pkr.hcl", filename)

	out, err := os.Create(newFile)
	if err != nil {
		log.Fatalf("Failed to create file: %s\n", err)
	}
	defer out.Close()
	_, err = out.Write(file)
	if err != nil {
		log.Fatalf("Failed to write file: %s\n", err)
	}
	log.Printf("Replaced file written to: %s\n", out.Name())

	// Clean up the downloaded tarball
	err = os.Remove("images-release.tar.gz")
	if err != nil {
		log.Fatalf("Failed to remove tarball: %s\n", err)
	}

	command := exec.Command("packer", "build", "-var", "architecture="+args.arch, "--only", "qemu.img", newFile)

	command.Stdout = os.Stdout
	if err := command.Run(); err != nil {
		log.Print(command.String())
		log.Fatal("could not run command: ", err)
	}
	log.Printf("Packer build completed successfully.\n")

	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	command = exec.Command("oci", "os", "object", "put", "--parallel-upload-count", "100", "--bucket-name", args.bucketName, "--name", fmt.Sprintf("ubuntu-gha-image-%s", timestamp), "--file", imageFile)

	command.Stdout = os.Stdout
	if err := command.Run(); err != nil {
		log.Print(command.String())
		log.Fatal("could not run command: ", err)
	}

	command = exec.Command("oci", "compute", "image", "import", "from-object", "--bucket-name", args.bucketName, "--compartment-id", args.compartmentId, "--namespace", args.namespace, "--operating-system", imageName, "--display-name", imageName, "--name", fmt.Sprintf("ubuntu-gha-image-%s", timestamp), "--operating-system-version", *selectedRelease.TagName, "--launch-mode", "PARAVIRTUALIZED")
	command.Stdout = os.Stdout
	if err := command.Run(); err != nil {
		log.Print(command.String())
		log.Fatal("could not run command: ", err)
	}

	log.Println("New Ubuntu 24.04 image created successfully.")
	return nil
}

func imageExists(imageName, imageVersion string) (bool, error) {
	command := exec.Command("oci", "compute", "image", "list", "--compartment-id", args.compartmentId, "--operating-system", imageName, "--operating-system-version", imageVersion)
	output, err := command.Output()

	image := &core.Image{}
	if err != nil {
		log.Print(command.String())
		log.Fatal("could not run command: ", err)
		return false, err
	}
	err = json.Unmarshal(output, image)
	if err != nil || image.OperatingSystem != &imageName || image.OperatingSystemVersion != &imageVersion {
		return false, err
	}

	return true, nil
}

func extractPackerFileFromURL(url string, path string) (string, error) {
	// Extract the packer file from the URL
	tarball := "images-release.tar.gz"

	out, err := os.Create(tarball)
	if err != nil {
		return "", err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", err
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return extractPackerFileFromTarball(tarball, path)
}

func extractPackerFileFromTarball(tarballPath string, path string) (string, error) {
	f, err := os.Open(tarballPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer gzf.Close()

	// Create a new tar reader
	tarReader := tar.NewReader(gzf)
	filename := ""

	// Iterate through the files in the tarball
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of tarball
		}
		if err != nil {
			return "", err
		}
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(header.Name); err != nil {
				if err := os.MkdirAll(header.Name, 0755); err != nil {
					return "", err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			if strings.Contains(header.Name, path) {
				filename = header.Name
			}
			f, err := os.OpenFile(header.Name, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}

			// copy over contents
			if _, err := io.Copy(f, tarReader); err != nil {
				return "", err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
	if filename != "" {
		return filename, nil
	}

	return "", fmt.Errorf("%s not found in tarball", path)
}

func init() {
	flags := Cmd.Flags()

	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug logging",
	)
	flags.StringVar(
		&args.os,
		"os",
		"ubuntu",
		"Operating System",
	)

	flags.StringVar(
		&args.osVersion,
		"os-version",
		"24.04",
		"Operating System Version",
	)

	flags.StringVar(
		&args.arch,
		"arch",
		"x86",
		"Architecture",
	)

	flags.StringVar(
		&args.bucketName,
		"bucketName",
		"bucket-20250428-1925",
		"Oracle Cloud Infrastructure bucket name",
	)

	flags.StringVar(
		&args.compartmentId,
		"compartmentId",
		"ocid1.compartment.oc1..aaaaaaaa22icap66vxktktubjlhf6oxvfhev6n7udgje2chahyrtq65ga63a",
		"Oracle Cloud Infrastructure compartment ID",
	)

	flags.StringVar(
		&args.namespace,
		"namespace",
		"axtwf1hkrwcy",
		"Oracle Cloud Infrastructure namespace",
	)

	flags.StringVar(
		&args.isoURL,
		"isoURL",
		"https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
		"ISO URL for Packer to use",
	)

	flags.StringVar(
		&args.isoChecksum,
		"isoChecksum",
		"file:https://cloud-images.ubuntu.com/noble/current/SHA256SUMS",
		"ISO Checksum for Packer to use",
	)

	flags.StringVar(
		&args.accelerator,
		"accelerator",
		"kvm",
		"Accelerator for Packer to use (amd64: kvm | arm64: tcg)",
	)

	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	Cmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "prom"}, cobra.ShellCompDirectiveDefault
	})

	replacements[`dynamic "azure_tag" {
    for_each = var.azure_tags
    content {
      name = azure_tag.key
      value = azure_tag.value
    }
  }
}`] = fmt.Sprintf(`dynamic "azure_tag" {
    for_each = var.azure_tags
    content {
      name = azure_tag.key
      value = azure_tag.value
    }
  }
}

variable architecture {
	type        = string
	default     = "amd64"
	description = "Target architecture (amd64 or arm64)"
}

source "qemu" "img" {
	qemu_binary          = var.architecture == "arm64" ? "/usr/bin/qemu-system-aarch64" : "/usr/bin/qemu-system-x86_64"
	qemuargs             = var.architecture == "arm64" ? [
								["-machine", "virt"],
								["-cpu", "cortex-a57"],
								["-bios", "/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"]
							] : []
	vm_name              = "image.raw"
	cd_files             = ["./cloud-init/*"]
	cd_label             = "cidata"
	disk_compression     = true
	disk_image           = true
	iso_url              = "%s"
	iso_checksum         = "%s"
	memory               = 12000
	cpus                 = 6
	output_directory     = "build/"
	accelerator          = "%s"
	disk_size            = "80G"
	disk_interface       = "virtio"
	format               = "raw"
	net_device           = "virtio-net"
	boot_wait            = "15s"
	shutdown_command     = "echo 'packer' | sudo -S shutdown -P now"
	ssh_username         = "ubuntu"
	ssh_password         = "ubuntu"
	ssh_timeout          = "60m"
	headless             = true
}`, args.isoURL, args.isoChecksum, args.accelerator)

	replacements[`sources = ["source.azure-arm.build_image"]`] = `sources = ["source.azure-arm.build_image", "source.qemu.img"]
	provisioner "shell" {
	  execute_command = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
      inline = ["touch /etc/waagent.conf"]
	}`

	replacements[`["sleep 30", "/usr/sbin/waagent -force -deprovision+user && export HISTSIZE=0 && sync"]`] = `["sleep 30", "export HISTSIZE=0 && sync"]`

	// At this point this is the only Ubuntu-specific hard coded blocks we have left.
	replacements[`destination = "${path.root}/../Ubuntu2404-Readme.md"`] = `only = ["azure-arm.build_image"]
    destination = "${path.root}/../Ubuntu2404-Readme.md"`

	replacements[`destination = "${path.root}/../software-report.json"`] = `only = ["azure-arm.build_image"]
    destination = "${path.root}/../software-report.json"`

}
