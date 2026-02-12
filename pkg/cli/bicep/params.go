/*
Copyright 2023 The Radius Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bicep

import (
	"fmt"
	"sort"
	"strings"
)

// ValidateRequiredParameters checks the compiled ARM JSON template for parameters
// that have no default value. If any required parameters are found and no parameter
// file was provided, it returns a descriptive error listing them.
//
// This implements the US1 acceptance scenario: "Given a Bicep file with required
// parameters (no defaults) and no --parameters flag, When I run rad app graph
// app.bicep, Then I receive a clear error listing the missing required parameters."
func ValidateRequiredParameters(template map[string]any, providedParams map[string]any) error {
	params, err := ExtractParameters(template)
	if err != nil {
		return fmt.Errorf("failed to extract template parameters: %w", err)
	}

	var missing []string
	for name, paramRaw := range params {
		// Skip parameters that have a default value
		if _, hasDefault := DefaultValue(paramRaw); hasDefault {
			continue
		}

		// Skip parameters already provided
		if providedParams != nil {
			if _, provided := providedParams[name]; provided {
				continue
			}
		}

		missing = append(missing, name)
	}

	if len(missing) == 0 {
		return nil
	}

	// Sort for deterministic error output
	sort.Strings(missing)

	return fmt.Errorf(
		"the Bicep file requires the following parameters that were not provided: %s\n"+
			"Use --parameters <file.json> to supply parameter values",
		strings.Join(missing, ", "),
	)
}
