package dispatch

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func (i Instance) cliCreate() (Instance, error) {
	createCommand := flag.NewFlagSet("create", flag.ExitOnError)
	createName := createCommand.String("name", "", "cluster name")
	createSize := createCommand.String("size", "small", "cluster node size")
	nodeCount := createCommand.String("nodes", "2", "cluster node count")
	autoCreate := createCommand.Bool("yes", false, "skip verification prompt for cluster creation")

	err := createCommand.Parse(os.Args[2:])
	if err != nil {
		log.Error("Failed to parse create command.")
		return Instance{}, err
	}

	i.Action = createAction
	i.Name = strings.ToLower(*createName)
	i.Size = *createSize
	i.Count = *nodeCount
	i.Verified = *autoCreate

	return i, nil
}

func (i Instance) cliDelete() (Instance, error) {
	deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteName := deleteCommand.String("name", "", "cluster name")
	autoDelete := deleteCommand.Bool("yes", false, "skip verification prompt for cluster deletion")

	err := deleteCommand.Parse(os.Args[2:])
	if err != nil {
		log.Error("Failed to parse delete command.")
		return Instance{}, err
	}

	i.Action = deleteAction
	i.Name = strings.ToLower(*deleteName)
	i.Verified = *autoDelete

	return i, nil
}

func (i Instance) CLIWorkflow() (Instance, error) {
	var s Instance

	action := os.Args[1]

	switch action {
	case "-h":
		fmt.Printf("Dispatch options:\n dispatch create -h\n dispatch delete -h\n")
		os.Exit(0)

	case "version", "-v":
		fmt.Printf("Dispatch version: %s\n", version)
		os.Exit(0)

	case createAction:
		s, err := i.cliCreate()
		if err != nil {
			return s, err
		}

		if s.Name == "" {
			fmt.Println(" ! create events require the -name flag")
			os.Exit(0)
		} else {
			err := validateClusterName(s.Name)
			if err != nil {
				log.Error("Failed to provide valid cluster name")
				return s, err
			}
		}

		return s, err

	case deleteAction:
		s, err := i.cliDelete()
		if err != nil {
			return s, err
		}

		if s.Name == "" {
			fmt.Println(" ! delete events require the -name flag")
			os.Exit(0)
		} else {
			err := validateClusterName(s.Name)
			if err != nil {
				log.Error("Failed to provide valid cluster name")
				return s, err
			}
		}

		return s, err

	default:
		fmt.Printf(" ! %s is not a valid Dispatch option\n", action)
		fmt.Printf("\n Try 'dispatch create -h' or 'dispatch delete -h'\n")
		os.Exit(0)
	}

	return s, nil
}

func validateClusterName(name string) error {
	valid := regexp.MustCompile(`^[a-zA-Z][-a-zA-Z0-9]*`).MatchString(name)

	if !valid {
		return fmt.Errorf("invalid cluster name provided, '%s'\ncluster name must begin with a letter", name)
	}

	return nil
}
