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

package common

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/radius-project/radius/pkg/cli/manifest"
	"github.com/radius-project/radius/pkg/ucp/api/v20231001preview"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_GetResourceTypeDetails(t *testing.T) {
	t.Run("Get Resource Details Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		clientFactory, err := manifest.NewTestClientFactory(manifest.WithResourceProviderServerNoError)
		require.NoError(t, err)

		res, err := GetResourceTypeDetails(context.Background(), "MyCompany.Resources", "testResources", clientFactory, false)
		require.NoError(t, err)
		require.Equal(t, "MyCompany.Resources/testResources", res.Name)

	})

	t.Run("Get Resource Details Failure - Resource Provider Not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		clientFactory, err := manifest.NewTestClientFactory(manifest.WithResourceProviderServerNotFoundError)
		require.NoError(t, err)

		_, err = GetResourceTypeDetails(context.Background(), "MyCompany.Resources", "testResources", clientFactory, false)
		require.Error(t, err)
		require.Equal(t, "The resource type \"MyCompany.Resources/testResources\" does not exist.", err.Error())
	})

	t.Run("Get Resource Details Failure - Resource Type Not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		clientFactory, err := manifest.NewTestClientFactory(manifest.WithResourceProviderServerNoError)
		require.NoError(t, err)

		_, err = GetResourceTypeDetails(context.Background(), "MyCompany.Resources", "missingResources", clientFactory, false)
		require.Error(t, err)
		require.Equal(t, "The resource type \"MyCompany.Resources/missingResources\" does not exist.", err.Error())
	})

	t.Run("Get Resource Details Failures Other Than Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		clientFactory, err := manifest.NewTestClientFactory(manifest.WithResourceProviderServerInternalError)
		require.NoError(t, err)

		_, err = GetResourceTypeDetails(context.Background(), "MyCompany.Resources", "testResources", clientFactory, false)
		require.Error(t, err)
	})
}

func Test_ResourceTypesForProvider_Icon(t *testing.T) {
	summary := &v20231001preview.ResourceProviderSummary{
		Name: to.Ptr("MyCompany.Resources"),
		ResourceTypes: map[string]*v20231001preview.ResourceProviderSummaryResourceType{
			"withIcon": {
				Icon:     to.Ptr(`<svg/>`),
				IconHash: to.Ptr("cafebabe"),
			},
			"withoutIcon": {},
		},
	}

	result := ResourceTypesForProvider(summary)

	byName := map[string]ResourceType{}
	for _, rt := range result {
		byName[rt.Name] = rt
	}

	require.Equal(t, `<svg/>`, byName["MyCompany.Resources/withIcon"].Icon)
	require.Equal(t, "cafebabe", byName["MyCompany.Resources/withIcon"].IconHash)
	require.Empty(t, byName["MyCompany.Resources/withoutIcon"].Icon)
	require.Empty(t, byName["MyCompany.Resources/withoutIcon"].IconHash)
}
