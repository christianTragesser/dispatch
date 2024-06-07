package dispatch

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const binMode int = 0755
const privMode int = 0600
const pubMode int = 0644

type workspace struct {
	root       string
	kubeDir    string
	binPath    string
	pulumiPath string
}

func (w workspace) createWorkspace() (workspace, error) {
	home, homeSet := os.LookupEnv("HOME")

	if homeSet {
		w.root = filepath.Join(home, ".dispatch")
		w.kubeDir = filepath.Join(home, ".dispatch", ".kube")
		w.binPath = filepath.Join(home, ".dispatch", "bin", "pulumi")
		w.pulumiPath = filepath.Join(home, ".dispatch", "bin", "pulumi", pulumiVersion, runtime.GOOS)

		t := reflect.TypeOf(w)
		v := reflect.ValueOf(w)

		for i := 0; i < t.NumField(); i++ {
			err := createDirectory(v.Field(i).String())
			if err != nil {
				log.Error("Failed to create dispatch workspace.")
				return workspace{}, err
			}
		}

		err := w.createKubeConfig()
		if err != nil {
			log.Error("Failed to create Kubernetes config.")
			return workspace{}, err
		}

		err = w.installPulumi()
		if err != nil {
			log.Error("Failed to install pulumi.")
			return workspace{}, err
		}

		err = w.createDispatchConfig()
		if err != nil {
			log.Error("Failed to create dispatch config.")
			return workspace{}, err
		}
	} else {
		fmt.Print("$HOME environment variable not found, exiting.\n")
		os.Exit(1)
	}

	return w, nil
}

func (w workspace) createKubeConfig() error {
	configFile := w.kubeDir + "/config"

	_, err := os.Stat(configFile)

	if os.IsNotExist(err) {
		config, err := os.Create(configFile)
		if err != nil {
			log.Error("Failed to create kube config file")
			return err
		}

		config.Close()

		err = os.Chmod(configFile, fs.FileMode(privMode))
		if err != nil {
			log.Error("Failed to set file permissions for kube config")
			return err
		}
	}

	return nil
}

func (w workspace) installPulumi() error {
	var architecture string

	err := removePreviousPulumiBins(w.binPath)
	if err != nil {
		return nil
	}

	switch runtime.GOARCH {
	case "amd64":
		architecture = "x64"
	case "arm64":
		architecture = "arm64"
	default:
		return errors.New("unsupported architecture")
	}

	baseURL := "https://github.com/pulumi/pulumi/releases/download/v" + pulumiVersion + "/"
	artifactFile := "pulumi-v" + pulumiVersion + "-" + runtime.GOOS + "-" + architecture + ".tar.gz"
	tarURL := baseURL + artifactFile

	// check for omnibus install of pulumi
	_, err = os.Stat(filepath.Join(w.pulumiPath, "pulumi", "pulumi"))

	if os.IsNotExist(err) {
		fmt.Printf(" + Installing omnibus pulumi version %s\n", pulumiVersion)

		// create destination .tar.gz file
		tarGz, err := os.Create(w.pulumiPath + "/" + artifactFile)
		if err != nil {
			log.Error("Failed to create pulumi artifact destination file.")
			return err
		}
		defer tarGz.Close()

		// download pulumi release artifact
		resp, err := http.Get(tarURL)
		if err != nil {
			log.Error("Failed todownload pulumi artifact.")
			return err
		}
		defer resp.Body.Close()

		// write artifact archive file to destination file
		_, err = io.Copy(tarGz, resp.Body)
		if err != nil {
			log.Error("Failed to save pulumi bundle artifact.")
			return err
		}

		// extract pulumi archive file
		err = extractTarGz(filepath.Join(w.pulumiPath, artifactFile))
		if err != nil {
			log.Error("Failed to extract pulumi bundle artifact.")
			return err
		}

		err = os.Remove(filepath.Join(w.pulumiPath, artifactFile))
		if err != nil {
			log.Error("Failed to delete pulumi bundle artifact.")
			return err
		}
	} else {
		fmt.Printf(" . Found pulumi at %s\n", w.pulumiPath)
	}

	return nil
}

func (w workspace) createDispatchConfig() error {
	var dispatchUID string

	configFile := w.root + "/dispatch.conf"

	_, readErr := os.Stat(configFile)

	if os.IsNotExist(readErr) {
		fmt.Print(" + Please enter a user ID: ")
		fmt.Scanf("%s", &dispatchUID)

		if len(dispatchUID) == 0 {
			fmt.Println("   ! You must provide a user ID, exiting.")
			os.Exit(0)
		}

		configMap := map[string]string{"uid": dispatchUID}

		configData, err := yaml.Marshal(configMap)
		if err != nil {
			log.Error("Failed to set dispatch user ID.")
			return err
		}

		err = os.WriteFile(configFile, configData, fs.FileMode(pubMode))
		if err != nil {
			log.Error("Failed to write dispatch config file.")
			return err
		}
	} else {
		uid, err := getDispatchUserID(configFile)
		if err != nil {
			return err
		}

		fmt.Printf(" . Found user ID '%s'\n", uid)
	}

	return nil
}

func createDirectory(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		log.Error("Failed to create dispatch directory.")

		return err
	}

	return nil
}

func removePreviousPulumiBins(binPath string) error {
	installedVersions, err := os.ReadDir(binPath)
	if err != nil {
		log.Error("Failed to list pulumi binary directory.")
		return err
	}

	for _, v := range installedVersions {
		if v.Name() != pulumiVersion {
			fmt.Printf(" - removing version %s of omnibus pulumi\n", v.Name())

			err := os.RemoveAll(filepath.Join(binPath, v.Name()))
			if err != nil {
				log.Error("Failed to delete previous pulumi binary.")
			}
		}
	}

	return nil
}

func extractTarGz(archivePath string) error {
	fileStream, err := os.Open(archivePath)
	if err != nil {
		log.Error("Failed to open archive file.")
		return err
	}

	tarStream, err := gzip.NewReader(fileStream)
	if err != nil {
		log.Error("Failed to decompress tar.gz file.")
		return err
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
			log.Error("Failed to create extraction directory.")
			return err
		}
	}

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Error("Failed to untar archive file.")
			return err
		}

		if header == nil {
			continue
		}

		destinationTarget := filepath.Join(destinationPath, header.Name) // #nosec

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(destinationTarget); err != nil {
				if err := os.Mkdir(destinationTarget, fs.FileMode(binMode)); err != nil {
					log.Error("Failed to create archive directory.")
					return err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(destinationTarget, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				log.Error("Failed to open archive file destination.")
				return err
			}

			if _, err := io.Copy(f, tarReader); err != nil {
				log.Error("Failed to copy archive file contents.")
				return err
			} // #nosec

			f.Close()
		default:
			log.Error("Unknown archive type.")
		}
	}

	return err
}

func getDispatchUserID(configFile string) (string, error) {
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Error("Failed to read dispatch config file.")
		return "", err
	}

	configMap := make(map[string]string)
	err = yaml.Unmarshal(configData, &configMap)
	if err != nil {
		log.Error("Failed to retireve user ID from dispatch config file.")
		return "", err
	}

	return configMap["uid"], nil
}
