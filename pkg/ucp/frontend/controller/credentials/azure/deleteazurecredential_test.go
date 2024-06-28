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
package azure

import (
	"context"
	"errors"
	"net/http"
	"testing"

	v1 "github.com/radius-project/radius/pkg/armrpc/api/v1"
	armrpc_controller "github.com/radius-project/radius/pkg/armrpc/frontend/controller"
	armrpc_rest "github.com/radius-project/radius/pkg/armrpc/rest"
	"github.com/radius-project/radius/pkg/armrpc/rpctest"
	"github.com/radius-project/radius/pkg/ucp/datamodel"
	"github.com/radius-project/radius/pkg/ucp/secret"
	"github.com/radius-project/radius/pkg/ucp/store"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Credential_Delete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient := store.NewMockStorageClient(mockCtrl)
	mockSecretClient := secret.NewMockClient(mockCtrl)

	credentialCtrl, err := NewDeleteAzureCredential(armrpc_controller.Options{
		StorageClient: mockStorageClient,
	}, mockSecretClient)
	require.NoError(t, err)

	tests := []struct {
		name       string
		url        string
		headerfile string
		fn         func(mockStorageClient store.MockStorageClient, mockSecretClient secret.MockClient)
		expected   armrpc_rest.Response
		err        error
	}{
		{
			name:       "test_credential_deletion",
			url:        "/planes/azure/azurecloud/providers/System.Azure/credentials/default?api-version=2023-10-01-preview",
			headerfile: testHeaderFile,
			fn:         setupCredentialDeleteSuccessMocks,
			expected:   armrpc_rest.NewOKResponse(nil),
			err:        nil,
		},
		{
			name:       "test_non_existent_credential_deletion",
			url:        "/planes/azure/azurecloud/providers/System.Azure/credentials/default?api-version=2023-10-01-preview",
			headerfile: testHeaderFile,
			fn:         setupNonExistentCredentialDeleteMocks,
			expected:   armrpc_rest.NewNoContentResponse(),
			err:        nil,
		},
		{
			name:       "test_failed_credential_existence_check",
			url:        "/planes/azure/azurecloud/providers/System.Azure/credentials/default?api-version=2023-10-01-preview",
			headerfile: testHeaderFile,
			fn:         setupCredentialExistenceCheckFailureMocks,
			expected:   nil,
			err:        errors.New("test_failure"),
		},
		{
			name:       "test_non_existent_secret_deletion",
			url:        "/planes/azure/azurecloud/providers/System.Azure/credentials/default?api-version=2023-10-01-preview",
			headerfile: testHeaderFile,
			fn:         setupNonExistentSecretDeleteMocks,
			expected:   armrpc_rest.NewNoContentResponse(),
			err:        nil,
		},
		{
			name:       "test_secret_deletion_failure",
			url:        "/planes/azure/azurecloud/providers/System.Azure/credentials/default?api-version=2023-10-01-preview",
			headerfile: testHeaderFile,
			fn:         setupSecretDeleteFailureMocks,
			expected:   nil,
			err:        errors.New("Failed secret deletion"),
		},
		{
			name:       "test_non_existing_credential_deletion_from_storage",
			url:        "/planes/azure/azurecloud/providers/System.Azure/credentials/default?api-version=2023-10-01-preview",
			headerfile: testHeaderFile,
			fn:         setupNonExistingCredentialDeleteFromStorageMocks,
			expected:   armrpc_rest.NewNoContentResponse(),
			err:        nil,
		},
		{
			name:       "test_failed_credential_deletion_from_storage",
			url:        "/planes/azure/azurecloud/providers/System.Azure/credentials/default?api-version=2023-10-01-preview",
			headerfile: testHeaderFile,
			fn:         setupFailedCredentialDeleteFromStorageMocks,
			expected:   nil,
			err:        errors.New("Failed Storage Deletion"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(*mockStorageClient, *mockSecretClient)
			request, err := rpctest.NewHTTPRequestFromJSON(context.Background(), http.MethodDelete, tt.headerfile, nil)
			require.NoError(t, err)

			ctx := rpctest.NewARMRequestContext(request)

			response, err := credentialCtrl.Run(ctx, nil, request)
			if tt.err != nil {
				require.Equal(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, response)
			}
		})
	}
}

func setupCredentialMocks(mockStorageClient store.MockStorageClient) {
	datamodelCredential := datamodel.AzureCredential{
		BaseResource: v1.BaseResource{},
		Properties: &datamodel.AzureCredentialResourceProperties{
			Kind: datamodel.AWSAccessKeyCredentialKind,
		},
	}

	mockStorageClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, id string, options ...store.GetOptions) (*store.Object, error) {
			return &store.Object{
				Metadata: store.Metadata{
					ID: datamodelCredential.TrackedResource.ID,
				},
				Data: &datamodelCredential,
			}, nil
		}).Times(1)
}

func setupCredentialDeleteSuccessMocks(mockStorageClient store.MockStorageClient, mockSecretClient secret.MockClient) {
	setupCredentialMocks(mockStorageClient)
	mockSecretClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockStorageClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
}

func setupNonExistentCredentialDeleteMocks(mockStorageClient store.MockStorageClient, mockSecretClient secret.MockClient) {
	mockStorageClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &store.ErrNotFound{}).Times(1)
}

func setupCredentialExistenceCheckFailureMocks(mockStorageClient store.MockStorageClient, mockSecretClient secret.MockClient) {
	mockStorageClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("test_failure")).Times(1)
}

func setupNonExistentSecretDeleteMocks(mockStorageClient store.MockStorageClient, mockSecretClient secret.MockClient) {
	setupCredentialMocks(mockStorageClient)

	mockSecretClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(&secret.ErrNotFound{}).Times(1)
}

func setupSecretDeleteFailureMocks(mockStorageClient store.MockStorageClient, mockSecretClient secret.MockClient) {
	setupCredentialMocks(mockStorageClient)

	mockSecretClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("Failed secret deletion")).Times(1)
}

func setupNonExistingCredentialDeleteFromStorageMocks(mockStorageClient store.MockStorageClient, mockSecretClient secret.MockClient) {
	setupCredentialMocks(mockStorageClient)

	mockSecretClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockStorageClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(&store.ErrNotFound{}).Times(1)
}

func setupFailedCredentialDeleteFromStorageMocks(mockStorageClient store.MockStorageClient, mockSecretClient secret.MockClient) {
	setupCredentialMocks(mockStorageClient)

	mockSecretClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockStorageClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("Failed Storage Deletion")).Times(1)
}
