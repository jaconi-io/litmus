package experiments

import (
	"context"

	"github.com/jaconi-io/litmus/environment"
	"google.golang.org/api/compute/v1"

	clients "github.com/litmuschaos/litmus-go/pkg/clients"
)

// gcpVMStopDetails extend the default experiment details.
type gcpVMStopDetails struct {
	environment.ExperimentDetails
	GCPInstance string `required:"true" split_words:"true"`
	GCPProject  string `required:"true" split_words:"true"`
	GCPZone     string `required:"true" split_words:"true"`
}

// GCPVMStop stops a virtual machine instance.
func GCPVMStop(clients clients.ClientSets) error {
	details := &gcpVMStopDetails{}
	experiment, err := NewExperiment("gcp-vm-stop", clients, details)
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

		return nil
	})
}
