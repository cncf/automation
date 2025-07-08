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
	"regexp"
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
	imageFile := "build/image.raw"
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

	if args.arch == "arm64" {
		baseDir := strings.Split(filename, "/")[0]
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-azcopy.sh", "https://aka.ms/downloadazcopy-v10-linux", "https://github.com/Azure/azure-storage-azcopy/releases/download/v10.29.1/azcopy_linux_arm64_10.29.1.tar.gz")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-runner-package.sh", "actions-runner-linux-x64", "actions-runner-linux-arm64" )
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-bicep.sh", "linux-x64", "linux-arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-julia.sh", "x86_64", "aarch64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-miniconda.sh", "x86_64", "aarch64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-aws-tools.sh", "awscli-exe-linux-x86_64", "awscli-exe-linux-aarch64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-aws-tools.sh", "aws-sam-cli-linux-x86_64", "aws-sam-cli-linux-arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-aws-tools.sh", "ubuntu_64bit", "ubuntu_arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-cmake.sh", "x86_64", "aarch64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-pulumi.sh", "linux-x64", "linux-arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-dotnetcore-sdk.sh", "linux-x64", "linux-arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-firefox.sh", "linux64.tar.gz", "linux-aarch64.tar.gz")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-swift.sh", "\\$\\(lsb_release -rs\\)", "24.04-aarch64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-microsoft-edge.sh", "arch=amd64", "arch=arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-kubernetes-tools.sh", "linux-amd64", "linux-arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-container-tools.sh","http://archive.ubuntu.com/.*", "https://launchpadlibrarian.net/683466454/containernetworking-plugins_1.1.1+ds1-3build1_arm64.deb")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-container-tools.sh", "amd64.deb", "arm64.db")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-oras-cli.sh", "linux_amd64", "linux_arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-yq.sh", "linux_amd64", "linux_arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-docker.sh", "amd64", "arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-packer.sh", "amd64", "arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-terraform.sh", "amd64", "arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-aliyun-cli.sh", "amd64", "arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-github-cli.sh", "amd64", "arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-java-tools.sh", "amd64", "arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-pypy.sh", "x64", "aarch64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-codeql-bundle.sh", "/x64", "/arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/install-ninja.sh", "ninja-linux.zip", "ninja-linux-aarch64.zip")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/configure-dpkg.sh", "wget .*", "wget http://launchpadlibrarian.net/723810004/libicu74_74.2-1ubuntu3_arm64.deb")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/configure-dpkg.sh", "libicu70_70.1-2_amd64.deb", "libicu74_74.2-1ubuntu3_arm64.deb")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/build/configure-dpkg.sh", "EXPECTED_LIBICU_SHA512=.*", "EXPECTED_LIBICU_SHA512=f5bc20c081d5dc6642a066052e69982702cf4b8638f77719567f7f30f622aae59ec1c23cb17842532c141768460369c176ad079e4ed22d6f4436f4ad86f30f79")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/docs-gen/SoftwareReport.CachedTools.psm1", "x64", "aarch64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/docs-gen/SoftwareReport.Tools.psm1", "x64", "arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/docs-gen/Generate-SoftwareReport.ps1", "Import-Module \\(Join-Path \\$PSScriptRoot \"SoftwareReport.Browsers.psm1\"\\) -DisableNameChecking", "")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/docs-gen/Generate-SoftwareReport.ps1", "# Browsers and Drivers\n.*\n.*\n.*\n.*\n.*\n.*\n.*\n.*\n.*\n.*", "")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/docs-gen/Generate-SoftwareReport.ps1", "# Environment variables\n.*", "")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/scripts/tests/Browsers.Tests.ps1", "Describe \"Chrome\"(.|\n)*", "")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/toolsets/toolset-2404.json", "linux-amd64", "linux-arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/toolsets/toolset-2404.json", "linux-x86_64", "linux-aarch64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/toolsets/toolset-2404.json", "x64", "arm64")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/toolsets/toolset-2404.json", "\"PyPy\",\n            \"arch\": \"arm64\"", "\"PyPy\",\n            \"arch\": \"aarch64\"")
		replaceArmPackageLinks(baseDir, "/images/ubuntu/toolsets/toolset-2404.json", "\"Ruby\",\n            \"platform_version\": \"24.04\"", "\"Ruby\",\n            \"platform_version\": \"24.04-arm64\"")
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
	output, err := command.Output()
	if err != nil {
		log.Fatal("failed to run OCI command: ", err)
	}

	// Need to update arm64 image capabilities
	if args.arch == "arm64" {
		var result struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}

		if err := json.Unmarshal(output, &result); err != nil {
			log.Fatal("failed to parse JSON: ", err)
		}
		imageID := result.Data.ID

		for {
			state, err := getImageState(imageID)
			if err != nil {
				log.Println("Error checking image state:", err)
			} else {
				log.Println("Current lifecycle-state:", state)
				if state == "AVAILABLE" {
					log.Println("Image is AVAILABLE!")
					break
				}
			}
			log.Println("Waiting for 30 seconds before retrying...")
			time.Sleep(30 * time.Second)
		}

		// Update image capabilities
		replaceArmPackageLinks("/home/ubuntu/automation/ci/gha-runner-vm", "/capability-update.json", "REPLACE_IMAGE_ID", imageID)
		command = exec.Command("oci", "raw-request", "--http-method", "POST", "--target-uri", "https://iaas.us-sanjose-1.oraclecloud.com/20160918/computeImageCapabilitySchemas", "--request-body", "file:///home/ubuntu/automation/ci/gha-runner-vm/capability-update.json")
		output, err = command.CombinedOutput()
		if err != nil {
			log.Print(command.String())
			log.Printf("OCI command failed. Output:\n%s", string(output))
			log.Fatal("could not run command: ", err)
		}

		// Add VM.Standard.A1.Flex compatibility
		command = exec.Command("oci", "raw-request", "--http-method", "PUT", "--target-uri", "https://iaas.us-sanjose-1.oraclecloud.com/20160918/images/" + imageID + "/shapes/VM.Standard.A1.Flex", "--request-body", "{\"ocpuConstraints\":{\"min\":\"1\",\"max\":\"80\"},\"memoryConstraints\":{\"minInGBs\":\"1\",\"maxInGBs\":\"512\"},\"imageId\":\"" + imageID + "\",\"shape\":\"VM.Standard.A1.Flex\"}")
		output, err = command.CombinedOutput()
		if err != nil {
			log.Print(command.String())
			log.Printf("OCI command failed. Output:\n%s", string(output))
			log.Fatal("could not run command: ", err)
		}

		// Add BM.Standard.A1.160 compatibility
		command = exec.Command("oci", "raw-request", "--http-method", "PUT", "--target-uri", "https://iaas.us-sanjose-1.oraclecloud.com/20160918/images/" + imageID + "/shapes/BM.Standard.A1.160", "--request-body", "{\"imageId\":\"" + imageID + "\",\"shape\":\"BM.Standard.A1.160\"}")
		output, err = command.CombinedOutput()
		if err != nil {
			log.Print(command.String())
			log.Printf("OCI command failed. Output:\n%s", string(output))
			log.Fatal("could not run command: ", err)
		}

		// Remove other amd64/x86 compatibility
		removeList := []string{
			"BM.Standard2.52",
			"BM.DenseIO.E4.128",
			"BM.Standard.E4.128",
			"BM.Standard.E3.128",
			"BM.Standard.E2.64",
			"BM.DenseIO2.52",
			"VM.Standard.E5.Flex",
			"VM.Standard.E4.Flex",
			"VM.Standard.E3.Flex",
			"VM.Standard2.1",
			"VM.Standard2.2",
			"VM.Standard2.4",
			"VM.Standard2.8",
			"VM.Standard2.16",
			"VM.Standard2.24",
			"VM.Standard.E2.1",
			"VM.Standard.E2.2",
			"VM.Standard.E2.4",
			"VM.Standard.E2.8",
			"VM.Standard.E2.1.Micro",
			"VM.Standard3.Flex",
			"VM.DenseIO2.8",
			"VM.DenseIO2.16",
			"VM.DenseIO2.24",
		}

		for _, machine := range removeList {
			command = exec.Command("oci", "raw-request", "--http-method", "DELETE", "--target-uri", "https://iaas.us-sanjose-1.oraclecloud.com/20160918/images/" + imageID + "/shapes/" + machine, "--request-body", "{\"imageId\":\"" + imageID + "\"}")
			output, err := command.CombinedOutput()
			if err != nil {
				log.Print(command.String())
				log.Printf("OCI command failed. Output:\n%s", string(output))
				log.Fatal("could not run command: ", err)
			}
		}
	}

	log.Println("New Ubuntu 24.04 image created successfully.")
	return nil
}

