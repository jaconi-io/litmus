package environment_test

import (
	"os"
	"testing"

	. "github.com/jaconi-io/litmus/environment"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/stretchr/testify/assert"

	k8sTypes "k8s.io/apimachinery/pkg/types"
)

type testDetails struct {
	ExperimentDetails
	Additional string `required:"true"`
}

// Make sure an error is returned, if ExperimentDetails is not a struct pointer.
func TestPopulateNoPointer(t *testing.T) {
	err := Populate("test", &types.ChaosDetails{}, struct{}{}, &types.ResultDetails{})
	assert.EqualError(t, err, "specification must be a struct pointer")
}

// Make sure an error is returned, if ExperimentDetails is not embedded.
func TestPopulateEmbedMissing(t *testing.T) {
	err := Populate("test", &types.ChaosDetails{}, &struct{}{}, &types.ResultDetails{})
	assert.EqualError(t, err, "ExperimentDetails is mising; make sure *environment.ExperimentDetails is embedded into *struct {}")
}

// Make sure an error is returned, if ExperimentDetails exists but is the wrong type.
func TestPopulateEmbedWrongType(t *testing.T) {
	err := Populate("test", &types.ChaosDetails{}, &struct {
		ExperimentDetails string
	}{}, &types.ResultDetails{})
	assert.EqualError(t, err, "could not convert string to *environment.ExperimentDetails; make sure ExperimentDetails has the correct type")
}

// Make sure an error is returned, if validation of additional fields fails.
func TestPopulateMissingAdditional(t *testing.T) {
	err := Populate("test", &types.ChaosDetails{}, &testDetails{}, &types.ResultDetails{})
	assert.EqualError(t, err, "required key ADDITIONAL missing value")
}

func TestPopulateDefaults(t *testing.T) {
	defer tmpEnv(map[string]string{
		"ADDITIONAL": "",
	})()

	chaos := &types.ChaosDetails{}
	experiment := &testDetails{}
	result := &types.ResultDetails{}

	err := Populate("test", chaos, experiment, result)
	assert.NoError(t, err)

	assert.Equal(t, "", experiment.Additional)

	assert.Equal(t, "", experiment.AppKind)
	assert.Equal(t, "", experiment.AppLabel)
	assert.Equal(t, "", experiment.AppNamespace)
	assert.Equal(t, "litmus", experiment.ChaosNamespace)
	assert.Equal(t, "", experiment.ExperimentName)

	assert.Equal(t, false, chaos.AppDetail.AnnotationCheck)
	assert.Equal(t, "litmuschaos.io/chaos", chaos.AppDetail.AnnotationKey)
	assert.Equal(t, "true", chaos.AppDetail.AnnotationValue)
	assert.Equal(t, "", chaos.AppDetail.Kind)
	assert.Equal(t, "", chaos.AppDetail.Label)
	assert.Equal(t, "", chaos.AppDetail.Namespace)

	assert.Equal(t, 0, chaos.ChaosDuration)
	assert.Equal(t, "litmus", chaos.ChaosNamespace)
	assert.Equal(t, "", chaos.ChaosPodName)
	assert.Equal(t, k8sTypes.UID(""), chaos.ChaosUID)
	assert.Equal(t, 0, chaos.Delay)
	assert.Equal(t, "", chaos.EngineName)
	assert.Equal(t, "test", chaos.ExperimentName)
	assert.Equal(t, "", chaos.InstanceID)
	assert.Equal(t, "retain", chaos.JobCleanupPolicy)
	assert.Equal(t, []string(nil), chaos.ParentsResources)
	assert.Equal(t, "", chaos.ProbeImagePullPolicy)
	assert.Equal(t, false, chaos.Randomness)
	assert.Equal(t, []v1alpha1.TargetDetails(nil), chaos.Targets)
	assert.Equal(t, 0, chaos.Timeout)
}

func TestPopulate(t *testing.T) {
	defer tmpEnv(map[string]string{
		"ADDITIONAL":         "foo",
		"ANNOTATION_CHECK":   "true",
		"ANNOTATION_KEY":     "foo",
		"ANNOTATION_VALUE":   "bar",
		"APP_KIND":           "deployment",
		"APP_LABEL":          "foo=bar",
		"APP_NAMESPACE":      "default",
		"CHAOS_DURATION":     "30m",
		"CHAOS_NAMESPACE":    "chaos",
		"CHAOS_ENGINE":       "foo",
		"EXPERIMENT_NAME":    "foo",
		"JOB_CLEANUP_POLICY": "delete",
	})()

	chaos := &types.ChaosDetails{}
	experiment := &testDetails{}
	result := &types.ResultDetails{}

	err := Populate("test", chaos, experiment, result)
	assert.NoError(t, err)

	assert.Equal(t, "foo", experiment.Additional)

	assert.Equal(t, "deployment", experiment.AppKind)
	assert.Equal(t, "foo=bar", experiment.AppLabel)
	assert.Equal(t, "default", experiment.AppNamespace)
	assert.Equal(t, "chaos", experiment.ChaosNamespace)
	assert.Equal(t, "foo", experiment.EngineName)
	assert.Equal(t, "foo", experiment.ExperimentName)

	assert.Equal(t, true, chaos.AppDetail.AnnotationCheck)
	assert.Equal(t, "foo", chaos.AppDetail.AnnotationKey)
	assert.Equal(t, "bar", chaos.AppDetail.AnnotationValue)
	assert.Equal(t, "deployment", chaos.AppDetail.Kind)
	assert.Equal(t, "foo=bar", chaos.AppDetail.Label)
	assert.Equal(t, "default", chaos.AppDetail.Namespace)

	assert.Equal(t, 1800000000000, chaos.ChaosDuration)
	assert.Equal(t, "chaos", chaos.ChaosNamespace)
	// assert.NotEqual(t, "", chaos.ChaosPodName)
	// assert.NotEqual(t, k8sTypes.UID(""), chaos.ChaosUID)
	// assert.NotEqual(t, 0, chaos.Delay)
	assert.Equal(t, "foo", chaos.EngineName)
	// assert.NotEqual(t, "foo", chaos.ExperimentName)
	// assert.NotEqual(t, "", chaos.InstanceID)
	assert.Equal(t, "delete", chaos.JobCleanupPolicy)
	// assert.NotEqual(t, []string(nil), chaos.ParentsResources)
	// assert.NotEqual(t, "", chaos.ProbeImagePullPolicy)
	// assert.NotEqual(t, false, chaos.Randomness)
	// assert.NotEqual(t, []v1alpha1.TargetDetails(nil), chaos.Targets)
	// assert.NotEqual(t, 0, chaos.Timeout)
}

func tmpEnv(env map[string]string) func() {
	old := map[string]string{}
	for key, value := range env {
		if oldValue, ok := os.LookupEnv(key); ok {
			old[key] = oldValue
		}

		err := os.Setenv(key, value)
		if err != nil {
			panic(err)
		}
	}

	return func() {
		for key, value := range old {
			err := os.Setenv(key, value)
			if err != nil {
				panic(err)
			}
		}
	}
}
