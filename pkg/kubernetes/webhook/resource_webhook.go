// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/project-radius/radius/pkg/cli/armtemplate"
	radiusv1alpha3 "github.com/project-radius/radius/pkg/kubernetes/api/radius/v1alpha3"
	"github.com/project-radius/radius/pkg/kubernetes/converters"
	"github.com/project-radius/radius/pkg/kubernetes/webhook/external"
	"github.com/project-radius/radius/pkg/radrp/schema"
	"github.com/project-radius/radius/pkg/renderers"
)

type ResourceWebhook struct {
	external.ValidatingWebhook
}

func (w *ResourceWebhook) SetupWebhookWithManager(mgr manager.Manager) error {
	return external.NewGenericWebhookManagedBy(mgr).
		WithValidatePath("/validate-radius-dev-v1alpha3-resource").
		Complete(w)
}

func (w *ResourceWebhook) ValidateCreate(ctx context.Context, request admission.Request, object *unstructured.Unstructured) admission.Response {
	_ = log.FromContext(ctx)

	resource := &radiusv1alpha3.Resource{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, resource)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	renderResource := &renderers.RendererResource{}
	err = converters.ConvertToRenderResource(resource, renderResource)

	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	validator, err := w.findValidator(renderResource.ResourceType)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	template := resource.Spec.Template

	// Get arm template from template part
	if template == nil {
		return admission.Errored(http.StatusBadRequest, errors.New("template is nil"))
	}

	armResource := &armtemplate.Resource{}
	err = json.Unmarshal(template.Raw, armResource)

	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if armResource.Body != nil {
		armJson, err := json.Marshal(armResource.Body)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		validationErrs := validator.ValidateJSON(armJson)
		if len(validationErrs) > 0 {
			return admission.Errored(http.StatusBadRequest, &schema.AggregateValidationError{Details: validationErrs})
		}
	}

	return admission.Allowed("")
}

func (w *ResourceWebhook) findValidator(resourceType string) (schema.Validator, error) {
	validator, ok := schema.GetValidator(resourceType)
	if !ok {
		return nil, fmt.Errorf("no validator found for resource type %s", resourceType)
	}
	return validator, nil
}
