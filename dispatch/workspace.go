package dispatch

import (
	"fmt"
	"os"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"
)

func ensureDirs(paths [3]string) {
	for _, path := range paths {
		err := os.Mkdir(path, os.ModePerm)
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
			ReportErr(err, "create RSA key")
		}

		err = key.Validate()
		if err != nil {
			ReportErr(err, "validate private key")
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
			ReportErr(err, "create public RSA key")
		}

		pubKeyBytes := ssh.MarshalAuthorizedKey(pubRSAKey)

		// Write RSA key pair to disk
		if err := ioutil.WriteFile(keyFile, keyPEM, 0600); err != nil {
			ReportErr(err, "save private key")
		}

		if err := ioutil.WriteFile(keyFile+".pub", pubKeyBytes, 0644); err != nil {
			ReportErr(err, "save public key")
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
			ReportErr(err, "create kube config file")
		}
		config.Close()

		os.Chmod(configFile, 0600)
	}

}

func ensureDispatchConfig(homeDir string) string {
	var dispatchUID string
	configFile := homeDir + "/.config"

	_, readErr := os.Stat(configFile)

	if os.IsNotExist(readErr) {
		fmt.Print("\n + Please enter a user ID: ")
		fmt.Scanf("%s", &dispatchUID)

		configMap := map[string]string{"uid": dispatchUID}

		configData, err := yaml.Marshal(configMap)
		if err != nil {
			ReportErr(err, "set UID")
		}

		writeErr := ioutil.WriteFile(configFile, configData, 0644)
		if writeErr != nil {
			ReportErr(writeErr, "write Dispatch config file")
		}
	} else {
		configData, readErr := ioutil.ReadFile(configFile)
		if readErr != nil {
			ReportErr(readErr, "read Dispatch config file")
		}

		configMap := make(map[string]string)
		yamlErr := yaml.Unmarshal(configData, &configMap)
		if yamlErr != nil {
			ReportErr(readErr, "set UID from config file")
		}

		dispatchUID = configMap["uid"]

		fmt.Printf(" . Found Dispatch UID '%s'\n\n", configMap["uid"])

	}

	return dispatchUID
}

func EnsureWorkspace() string {
	fmt.Print("Ensuring workspace:\n")
	var dispatchUID string
	home, homeSet := os.LookupEnv("HOME")

	if homeSet {
		dispatchDir := home + "/.dispatch"
		workspaceDirs := [3]string{
			dispatchDir,
			dispatchDir + "/.ssh",
			dispatchDir + "/.kube",
		}

		ensureDirs(workspaceDirs)
		ensureRSAKeys(workspaceDirs[1])
		ensureKubeConfig(workspaceDirs[2])
		dispatchUID = ensureDispatchConfig(workspaceDirs[0])

	} else {
		fmt.Print("$HOME environment variable not found, exiting.\n")
		os.Exit(1)
	}

	return dispatchUID
}
