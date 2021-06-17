package experiments

import (
	"context"

	"github.com/jaconi-io/litmus/environment"
	"google.golang.org/api/compute/v1"

	clients "github.com/litmuschaos/litmus-go/pkg/clients"
)

// gcpVMRestartDetails extend the default experiment details.
type gcpVMRestartDetails struct {
	environment.ExperimentDetails
	GCPInstance string `required:"true" split_words:"true"`
	GCPProject  string `required:"true" split_words:"true"`
	GCPZone     string `required:"true" split_words:"true"`
}

// GCPVMRestart restarts a virtual machine instance.
func GCPVMRestart(clients clients.ClientSets) error {
	details := &gcpVMRestartDetails{}
	experiment, err := NewExperiment("gcp-vm-restart", clients, details)
	if err != nil {
		return err
	}

	return experiment.Run(func(ctx context.Context) error {
		svc, err := compute.NewService(ctx)
		if err != nil {
			return err
		}

		_, err = svc.Instances.Stop(details.GCPProject, details.GCPZone, details.GCPInstance).Context(ctx).Do()
		if err != nil {
			return err
		}

		_, err = svc.Instances.Start(details.GCPProject, details.GCPZone, details.GCPInstance).Context(ctx).Do()
		if err != nil {
			return err
		}

		return nil
	})
}
