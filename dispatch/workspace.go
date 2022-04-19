package dispatch

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"
)

const binMode int = 0755
const privMode int = 0600
const pubMode int = 0644

type workspace struct {
	root, ssh, kube, bin string
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
		if err := ioutil.WriteFile(keyFile, keyPEM, fs.FileMode(privMode)); err != nil {
			reportErr(err, "save private key")
		}

		if err := ioutil.WriteFile(keyFile+".pub", pubKeyBytes, fs.FileMode(pubMode)); err != nil {
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

		writeErr := ioutil.WriteFile(configFile, configData, fs.FileMode(pubMode))
		if writeErr != nil {
			reportErr(writeErr, "write Dispatch config file")
		}
	} else {
		configData, readErr := ioutil.ReadFile(configFile)
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

func ensureKOPS(binDir string) {
	kopsDLPath := "https://github.com/kubernetes/kops/releases/download/v"
	kopsURL := kopsDLPath + kopsVersion + "/kops-" + runtime.GOOS + "-" + runtime.GOARCH
	kopsBin := binDir + "/kops"

	ensureDir(binDir)

	_, err := os.Stat(kopsBin)

	if os.IsNotExist(err) {
		fmt.Printf(" + Downloading kOps v%s\n", kopsVersion)

		resp, err := http.Get(kopsURL)

		if err != nil {
			reportErr(err, "download kOps")
		}
		defer resp.Body.Close()

		fileHandle, err := os.OpenFile(kopsBin, os.O_CREATE|os.O_APPEND|os.O_RDWR, fs.FileMode(pubMode))
		if err != nil {
			reportErr(err, "buffer kOps download")
		}
		defer fileHandle.Close()

		_, err = io.Copy(fileHandle, resp.Body)
		if err != nil {
			reportErr(err, "save kOps binary")
		}

		err = os.Chmod(kopsBin, fs.FileMode(binMode))
		if err != nil {
			reportErr(err, "set file permissions for kOps binary")
		}
	} else {
		fmt.Printf(" . Found kOps at %s\n", kopsBin)
	}
}

func ensureWorkspace() string {
	var dispatchUID string

	home, homeSet := os.LookupEnv("HOME")

	if homeSet {
		dispatchDir := home + "/.dispatch"

		sessionDirs := workspace{
			root: dispatchDir,
			ssh:  dispatchDir + "/.ssh",
			kube: dispatchDir + "/.kube",
			bin:  dispatchDir + "/bin/" + kopsVersion + "/" + runtime.GOOS,
		}

		ensureDir(sessionDirs.root)
		ensureRSAKeys(sessionDirs.ssh)
		ensureKubeConfig(sessionDirs.kube)
		ensureKOPS(sessionDirs.bin)
		dispatchUID = ensureDispatchConfig(sessionDirs.root)
	} else {
		fmt.Print("$HOME environment variable not found, exiting.\n")
		os.Exit(1)
	}

	return dispatchUID
}
