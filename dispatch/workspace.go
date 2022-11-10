package dispatch

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"
)

const binMode int = 0755
const privMode int = 0600
const pubMode int = 0644

type workspace struct {
	root       string
	ssh        string
	kube       string
	binPath    string
	pulumiPath string
}

func ensureDir(path string) {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		reportErr(err, "creating dispatch workspace")
	}
}

func ensureRSAKeys(sshDir string) {
	keyFile := sshDir + "/kops_rsa"
	bitSize := 4096

	ensureDir(sshDir)

	_, err := os.Stat(keyFile)

	if os.IsNotExist(err) {
		fmt.Printf(" + Creating RSA key %s for KOPS\n", keyFile)
		// Create private RSA key in PEM format
		key, err := rsa.GenerateKey(rand.Reader, bitSize)
		if err != nil {
			reportErr(err, "create RSA key")
		}

		err = key.Validate()
		if err != nil {
			reportErr(err, "validate private key")
		}

		keyPEM := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(key),
			},
		)

		// Create ssh-rsa public key
		pubRSAKey, err := ssh.NewPublicKey(&key.PublicKey)
		if err != nil {
			reportErr(err, "create public RSA key")
		}

		pubKeyBytes := ssh.MarshalAuthorizedKey(pubRSAKey)

		// Write RSA key pair to disk
		if err := os.WriteFile(keyFile, keyPEM, fs.FileMode(privMode)); err != nil {
			reportErr(err, "save private key")
		}

		if err := os.WriteFile(keyFile+".pub", pubKeyBytes, fs.FileMode(pubMode)); err != nil {
			reportErr(err, "save public key")
		}
	} else {
		fmt.Printf(" . Found %s RSA private key\n", keyFile)
	}
}

func ensureKubeConfig(kubeDir string) {
	configFile := kubeDir + "/config"

	ensureDir(kubeDir)

	_, err := os.Stat(configFile)

	if os.IsNotExist(err) {
		config, err := os.Create(configFile)
		if err != nil {
			reportErr(err, "create kube config file")
		}

		config.Close()

		err = os.Chmod(configFile, fs.FileMode(privMode))
		if err != nil {
			reportErr(err, "set file permissions for kube config")
		}
	}
}

func ensureDispatchConfig(dispatchDir string) string {
	var dispatchUID string

	configFile := dispatchDir + "/dispatch.conf"

	_, readErr := os.Stat(configFile)

	if os.IsNotExist(readErr) {
		fmt.Print(" + Please enter a user ID: ")
		fmt.Scanf("%s", &dispatchUID)

		configMap := map[string]string{"uid": dispatchUID}

		configData, err := yaml.Marshal(configMap)
		if err != nil {
			reportErr(err, "set UID")
		}

		writeErr := os.WriteFile(configFile, configData, fs.FileMode(pubMode))
		if writeErr != nil {
			reportErr(writeErr, "write Dispatch config file")
		}
	} else {
		configData, readErr := os.ReadFile(configFile)
		if readErr != nil {
			reportErr(readErr, "read Dispatch config file")
		}

		configMap := make(map[string]string)
		yamlErr := yaml.Unmarshal(configData, &configMap)
		if yamlErr != nil {
			reportErr(readErr, "set UID from config file")
		}

		dispatchUID = configMap["uid"]

		fmt.Printf(" . Found user ID '%s'\n", configMap["uid"])
	}

	return dispatchUID
}

func removePreviousPulumiBins(binPath string) {
	const preferredVersions = 1

	installedVersions, err := os.ReadDir(binPath)
	if err != nil {
		reportErr(err, "list pulumi binary directory")
	}

	for _, v := range installedVersions {
		if v.Name() != pulumiVersion {
			fmt.Printf(" - removing version %s of omnibus pulumi", v.Name())
			fmt.Println(filepath.Join(binPath, v.Name()))

			err := os.RemoveAll(filepath.Join(binPath, v.Name()))
			if err != nil {
				reportErr(err, "delete previous pulumi binary")
			}
		}
	}
}

