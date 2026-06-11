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
	"strings"
	"time"

	"github.com/google/go-github/v71/github"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/spf13/cobra"
)

var replacements = make(map[string]string)
var selectedRelease *github.RepositoryRelease
var Cmd = &cobra.Command{
	Use:  "gha-runner-vm-oci",
	Long: "Generate and upload a new GHA runner image to OCI using the oracle-oci Packer builder",
	RunE: run,
}
var args struct {
	debug              bool
	os                 string
	osVersion          string
	arch               string
	compartmentId      string
	baseImageOCID      string
	availabilityDomain string
	subnetOCID         string
	ociShape           string
	ociRegion          string
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
	// 20250714: They changed it to images/ubuntu/templates/build.ubuntu-24_04.pkr.hcl
	filepath := "images/ubuntu/templates/"
	sourceFile := fmt.Sprintf("%sbuild.%s-%s-%s.pkr.hcl", filepath, args.os, strings.ReplaceAll(args.osVersion, ".", "_"), args.arch)
	varsFile := "variable.ubuntu.pkr.hcl"
	filename := ""
	varsFilename := ""
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
				if os.Getenv("GITHUB_PERIODIC") == "true" {
					log.Println("Image already exists.")
					return nil
				}
			}

			log.Printf("Download URL: %s\n", downloadURL)

			filename, err = extractPackerFileFromURL(downloadURL, sourceFile)
			if err != nil {
				log.Fatalf("Failed to extract packer file: %s\n", err)
			}
			varsFilename, err = extractPackerFileFromURL(downloadURL, varsFile)
			if err != nil {
				log.Fatalf("Failed to extract packer file: %s\n", err)
			}
			selectedRelease = release
			break
		}
	}

	pkrContent, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %s\n", err)
	}

	// Read vars file
	varsContent, err := os.ReadFile(varsFilename)
	if err != nil {
		log.Fatalf("Failed to open vars file: %s\n", err)
	}

	// Set oracle-oci source block now that imageName is available
	regionLine := ""
	if args.ociRegion != "" {
		regionLine = fmt.Sprintf("\n  region               = \"%s\"", args.ociRegion)
	}
	sourceBlock := fmt.Sprintf(`source "oracle-oci" "img" {
  availability_domain  = "%s"
  base_image_ocid      = "%s"
  compartment_ocid     = "%s"
  shape                = "%s"
  subnet_ocid          = "%s"%s
  image_name           = "%s"
  instance_name        = "packer-build"
  disk_size            = 80
  ssh_username         = "ubuntu"
  communicator         = "ssh"
  image_launch_mode	   = "PARAVIRTUALIZED"
  shape_config {
    ocpus = 16
    memory_in_gbs = 64
  }
}`, args.availabilityDomain, args.baseImageOCID, args.compartmentId, args.ociShape, args.subnetOCID, regionLine, imageName)

	for key, value := range replacements {
		log.Printf("Replacing %s with %s\n", key, value)
		pkrContent = bytes.ReplaceAll(pkrContent, []byte(key), []byte(value))
	}

	// Second pass: replace %%SOURCEBLOCK%% placeholder (must happen after the build block replacement inserts it)
	pkrContent = bytes.ReplaceAll(pkrContent, []byte("%%SOURCEBLOCK%%"), []byte(sourceBlock))

	mergedContent := append(varsContent, []byte("\n")...)
	mergedContent = append(mergedContent, pkrContent...)

	// Add required_plugins block for oracle-oci
	requiredPlugins := `
packer {
  required_plugins {
    oracle = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/oracle"
    }
  }
}
`
	mergedContent = append([]byte(requiredPlugins), mergedContent...)

	// Append manifest post-processor before the closing brace of the build block
	idx := bytes.LastIndex(mergedContent, []byte("}"))
	if idx != -1 {
		postProc := []byte(`
  post-processor "manifest" {
    output     = "packer-manifest.json"
    strip_path = true
  }
`)
		mergedContent = append(mergedContent[:idx], append(postProc, mergedContent[idx:]...)...)
	}

	newFile := fmt.Sprintf("%s-replaced.pkr.hcl", filename)

	out, err := os.Create(newFile)
	if err != nil {
		log.Fatalf("Failed to create file: %s\n", err)
	}
	defer out.Close()
	_, err = out.Write(mergedContent)
	if err != nil {
		log.Fatalf("Failed to write file: %s\n", err)
	}
	log.Printf("Replaced file written to: %s\n", out.Name())

	// Clean up the downloaded tarball
	err = os.Remove("images-release.tar.gz")
	if err != nil {
		log.Fatalf("Failed to remove tarball: %s\n", err)
	}

	baseDir := strings.Split(filename, "/")[0]
	installRunnerPackage(baseDir)

	command := exec.Command("packer", "build", "-var", "architecture=arm64", newFile)

	command.Stdout = os.Stdout
	if err := command.Run(); err != nil {
		log.Print(command.String())
		log.Fatal("could not run command: ", err)
	}
	log.Printf("Packer build completed successfully.\n")

	// oracle-oci builder creates the image directly in OCI.
	// Read the manifest to get the image OCID.
	manifestData, err := os.ReadFile("packer-manifest.json")
	if err != nil {
		log.Fatal("failed to read packer manifest: ", err)
	}
	var manifest struct {
		Builds []struct {
			ArtifactID string `json:"artifact_id"`
		} `json:"builds"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		log.Fatal("failed to parse packer manifest: ", err)
	}
	if len(manifest.Builds) == 0 {
		log.Fatal("no builds found in packer manifest")
	}
	imageID := manifest.Builds[len(manifest.Builds)-1].ArtifactID
	log.Printf("Oracle OCI image created: %s\n", imageID)

	// Rename to rc- prefix (release-candidate) -- github action will update it if tests are successful.
	command = exec.Command("oci", "compute", "image", "update", "--image-id", imageID, "--operating-system", "rc-"+imageName, "--display-name", "rc-"+imageName, "--operating-system-version", *selectedRelease.TagName, "--force", "--region", args.ociRegion)
	rcOutput, err := command.CombinedOutput()
	if err != nil {
		log.Printf("OCI update command failed. Output:\n%s", string(rcOutput))
		log.Fatal("could not run command: ", err)
	}

	// expose Image Id to GitHub action
	f2, _ := os.OpenFile("/tmp/image_ocid", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	defer f2.Close()
	fmt.Fprintln(f2, imageID)

	// expose OS Image Tag to GitHub action
	f3, _ := os.OpenFile("/tmp/image_tag", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	defer f3.Close()
	fmt.Fprintln(f3, *selectedRelease.TagName)

	for {
		state, err := getImageState(imageID)
		if err != nil {
			log.Println("Error checking image state:", err)
		} else {
			log.Println("Current lifecycle-state:", state)
			if state == "AVAILABLE" {
				log.Printf("Image %s is AVAILABLE!", imageID)
				break
			}
		}
		log.Println("Waiting for 60 seconds before retrying...")
		time.Sleep(60 * time.Second)
	}

	log.Println("New Ubuntu 24.04 image created successfully.")
	return nil
}

func getImageState(imageID string) (string, error) {
	command := exec.Command("oci", "compute", "image", "get", "--image-id", imageID, "--region", args.ociRegion)

	output, err := command.Output()
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
	command := exec.Command("oci", "compute", "image", "list", "--compartment-id", args.compartmentId, "--operating-system", imageName, "--operating-system-version", imageVersion, "--region", args.ociRegion)
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

	if len(output) == 0 {
		return false, fmt.Errorf("could not find image")
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

func installRunnerPackage(baseDir string) error {
	// MS removed the runner package installation script. This one puts it back manually
	log.Println("Creating runner package installation script...")

	scriptContent := `#!/bin/bash -e
################################################################################
##  File:  install-runner-package.sh
##  Desc:  Download and Install runner package
################################################################################

# Source the helpers for use with the script
source $HELPER_SCRIPTS/install.sh

download_url=$(resolve_github_release_asset_url "actions/runner" 'test("actions-runner-linux-arm64-[0-9]+\\.[0-9]{3}\\.[0-9]+\\.tar\\.gz$")' "latest")
archive_name="${download_url##*/}"
archive_path=$(download_with_retry "$download_url")

mkdir -p /opt/runner-cache
mv "$archive_path" "/opt/runner-cache/$archive_name"
`

	// Write the script to the scripts directory where packer expects it
	scriptPath := baseDir + "/images/ubuntu/scripts/build/install-runner-package.sh"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to create install-runner-package.sh: %w", err)
	}

	log.Println("Runner package installation script created successfully")
	return nil
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
		"arm64",
		"Architecture",
	)

	flags.StringVar(
		&args.compartmentId,
		"compartmentId",
		"ocid1.compartment.oc1..aaaaaaaa22icap66vxktktubjlhf6oxvfhev6n7udgje2chahyrtq65ga63a",
		"Oracle Cloud Infrastructure compartment ID",
	)

	flags.StringVar(
		&args.baseImageOCID,
		"base-image-ocid",
		"",
		"OCI base image OCID (e.g. Ubuntu platform image)",
	)

	flags.StringVar(
		&args.availabilityDomain,
		"availability-domain",
		"bzBe:US-SANJOSE-1-AD-1",
		"OCI availability domain for the build instance",
	)

	flags.StringVar(
		&args.subnetOCID,
		"subnet-ocid",
		"ocid1.subnet.oc1.us-sanjose-1.aaaaaaaahgdslvujnywu3hvhqbvgz23souseseozvypng7ehnxgcotislubq",
		"OCI subnet OCID for the build instance",
	)

	flags.StringVar(
		&args.ociShape,
		"oci-shape",
		"VM.Standard.A1.Flex",
		"OCI instance shape for the build instance",
	)

	flags.StringVar(
		&args.ociRegion,
		"oci-region",
		"us-sanjose-1",
		"OCI region (optional, uses SDK default if empty)",
	)

	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	Cmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "prom"}, cobra.ShellCompDirectiveDefault
	})

	replacements[`build {
  sources = ["source.azure-arm.image"]`] = `variable architecture {
  type        = string
  default     = "amd64"
  description = "Target architecture (amd64 or arm64)"
}

%%SOURCEBLOCK%%

build {
  sources = ["source.oracle-oci.img"]

  provisioner "shell" {
    execute_command = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
    inline = ["touch /etc/waagent.conf"]
  }`

	replacements[`provisioner "shell" {
    environment_vars = ["IMAGE_VERSION=${var.image_version}", "INSTALLER_SCRIPT_FOLDER=${var.installer_script_folder}"]`] = `  provisioner "shell" {
    environment_vars = ["IMAGE_VERSION=${var.image_version}", "INSTALLER_SCRIPT_FOLDER=${var.installer_script_folder}", "CODEQL_JAVA_HOME=/usr/lib/jvm/temurin-21-jdk-arm64"]`

	// Remove edge installation, there is no arm build from Microsoft
	replacements[`"${path.root}/../scripts/build/install-microsoft-edge.sh",`] = ``

	// Remove chrome installation, there is no arm build from Google
	replacements[`"${path.root}/../scripts/build/install-google-chrome.sh",`] = ``

	replacements[`"${path.root}/../scripts/build/install-actions-cache.sh",`] = `"${path.root}/../scripts/build/install-actions-cache.sh",
				"${path.root}/../scripts/build/install-runner-package.sh",`

	replacements[`sources = ["source.azure-arm.build_image"]`] = `sources = ["source.azure-arm.build_image", "source.oracle-oci.img"]
		provisioner "shell" {
			execute_command = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
				inline = ["touch /etc/waagent.conf"]
		}`

	replacements[`["sleep 30", "/usr/sbin/waagent -force -deprovision+user && export HISTSIZE=0 && sync"]`] = `[
      "sleep 30",
      "export HISTSIZE=0 && sync",
      "usermod -aG docker ubuntu",
      "apt install -y libelf-dev",
      "rm -rf /var/lib/apt/lists/*"
    ]`
	// At this point this is the only Ubuntu-specific hard coded blocks we have left.
	replacements[`destination = "${path.root}/../Ubuntu2404-Readme.md"`] = `only = ["azure-arm.build_image"]
			destination = "${path.root}/../Ubuntu2404-Readme.md"`

	replacements[`destination = "${path.root}/../software-report.json"`] = `only = ["azure-arm.build_image"]
			destination = "${path.root}/../software-report.json"`
}
