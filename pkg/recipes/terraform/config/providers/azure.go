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

package providers

import (
	"context"
	"errors"
	"fmt"

	"github.com/radius-project/radius/pkg/azure/tokencredentials"
	"github.com/radius-project/radius/pkg/components/secret"
	"github.com/radius-project/radius/pkg/components/secret/secretprovider"
	"github.com/radius-project/radius/pkg/corerp/datamodel"
	"github.com/radius-project/radius/pkg/recipes"
	"github.com/radius-project/radius/pkg/sdk"
	"github.com/radius-project/radius/pkg/ucp/credentials"
	ucp_datamodel "github.com/radius-project/radius/pkg/ucp/datamodel"
	"github.com/radius-project/radius/pkg/ucp/resources"
	resources_azure "github.com/radius-project/radius/pkg/ucp/resources/azure"
	"github.com/radius-project/radius/pkg/ucp/ucplog"
)

// Provider's config parameters need to match the values expected by Terraform
// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs
const (
	AzureProviderName = "azurerm"

	azureFeaturesParam          = "features"
	azureSubIDParam             = "subscription_id"
	azureClientIDParam          = "client_id"
	azureClientSecretParam      = "client_secret"
	azureTenantIDParam          = "tenant_id"
	azureUseOIDCParam           = "use_oidc"
	azureUseCLIParam            = "use_cli"
	azureOIDCTokenFilePathParam = "oidc_token_file_path"

	// The Azure AD Workload Identity Mutating Admission Webhook projects a signed service account token to
	// this well known path.
	// https://azure.github.io/azure-workload-identity/docs/installation/mutating-admission-webhook.html
	// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs#argument-reference
	azureOIDCTokenFilePath = "/var/run/secrets/azure/tokens/azure-identity-token"
)

var _ Provider = (*azureProvider)(nil)

type azureProvider struct {
	ucpConn        sdk.Connection
	secretProvider *secretprovider.SecretProvider
}

// NewAzureProvider creates a new AzureProvider instance.
func NewAzureProvider(ucpConn sdk.Connection, secretProvider *secretprovider.SecretProvider) Provider {
	return &azureProvider{ucpConn: ucpConn, secretProvider: secretProvider}
}

// BuildConfig generates the Terraform provider configuration for Azure provider. It checks if the environment
// configuration contains an Azure provider scope and if so, parses the scope to get the subscriptionID and adds
// it to the config map. If the scope is invalid, an error is returned.
// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs
func (p *azureProvider) BuildConfig(ctx context.Context, envConfig *recipes.Configuration) (map[string]any, error) {
	// features block is required for Azure provider even if it is empty
	// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs#argument-reference
	config := map[string]any{
		azureFeaturesParam: map[string]any{},
	}

	subscriptionID, err := p.parseScope(ctx, envConfig)
	if err != nil {
		return nil, err
	}

	credentialsProvider, err := p.getCredentialsProvider()
	if err != nil {
		return nil, err
	}

	credentials, err := fetchAzureCredentials(ctx, credentialsProvider)
	if err != nil {
		return nil, err
	}

	return p.generateProviderConfigMap(config, credentials, subscriptionID), nil
}

// parseScope parses an Azure provider scope and returns the associated subscription id
// Example scope: /subscriptions/test-sub/resourceGroups/test-rg
func (p *azureProvider) parseScope(ctx context.Context, envConfig *recipes.Configuration) (subscriptionID string, err error) {
	logger := ucplog.FromContextOrDiscard(ctx)
	if (envConfig == nil) || (envConfig.Providers == datamodel.Providers{}) || (envConfig.Providers.Azure == datamodel.ProvidersAzure{}) || envConfig.Providers.Azure.Scope == "" {
		logger.Info("Azure provider scope is not configured on the Environment, skipping Azure subscriptionID configuration.")
		return "", nil
	}

	scope := envConfig.Providers.Azure.Scope
	parsedScope, err := resources.Parse(scope)
	if err != nil {
		return "", fmt.Errorf("invalid Azure provider scope %q is configured on the Environment, error parsing: %s", scope, err.Error())
	}

	subscription := parsedScope.FindScope(resources_azure.ScopeSubscriptions)
	if subscription == "" {
		return "", fmt.Errorf("invalid Azure provider scope %q is configured on the Environment, subscription is required in the scope", scope)
	}

	return subscription, nil
}

