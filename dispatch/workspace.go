package dispatch

import (
	"fmt"
	"os"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

func reportErr(err error, activity string) {
	fmt.Printf(" ! Failed to %s\n\n", activity)
	fmt.Print(err)
	fmt.Print("\n")
	os.Exit(1)
}

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
		fmt.Printf(" + Creating RSA key %s\n", keyFile)
		key, err := rsa.GenerateKey(rand.Reader, bitSize)
		if err != nil {
			reportErr(err, "create RSA key")
		}

		pub := key.Public()

		keyPEM := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(key),
			},
		)

		pubPEM := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PUBLIC KEY",
				Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
			},
		)

		if err := ioutil.WriteFile(keyFile, keyPEM, 0600); err != nil {
			reportErr(err, "save private key")
		}

		if err := ioutil.WriteFile(keyFile+".pub", pubPEM, 0644); err != nil {
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