func extractTarGz(archivePath string) {
	fileStream, err := os.Open(archivePath)
	if err != nil {
		reportErr(err, "open archive file")
	}

	tarStream, err := gzip.NewReader(fileStream)
	if err != nil {
		reportErr(err, "decompress gzip file")
	}
	defer tarStream.Close()

	tarReader := tar.NewReader(tarStream)

	// use archive path to set extraction location
	destinationSplit := strings.Split(archivePath, "/")
	artifactName := destinationSplit[len(destinationSplit)-1]

	if len(destinationSplit) > 0 {
		destinationSplit = destinationSplit[:len(destinationSplit)-1]
	}

	destinationPath := strings.Join(destinationSplit, "/")

	extractDir := filepath.Join(destinationPath, strings.Split(artifactName, "-")[0])

	if _, err := os.Stat(extractDir); err != nil {
		if err := os.Mkdir(extractDir, fs.FileMode(binMode)); err != nil {
			reportErr(err, "create extraction directory")
		}
	}

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			reportErr(err, "untar archive file")
		}

		if header == nil {
			continue
		}

		destinationTarget := filepath.Join(destinationPath, header.Name) // #nosec

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(destinationTarget); err != nil {
				if err := os.Mkdir(destinationTarget, fs.FileMode(binMode)); err != nil {
					reportErr(err, "create archive directory")
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(destinationTarget, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				reportErr(err, "open archive file destination")
			}

			if _, err := io.Copy(f, tarReader); err != nil {
				reportErr(err, "copy archive file contents")
			} // #nosec

			f.Close()
		default:
			reportErr(nil, "unknown type archive type")
		}
	}
}

func ensurePulumi(workDir workspace) {
	// it appears there isn't a pulumi binary for ARM?
	baseURL := "https://github.com/pulumi/pulumi/releases/download/v" + pulumiVersion + "/"
	artifactFile := "pulumi-v" + pulumiVersion + "-" + runtime.GOOS + "-x64.tar.gz"
	tarURL := baseURL + artifactFile

	ensureDir(workDir.pulumiPath)
	removePreviousPulumiBins(workDir.binPath)

	// check for omnibus install of pulumi
	_, err := os.Stat(filepath.Join(workDir.pulumiPath, "pulumi", "pulumi"))

	if os.IsNotExist(err) {
		fmt.Printf(" + Installing omnibus pulumi version %s\n", pulumiVersion)

		// create destination .tar.gz file
		tarGz, err := os.Create(workDir.pulumiPath + "/" + artifactFile)
		if err != nil {
			reportErr(err, "create pulumi artifact")
		}
		defer tarGz.Close()

		// download pulumi release artifact
		resp, err := http.Get(tarURL)
		if err != nil {
			reportErr(err, "download pulumi artifact")
		}
		defer resp.Body.Close()

		// write artifact archive file to destination file
		_, err = io.Copy(tarGz, resp.Body)
		if err != nil {
			reportErr(err, "save pulumi artifact")
		}

		// extract pulumi archive file
		extractTarGz(filepath.Join(workDir.pulumiPath, artifactFile))
	} else {
		fmt.Printf(" . Found pulumi at %s\n", workDir.pulumiPath)
	}
}

func ensureWorkspace() string {
	var dispatchUID string

	home, homeSet := os.LookupEnv("HOME")

	if homeSet {
		dispatchDir := filepath.Join(home, ".dispatch")

		sessionDirs := workspace{
			root:       dispatchDir,
			ssh:        filepath.Join(dispatchDir, ".ssh"),
			kube:       filepath.Join(dispatchDir, ".kube"),
			binPath:    filepath.Join(dispatchDir, "bin", "pulumi"),
			pulumiPath: filepath.Join(dispatchDir, "bin", "pulumi", pulumiVersion, runtime.GOOS),
		}

		ensureDir(sessionDirs.root)
		ensureRSAKeys(sessionDirs.ssh)
		ensureKubeConfig(sessionDirs.kube)
		ensurePulumi(sessionDirs)
		dispatchUID = ensureDispatchConfig(sessionDirs.root)
	} else {
		fmt.Print("$HOME environment variable not found, exiting.\n")
		os.Exit(1)
	}

	return dispatchUID
}