func (p *azureProvider) getCredentialsProvider() (*credentials.AzureCredentialProvider, error) {
	return credentials.NewAzureCredentialProvider(p.secretProvider, p.ucpConn, &tokencredentials.AnonymousCredential{})
}

// fetchAzureCredentials fetches Azure credentials from UCP. Returns nil if credentials not found error is received or the credentials are empty.
func fetchAzureCredentials(ctx context.Context, azureCredentialsProvider credentials.CredentialProvider[credentials.AzureCredential]) (*credentials.AzureCredential, error) {
	logger := ucplog.FromContextOrDiscard(ctx)
	credentials, err := azureCredentialsProvider.Fetch(ctx, credentials.AzureCloud, "default")
	if err != nil {
		if errors.Is(err, &secret.ErrNotFound{}) {
			logger.Info("Azure credentials are not registered, skipping credentials configuration.")
			return nil, nil
		}

		return nil, err
	}

	switch credentials.Kind {
	case ucp_datamodel.AzureServicePrincipalCredentialKind:
		if credentials.ServicePrincipal == nil ||
			credentials.ServicePrincipal.ClientID == "" ||
			credentials.ServicePrincipal.TenantID == "" ||
			credentials.ServicePrincipal.ClientSecret == "" {
			logger.Info("Azure service principal credentials are not registered, skipping credentials configuration.")
			return nil, nil
		}

		return credentials, nil
	case ucp_datamodel.AzureWorkloadIdentityCredentialKind:
		if credentials.WorkloadIdentity == nil ||
			credentials.WorkloadIdentity.ClientID == "" ||
			credentials.WorkloadIdentity.TenantID == "" {
			logger.Info("Azure workload identity credentials are not registered, skipping credentials configuration.")
			return nil, nil
		}

		return credentials, nil
	default:
		logger.Info("Azure credential is not supported, skipping credentials configuration, kind: %s", credentials.Kind)
		return nil, nil
	}

}

func (p *azureProvider) generateProviderConfigMap(configMap map[string]any, credentials *credentials.AzureCredential, subscriptionID string) map[string]any {
	if subscriptionID != "" {
		configMap[azureSubIDParam] = subscriptionID
	}

	switch credentials.Kind {
	case ucp_datamodel.AzureServicePrincipalCredentialKind:
		if credentials.ServicePrincipal != nil &&
			credentials.ServicePrincipal.ClientID != "" &&
			credentials.ServicePrincipal.TenantID != "" &&
			credentials.ServicePrincipal.ClientSecret != "" {
			configMap[azureClientIDParam] = credentials.ServicePrincipal.ClientID
			configMap[azureClientSecretParam] = credentials.ServicePrincipal.ClientSecret
			configMap[azureTenantIDParam] = credentials.ServicePrincipal.TenantID
		}
	case ucp_datamodel.AzureWorkloadIdentityCredentialKind:
		if credentials.WorkloadIdentity != nil &&
			credentials.WorkloadIdentity.ClientID != "" &&
			credentials.WorkloadIdentity.TenantID != "" {

			// Use OIDC for Workload Identity
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/service_principal_oidc
			configMap[azureClientIDParam] = credentials.WorkloadIdentity.ClientID
			configMap[azureTenantIDParam] = credentials.WorkloadIdentity.TenantID
			configMap[azureUseCLIParam] = false
			configMap[azureUseOIDCParam] = true
			configMap[azureOIDCTokenFilePathParam] = azureOIDCTokenFilePath
		}
	}

	return configMap
}
