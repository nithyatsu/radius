// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package converters

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/project-radius/radius/pkg/cli/armtemplate"
	"github.com/project-radius/radius/pkg/kubernetes"
	radiusv1alpha3 "github.com/project-radius/radius/pkg/kubernetes/api/radius/v1alpha3"
	"github.com/project-radius/radius/pkg/radrp/outputresource"
	"github.com/project-radius/radius/pkg/renderers"
	"k8s.io/apimachinery/pkg/runtime"
)

// Since the resource will be processed by an ARM template we need to convert it to an ARM-like representation.
func ConvertToARMResource(original *radiusv1alpha3.Resource, body map[string]interface{}) error {
	properties, ok := body["properties"].(map[string]interface{})
	if !ok {
		properties = map[string]interface{}{}
		body["properties"] = properties
	}

	// Using the user-provided definition as a 'base' merge in the computed properties
	if original.Status.ComputedValues != nil {
		computedValues := map[string]renderers.ComputedValueReference{}

		err := json.Unmarshal(original.Status.ComputedValues.Raw, &computedValues)
		if err != nil {
			return err
		}

		for key, value := range computedValues {
			properties[key] = value.Value
		}
	}

	return nil
}

func ConvertToRenderResource(original *radiusv1alpha3.Resource, result *renderers.RendererResource) error {
	result.ResourceName = original.Spec.Resource
	result.ResourceType = original.Annotations[kubernetes.LabelRadiusResourceType]
	result.ApplicationName = original.Spec.Application

	template := original.Spec.Template

	// Get arm template from template part
	if template == nil {
		return errors.New("must have template as part of CRD")
	}

	armResource := &armtemplate.Resource{}
	err := json.Unmarshal(template.Raw, armResource)

	if err != nil {
		return err
	}

	if armResource.Body != nil {
		properties, ok := armResource.Body["properties"]
		if ok {
			data, err := json.Marshal(properties)
			if err != nil {
				return err
			}

			err = json.Unmarshal(data, &result.Definition)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetComputedValues(status radiusv1alpha3.ResourceStatus) (map[string]renderers.ComputedValueReference, error) {
	computedValues := map[string]renderers.ComputedValueReference{}
	if status.ComputedValues != nil {
		err := json.Unmarshal(status.ComputedValues.Raw, &computedValues)
		if err != nil {
			return nil, err
		}
	}

	return computedValues, nil
}

func SetComputedValues(status *radiusv1alpha3.ResourceStatus, values map[string]renderers.ComputedValueReference) error {
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	// TODO convert from computed value to to interface{}
	status.ComputedValues = &runtime.RawExtension{Raw: data}
	return nil
}

func GetSecretValues(status radiusv1alpha3.ResourceStatus) (map[string]renderers.SecretValueReference, error) {
	secretValues := map[string]renderers.SecretValueReference{}
	if status.SecretValues != nil {
		err := json.Unmarshal(status.SecretValues.Raw, &secretValues)
		if err != nil {
			return nil, err
		}
	}

	return secretValues, nil
}

func SetSecretValues(status *radiusv1alpha3.ResourceStatus, values map[string]renderers.SecretValueReference) error {
	raw, err := json.Marshal(values)
	if err != nil {
		return err
	}

	status.SecretValues = &runtime.RawExtension{Raw: raw}
	return nil
}

func SetStatusForOutputResources(status *radiusv1alpha3.ResourceStatus, values []outputresource.OutputResource) error {
	for _, or := range values {
		if _, ok := status.Resources[or.LocalID]; !ok {
			return fmt.Errorf("Unable to find output resource: %v. Cannot set status", or.LocalID)
		}
		status.Resources[or.LocalID].Status = radiusv1alpha3.OutputResourceStatus{
			HealthState:                   or.Status.HealthState,
			HealthStateErrorDetails:       or.Status.HealthErrorDetails,
			ProvisioningState:             or.Status.ProvisioningState,
			ProvisioningStateErrorDetails: or.Status.ProvisioningErrorDetails,
		}
	}
	return nil
}
