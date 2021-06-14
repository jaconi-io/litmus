package experiments

import (
	"context"
	"errors"
	"fmt"

	"github.com/jaconi-io/litmus/environment"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	clients "github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/result"
	"github.com/litmuschaos/litmus-go/pkg/status"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
)

// Common Kubernetes event types.
const (
	eventTypeNormal  = "Normal"
	eventTypeWarning = "Warning"
)

type Experiment struct {
	Clients       clients.ClientSets
	ChaosDetails  *types.ChaosDetails
	EventDetails  *types.EventDetails
	ResultDetails *types.ResultDetails
}

func NewExperiment(experimentName string, clients clients.ClientSets, customDetails interface{}) (*Experiment, error) {
	// Populate detail structs with values from environment variables.
	chaosDetails := &types.ChaosDetails{}
	eventDetails := &types.EventDetails{}
	resultDetails := &types.ResultDetails{}
	err := environment.Populate(experimentName, chaosDetails, customDetails, resultDetails)
	if err != nil {
		return nil, err
	}

	return &Experiment{
		Clients:       clients,
		ChaosDetails:  chaosDetails,
		EventDetails:  eventDetails,
		ResultDetails: resultDetails,
	}, nil
}

func (e *Experiment) Run(f func(context.Context) error) error {

	// Initialize probes when running in the context of an engine.
	if err := e.runIfInEngineContext("initialize probes", func(context.Context) error {
		return probe.InitializeProbesInChaosResultDetails(e.ChaosDetails, e.Clients, e.ResultDetails)
	}); err != nil {
		return err
	}

	// Change the chaos result state to SOT (Start of Test).
	if err := e.run("change the chaos result state to SOT", func(context.Context) error {
		return result.ChaosResult(e.ChaosDetails, e.Clients, e.ResultDetails, "SOT")
	}); err != nil {
		return err
	}

	// Set the chaos result UID.
	if err := e.run("set the chaos result UID", func(context.Context) error {
		return result.SetResultUID(e.ResultDetails, e.Clients, e.ChaosDetails)
	}); err != nil {
		return err
	}

	// Generating the event in chaosresult to marked the verdict as awaited.
	e.updateResult(types.AwaitedVerdict, fmt.Sprintf("experiment: %s, Result: Awaited", e.ChaosDetails.ExperimentName), eventTypeNormal)

	// Display the application information.
	if e.ChaosDetails.AppDetail.Kind != "" || e.ChaosDetails.AppDetail.Label != "" || e.ChaosDetails.AppDetail.Namespace != "" {
		log.InfoWithValues("application information", map[string]interface{}{
			"experiment": e.ChaosDetails.ExperimentName,
			"kind":       e.ChaosDetails.AppDetail.Kind,
			"label":      e.ChaosDetails.AppDetail.Label,
			"namespace":  e.ChaosDetails.AppDetail.Namespace,
		})
	}

	// Calling AbortWatcher go routine. It will continuously watch for the abort signal and generate the required
	// events and result.
	go common.AbortWatcher(e.ChaosDetails.ExperimentName, e.Clients, e.ResultDetails, e.ChaosDetails, e.EventDetails)

	// Run pre-chaos application status check.
	if err := e.run("pre-chaos application status check", func(context.Context) error {
		return status.AUTStatusCheck(e.ChaosDetails.AppDetail.Namespace, e.ChaosDetails.AppDetail.Label, "", e.ChaosDetails.Timeout, e.ChaosDetails.Delay, e.Clients, e.ChaosDetails)
	}); err != nil {
		return err
	}

	if err := e.runIfInEngineContext("pre-chaos probes", func(context.Context) error {
		// Mark application under test (AUT) as running, as we already checked the status.
		msg := "AUT: Running"

		// Run the probes in the pre-chaos check.
		if len(e.ResultDetails.ProbeDetails) != 0 {
			if err := probe.RunProbes(e.ChaosDetails, e.Clients, e.ResultDetails, "PreChaos", e.EventDetails); err != nil {
				log.Errorf("Probe Failed, err: %v", err)
				failStep := "Failed while running probes"
				e.updateEngine(types.PreChaosCheck, "AUT: Running, Probes: Unsuccessful", eventTypeWarning)
				result.RecordAfterFailure(e.ChaosDetails, e.ResultDetails, failStep, e.Clients, e.EventDetails)
				return err
			}

			msg = "AUT: Running, Probes: Successful"
		}

		// Generating the events for the pre-chaos probes.
		e.updateEngine(types.PreChaosCheck, msg, eventTypeNormal)
		return nil
	}); err != nil {
		return err
	}

	// Execute the actual chaos.
	if err := e.run("chaos", f); err != nil {
		return err
	}

	// Run post-chaos application status check.
	if err := e.run("post-chaos application status check", func(context.Context) error {
		return status.AUTStatusCheck(e.ChaosDetails.AppDetail.Namespace, e.ChaosDetails.AppDetail.Label, "", e.ChaosDetails.Timeout, e.ChaosDetails.Delay, e.Clients, e.ChaosDetails)
	}); err != nil {
		return err
	}

	if err := e.runIfInEngineContext("post-chaos probes", func(context.Context) error {
		// Mark application under test (AUT) as running, as we already checked the status.
		msg := "AUT: Running"

		// Run the probes in the pre-chaos check.
		if len(e.ResultDetails.ProbeDetails) != 0 {
			if err := probe.RunProbes(e.ChaosDetails, e.Clients, e.ResultDetails, "PostChaos", e.EventDetails); err != nil {
				e.updateEngine(types.PreChaosCheck, "AUT: Running, Probes: Unsuccessful", eventTypeWarning)
				return err
			}

			msg = "AUT: Running, Probes: Successful"
		}

		// Generating the events for the pre-chaos probes.
		e.updateEngine(types.PreChaosCheck, msg, eventTypeNormal)
		return nil
	}); err != nil {
		return err
	}

	if e.ChaosDetails.EngineName != "" {
		// marking AUT as running, as we already checked the status of application under test
		msg := "AUT: Running"

		// run the probes in the post-chaos check
		if len(e.ResultDetails.ProbeDetails) != 0 {
			if err := probe.RunProbes(e.ChaosDetails, e.Clients, e.ResultDetails, "PostChaos", e.EventDetails); err != nil {
				e.updateEngine(types.PostChaosCheck, "AUT: Running, Probes: Unsuccessful", eventTypeWarning)
				return err
			}
			msg = "AUT: Running, Probes: Successful"
		}

		// generating post chaos event
		e.updateEngine(types.PostChaosCheck, msg, eventTypeNormal)
	}

	// Change the chaos result state to EOT (End of Test).
	if err := e.run("change the chaos result state to EOT", func(context.Context) error {
		return result.ChaosResult(e.ChaosDetails, e.Clients, e.ResultDetails, "EOT")
	}); err != nil {
		return err
	}

	// Add the verdict to the result.
	if e.ResultDetails.Verdict != v1alpha1.ResultVerdictPassed {
		e.updateResult(types.FailVerdict, fmt.Sprintf("experiment: %s, Result: %s", e.ChaosDetails.ExperimentName, e.ResultDetails.Verdict), eventTypeWarning)
	} else {
		e.updateResult(types.PassVerdict, fmt.Sprintf("experiment: %s, Result: %s", e.ChaosDetails.ExperimentName, e.ResultDetails.Verdict), eventTypeNormal)
	}

	// Add the verdict to the engine (when running in engine context).
	if e.ChaosDetails.EngineName != "" {
		var verdict string
		switch v := e.ResultDetails.Verdict; v {
		case v1alpha1.ResultVerdictPassed:
			verdict = "passed"
		case v1alpha1.ResultVerdictFailed:
			verdict = "failed"
		case v1alpha1.ResultVerdictStopped:
			verdict = "has been stopped"
		default:
			verdict = fmt.Sprintf("has unknown verdict %q", v)
		}

		e.updateEngine(types.Summary, fmt.Sprintf("experiment %q %s", e.ChaosDetails.ExperimentName, verdict), eventTypeNormal)
	}

	return nil
}

