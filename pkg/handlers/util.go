// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package handlers

import (
	"github.com/project-radius/radius/pkg/radrp/db"
	"github.com/project-radius/radius/pkg/radrp/outputresource"
)

// mergeProperties combines properties from a resource definition and a potentially existing resource.
// This is useful for cases where deploying a resource results in storage of generated values like names.
// By merging properties, the caller gets to see those values and reuse them.
func mergeProperties(resource outputresource.OutputResource, dbResource *db.OutputResource) map[string]string {
	properties, ok := resource.Resource.(map[string]string)
	if !ok {
		properties = map[string]string{}
	}

	if dbResource != nil {
		for k, v := range dbResource.PersistedProperties {
			_, ok := properties[k]
			if !ok {
				properties[k] = v
			}
		}
	}

	return properties
}
