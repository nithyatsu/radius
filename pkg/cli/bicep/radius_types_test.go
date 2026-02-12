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
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_IsRadiusResourceType(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		expected     bool
	}{
		{
			name:         "core container",
			resourceType: "Applications.Core/containers",
			expected:     true,
		},
		{
			name:         "core gateway",
			resourceType: "Applications.Core/gateways",
			expected:     true,
		},
		{
			name:         "redis cache",
			resourceType: "Applications.Datastores/redisCaches",
			expected:     true,
		},
		{
			name:         "mongo database",
			resourceType: "Applications.Datastores/mongoDatabases",
			expected:     true,
		},
		{
			name:         "dapr state store",
			resourceType: "Applications.Dapr/stateStores",
			expected:     true,
		},
		{
			name:         "messaging rabbit",
			resourceType: "Applications.Messaging/rabbitMQQueues",
			expected:     true,
		},
		{
			name:         "generic applications prefix",
			resourceType: "Applications.CustomProvider/myResource",
			expected:     true,
		},
		{
			name:         "azure resource",
			resourceType: "Microsoft.Storage/storageAccounts",
			expected:     false,
		},
		{
			name:         "aws resource",
			resourceType: "AWS.S3/Bucket",
			expected:     false,
		},
		{
			name:         "empty string",
			resourceType: "",
			expected:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsRadiusResourceType(tc.resourceType)
			require.Equal(t, tc.expected, result)
		})
	}
}

func Test_FilterRadiusResources(t *testing.T) {
	resources := []ExtractedResource{
		{Name: "frontend", Type: "Applications.Core/containers"},
		{Name: "storage", Type: "Microsoft.Storage/storageAccounts"},
		{Name: "cache", Type: "Applications.Datastores/redisCaches"},
		{Name: "bucket", Type: "AWS.S3/Bucket"},
	}

	filtered := FilterRadiusResources(resources)
	require.Len(t, filtered, 2)
	require.Equal(t, "frontend", filtered[0].Name)
	require.Equal(t, "cache", filtered[1].Name)
}

func Test_FilterRadiusResources_AllRadius(t *testing.T) {
	resources := []ExtractedResource{
		{Name: "frontend", Type: "Applications.Core/containers"},
		{Name: "cache", Type: "Applications.Datastores/redisCaches"},
	}

	filtered := FilterRadiusResources(resources)
	require.Len(t, filtered, 2)
}

func Test_FilterRadiusResources_NoneRadius(t *testing.T) {
	resources := []ExtractedResource{
		{Name: "storage", Type: "Microsoft.Storage/storageAccounts"},
	}

	filtered := FilterRadiusResources(resources)
	require.Empty(t, filtered)
}

func Test_ResourceCategory(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		expected     string
	}{
		{"container", "Applications.Core/containers", "container"},
		{"gateway", "Applications.Core/gateways", "gateway"},
		{"redis", "Applications.Datastores/redisCaches", "datastore"},
		{"mongo", "Applications.Datastores/mongoDatabases", "datastore"},
		{"secret store", "Applications.Core/secretStores", "secret"},
		{"extender", "Applications.Core/extenders", "extender"},
		{"unknown", "Applications.Core/environments", "resource"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ResourceCategory(tc.resourceType)
			require.Equal(t, tc.expected, result)
		})
	}
}
