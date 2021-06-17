package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jaconi-io/litmus/experiments"

	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var exps = map[string]func(clients.ClientSets) error{
	"gcp-vm-stop":    experiments.GCPVMStop,
	"gcp-vm-restart": experiments.GCPVMRestart,
}

func main() {
	// Create a slice of experiment names.
	keys := make([]string, len(exps))
	i := 0
	for key := range exps {
		keys[i] = key
		i++
	}

	// Get the experiment name from a command line flag.
	experiment := flag.String("experiment", keys[0], fmt.Sprintf("name of the experiment [%s]", strings.Join(keys, ", ")))

	clients := clients.ClientSets{}
	if err := clients.GenerateClientSetFromKubeConfig(); err != nil {
		log.Fatalf("failed to generate clients from kubernetes configuration: %v", err)
		return
	}

	values := map[string]interface{}{"experiment": *experiment}
	if f, ok := exps[*experiment]; ok {
		log.InfoWithValues("exection started", values)
		err := f(clients)
		if err != nil {
			log.ErrorWithValues(fmt.Sprintf("execution failed: %v", err), values)
			os.Exit(1)
		}
	} else {
		log.ErrorWithValues(fmt.Sprintf("unknown experiment %q; you might be using the wrong image", *experiment), values)
		os.Exit(2)
	}
}
