package dispatch

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"
)

func ensureDirs(paths [4]string) {
	for _, path := range paths {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			fmt.Printf(" . Found %s\n", path)
		}
	}
}

func ensureRSAKeys(sshDir string) {
	keyFile := sshDir + "/kops_rsa"
	bitSize := 4096

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
		if err := ioutil.WriteFile(keyFile, keyPEM, 0600); err != nil {
			reportErr(err, "save private key")
		}

		if err := ioutil.WriteFile(keyFile+".pub", pubKeyBytes, 0644); err != nil {
			reportErr(err, "save public key")
		}
	} else {
		fmt.Printf(" . Found %s RSA private key\n", keyFile)
	}
}

func ensureKubeConfig(kubeDir string) {
	configFile := kubeDir + "/config"

	_, err := os.Stat(configFile)

	if os.IsNotExist(err) {
		config, err := os.Create(configFile)
		if err != nil {
			reportErr(err, "create kube config file")
		}
		config.Close()

		os.Chmod(configFile, 0600)
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

		writeErr := ioutil.WriteFile(configFile, configData, 0644)
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
	kopsURL := "https://github.com/kubernetes/kops/releases/download/v" + KOPS_VERSION + "/kops-" + runtime.GOOS + "-" + runtime.GOARCH
	kopsBin := binDir + "/kops"

	_, err := os.Stat(kopsBin)

	if os.IsNotExist(err) {
		fmt.Printf(" + Downloading kOps v%s\n", KOPS_VERSION)
		resp, err := http.Get(kopsURL)
		if err != nil {
			reportErr(err, "download kOps")
		}
		defer resp.Body.Close()

		fileHandle, err := os.OpenFile(kopsBin, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
		if err != nil {
			reportErr(err, "buffer kOps download")
		}
		defer fileHandle.Close()

		_, err = io.Copy(fileHandle, resp.Body)
		if err != nil {
			reportErr(err, "save kOps binary")
		}

		os.Chmod(kopsBin, 0750)

	} else {
		fmt.Printf(" . Found kOps at %s\n", kopsBin)
	}

}

func ensureWorkspace() string {
	var dispatchUID string

	home, homeSet := os.LookupEnv("HOME")

	if homeSet {
		dispatchDir := home + "/.dispatch"
		workspaceDirs := [4]string{
			dispatchDir,
			dispatchDir + "/.ssh",
			dispatchDir + "/.kube",
			dispatchDir + "/bin/" + KOPS_VERSION,
		}

		ensureDirs(workspaceDirs)
		ensureRSAKeys(workspaceDirs[1])
		ensureKubeConfig(workspaceDirs[2])
		ensureKOPS(workspaceDirs[3])
		dispatchUID = ensureDispatchConfig(workspaceDirs[0])

	} else {
		fmt.Print("$HOME environment variable not found, exiting.\n")
		os.Exit(1)
	}

	return dispatchUID
}