func (e Experiment) updateResult(reason, msg, eventType string) {
	types.SetResultEventAttributes(e.EventDetails, reason, msg, eventType, e.ResultDetails)
	events.GenerateEvents(e.EventDetails, e.Clients, e.ChaosDetails, "ChaosResult")
}

func (e Experiment) updateEngine(reason, msg, eventType string) {
	types.SetEngineEventAttributes(e.EventDetails, reason, msg, eventType, e.ChaosDetails)
	events.GenerateEvents(e.EventDetails, e.Clients, e.ChaosDetails, "ChaosEngine")
}

// Run function with proper error handling.
func (e Experiment) run(step string, f func(context.Context) error) error {
	log.InfoWithValues(fmt.Sprintf("[Step]: %s", step), map[string]interface{}{
		"experiment": e.ChaosDetails.ExperimentName,
	})

	err := f(context.Background())
	if err != nil {
		msg := fmt.Sprintf("failed to %s: %v", step, err)
		log.ErrorWithValues(msg, map[string]interface{}{
			"experiment": e.ChaosDetails.ExperimentName,
		})
		result.RecordAfterFailure(e.ChaosDetails, e.ResultDetails, step, e.Clients, e.EventDetails)
		return errors.New(msg)
	}

	return nil
}

// Run function if in engine context.
func (e Experiment) runIfInEngineContext(step string, f func(context.Context) error) error {
	if e.ChaosDetails.EngineName == "" {
		log.InfoWithValues(fmt.Sprintf("[Skip]: %s (not running in engine context)", step), map[string]interface{}{
			"experiment": e.ChaosDetails.ExperimentName,
		})
		return nil
	}

	return e.run(step, f)
}