func getImageState(imageID string) (string, error) {
	command := exec.Command("oci", "compute", "image", "get","--image-id", imageID)

	output, err := command.CombinedOutput()
	if err != nil {
		log.Printf("OCI command failed. Output:\n%s", string(output))
		return "", fmt.Errorf("failed to run OCI command: %w", err)
	}

	var result struct {
		Data struct {
			LifecycleState string `json:"lifecycle-state"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result.Data.LifecycleState, nil
}

func imageExists(imageName, imageVersion string) (bool, error) {
	command := exec.Command("oci", "compute", "image", "list", "--compartment-id", args.compartmentId, "--operating-system", imageName, "--operating-system-version", imageVersion)
	output, err := command.CombinedOutput()

	if err != nil {
		log.Print(command.String())
		log.Printf("OCI command failed. Output:\n%s", string(output))
		log.Fatal("could not run command: ", err)
		return false, err
	}

	var response struct {
		Data []core.Image `json:"data"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		log.Printf("Error unmarshalling OCI response: %v. Response was: %s", err, string(output))
		return false, fmt.Errorf("could not unmarshal OCI response: %w", err)
	}

	for _, image := range response.Data {
		if image.OperatingSystem != nil && *image.OperatingSystem == imageName && image.OperatingSystemVersion != nil && *image.OperatingSystemVersion == imageVersion {
			log.Printf("Found image: %s", *image.OperatingSystemVersion)
			return true, nil
		}
	}

	return true, nil
}

func replaceArmPackageLinks(baseDir string, filename string, searchString string, replaceString string) (string, error) {
	scriptName := baseDir + filename
	err:= replaceInFileRegex(scriptName, map[*regexp.Regexp]string{
			regexp.MustCompile(searchString):
				replaceString,
		})
	if err != nil {
		log.Fatalf("Failed to patch %s: %v", filename, err)
		return "", err
	}
	return "", nil
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

func replaceInFileRegex(path string, patterns map[*regexp.Regexp]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	for re, replacement := range patterns {
		content = re.ReplaceAllString(content, replacement)
	}
	return os.WriteFile(path, []byte(content), 0755)
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
	qemuargs             = var.architecture == "arm64" ? [["-machine", "virt"], ["-cpu", "host"], ["-accel", "kvm"]] : []
	efi_boot             = var.architecture == "arm64" ? true : false
	efi_firmware_code    = var.architecture == "arm64" ? "/usr/share/AAVMF/AAVMF_CODE.fd" : ""
	efi_firmware_vars    = var.architecture == "arm64" ? "/usr/share/AAVMF/AAVMF_VARS.fd" : ""
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
	accelerator          = "kvm"
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
}`, args.isoURL, args.isoChecksum)

	if args.arch == "arm64" {
		replacements[`provisioner "shell" {
    environment_vars = ["HELPER_SCRIPTS=${var.helper_script_folder}", "INSTALLER_SCRIPT_FOLDER=${var.installer_script_folder}"]
    execute_command  = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
    scripts          = ["${path.root}/../scripts/build/install-powershell.sh"]
  }`] = `  provisioner "shell" {
    environment_vars = ["HELPER_SCRIPTS=${var.helper_script_folder}", "INSTALLER_SCRIPT_FOLDER=${var.installer_script_folder}"]
    execute_command  = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
    inline           = [
			"curl -L -o /tmp/powershell.tar.gz https://github.com/PowerShell/PowerShell/releases/download/v7.5.1/powershell-7.5.1-linux-arm64.tar.gz",
			"mkdir -p /opt/microsoft/powershell/7",
			"tar zxf /tmp/powershell.tar.gz -C /opt/microsoft/powershell/7",
			"chmod +x /opt/microsoft/powershell/7/pwsh",
			"ln -s /opt/microsoft/powershell/7/pwsh /usr/bin/pwsh"
		]
  }`

		replacements[`provisioner "shell" {
    environment_vars = ["IMAGE_VERSION=${var.image_version}", "INSTALLER_SCRIPT_FOLDER=${var.installer_script_folder}"]`] = `  provisioner "shell" {
    environment_vars = ["IMAGE_VERSION=${var.image_version}", "INSTALLER_SCRIPT_FOLDER=${var.installer_script_folder}", "CODEQL_JAVA_HOME=/usr/lib/jvm/temurin-21-jdk-arm64"]`

		// Remove edge installation, there is no arm build from Microsoft
		replacements[`"${path.root}/../scripts/build/install-microsoft-edge.sh",`] = ``

		// Remove chrome installation, there is no arm build from Google
		replacements[`"${path.root}/../scripts/build/install-google-chrome.sh",`] = ``
  }

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
