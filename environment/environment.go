package environment

import (
	"fmt"
	"reflect"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/litmuschaos/litmus-go/pkg/types"
)

// ExperimentDetails can be used as an embedded struct by the specific experiment details.
type ExperimentDetails struct {
	AppAnnotationCheck bool          `default:"false" envconfig:"ANNOTATION_CHECK"`
	AppAnnotationKey   string        `default:"litmuschaos.io/chaos" envconfig:"ANNOTATION_KEY"`
	AppAnnotationValue string        `default:"true" envconfig:"ANNOTATION_VALUE"`
	AppKind            string        `split_words:"true"`
	AppLabel           string        `split_words:"true"`
	AppNamespace       string        `split_words:"true"`
	ChaosDuration      time.Duration `split_words:"true"`
	ChaosNamespace     string        `default:"litmus" split_words:"true"`
	EngineName         string        `envconfig:"CHAOS_ENGINE"`
	ExperimentName     string        `split_words:"true"`
	JobCleanupPolicy   string        `default:"retain" split_words:"true"`
}

// Populate chaos, experiment and result details using environment variables.
func Populate(experimentName string, chaos *types.ChaosDetails, experiment interface{}, result *types.ResultDetails) error {
	err := envconfig.Process("", experiment)
	if err != nil {
		return err
	}

	elem := reflect.ValueOf(experiment).Elem()
	field := elem.FieldByName("ExperimentDetails")

	if !field.IsValid() {
		return fmt.Errorf("ExperimentDetails is mising; make sure %T is embedded into %T", &ExperimentDetails{}, experiment)
	}

	converted, ok := field.Interface().(ExperimentDetails)
	if !ok {
		return fmt.Errorf("could not convert %s to %T; make sure ExperimentDetails has the correct type", field.Type(), &ExperimentDetails{})
	}

	// Fallback to actual experiment name, if none has been set.
	if converted.ExperimentName == "" {
		converted.ExperimentName = experimentName
	}

	appDetails := types.AppDetails{}
	appDetails.AnnotationCheck = converted.AppAnnotationCheck
	appDetails.AnnotationKey = converted.AppAnnotationKey
	appDetails.AnnotationValue = converted.AppAnnotationValue
	appDetails.Kind = converted.AppKind
	appDetails.Label = converted.AppLabel
	appDetails.Namespace = converted.AppNamespace

	chaos.AppDetail = appDetails
	chaos.ChaosDuration = int(converted.ChaosDuration)
	chaos.ChaosNamespace = converted.ChaosNamespace
	chaos.EngineName = converted.EngineName
	chaos.ExperimentName = converted.ExperimentName
	chaos.JobCleanupPolicy = converted.JobCleanupPolicy

	types.SetResultAttributes(result, *chaos)
	return nil
}
