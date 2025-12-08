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

package deploy

import (
	"context"
	"fmt"
	"sort"
	"strings"

	v1 "github.com/radius-project/radius/pkg/armrpc/api/v1"
	"github.com/radius-project/radius/pkg/cli"
	"github.com/radius-project/radius/pkg/cli/bicep"
	"github.com/radius-project/radius/pkg/cli/clients"
	"github.com/radius-project/radius/pkg/cli/clierrors"
	"github.com/radius-project/radius/pkg/cli/cmd"
	"github.com/radius-project/radius/pkg/cli/cmd/commonflags"
	"github.com/radius-project/radius/pkg/cli/connections"
	"github.com/radius-project/radius/pkg/cli/deploy"
	"github.com/radius-project/radius/pkg/cli/filesystem"
	"github.com/radius-project/radius/pkg/cli/framework"
	"github.com/radius-project/radius/pkg/cli/output"
	"github.com/radius-project/radius/pkg/cli/workspaces"
	"github.com/radius-project/radius/pkg/corerp/api/v20231001preview"
	"github.com/radius-project/radius/pkg/corerp/api/v20250801preview"
	corerpv20250801 "github.com/radius-project/radius/pkg/corerp/api/v20250801preview"
	"github.com/radius-project/radius/pkg/to"
	"github.com/radius-project/radius/pkg/ucp/resources"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

const (
	appCoreProviderName    = "Applications.Core"
	radiusCoreProviderName = "Radius.Core"
)

// NewCommand creates an instance of the command and runner for the `rad deploy` command.
//

// NewCommand creates a new Cobra command and a Runner to deploy a Bicep or ARM template to a specified environment, with
// optional parameters. It also adds common flags to the command for workspace, resource group, environment name,
// application name and parameters.
func NewCommand(factory framework.Factory) (*cobra.Command, framework.Runner) {
	runner := NewRunner(factory)

	cmd := &cobra.Command{
		Use:   "deploy [file]",
		Short: "Deploy a template",
		Long: `Deploy a Bicep or ARM template
	
The deploy command compiles a Bicep or ARM template and deploys it to your default environment (unless otherwise specified).
	
You can combine Radius types as as well as other types that are available in Bicep such as Azure resources. See
the Radius documentation for information about describing your application and resources with Bicep.

You can specify parameters using the '--parameter' flag ('-p' for short). Parameters can be passed as:

- A file containing multiple parameters using the ARM JSON parameter format (see below)
- A file containing a single value in JSON format
- A key-value-pair passed in the command line

When passing multiple parameters in a single file, use the format described here:

	https://docs.microsoft.com/en-us/azure/azure-resource-manager/templates/parameter-files

You can specify parameters using multiple sources. Parameters can be overridden based on the 
order they are provided. Parameters appearing later in the argument list will override those defined earlier.
`,
		Example: `
# deploy a Bicep template
rad deploy myapp.bicep

# deploy an ARM template (json)
rad deploy myapp.json

# deploy to a specific workspace
rad deploy myapp.bicep --workspace production

# deploy using a specific environment
rad deploy myapp.bicep --environment production

# deploy using a specific environment and resource group
rad deploy myapp.bicep --environment production --group mygroup

# deploy using an environment ID and a resource group. The application will be deployed in mygroup scope, using the specified environment.
# use this option if the environment is in a different group.
rad deploy myapp.bicep --environment /planes/radius/local/resourcegroups/prod/providers/Applications.Core/environments/prod --group mygroup

# specify a string parameter
rad deploy myapp.bicep --parameters version=latest


# specify a non-string parameter using a JSON file
rad deploy myapp.bicep --parameters configuration=@myfile.json


# specify many parameters using an ARM JSON parameter file
rad deploy myapp.bicep --parameters @myfile.json


# specify parameters from multiple sources
rad deploy myapp.bicep --parameters @myfile.json --parameters version=latest
`,
		Args: cobra.ExactArgs(1),
		RunE: framework.RunCommand(runner),
	}

	commonflags.AddWorkspaceFlag(cmd)
	commonflags.AddResourceGroupFlag(cmd)
	commonflags.AddEnvironmentNameFlag(cmd)
	commonflags.AddApplicationNameFlag(cmd)
	commonflags.AddParameterFlag(cmd)

	return cmd, runner
}

// Runner is the runner implementation for the `rad deploy` command.
type Runner struct {
	Bicep                   bicep.Interface
	ConfigHolder            *framework.ConfigHolder
	ConnectionFactory       connections.Factory
	RadiusCoreClientFactory *corerpv20250801.ClientFactory
	Deploy                  deploy.Interface
	Output                  output.Interface

	ApplicationName     string
	EnvironmentNameOrID string
	FilePath            string
	Parameters          map[string]map[string]any
	Workspace           *workspaces.Workspace
	Providers           *clients.Providers
	EnvCheckResult      *EnvironmentCheckResult
}

// NewRunner creates a new instance of the `rad deploy` runner.
func NewRunner(factory framework.Factory) *Runner {
	return &Runner{
		Bicep:             factory.GetBicep(),
		ConnectionFactory: factory.GetConnectionFactory(),
		ConfigHolder:      factory.GetConfigHolder(),
		Deploy:            factory.GetDeploy(),
		Output:            factory.GetOutput(),
	}
}

// Validate runs validation for the `rad deploy` command.
//

// Validate validates the workspace, scope, environment name, application name, and parameters from the command
// line arguments and returns an error if any of these are invalid.
func (r *Runner) Validate(cmd *cobra.Command, args []string) error {
	workspace, err := cli.RequireWorkspace(cmd, r.ConfigHolder.Config, r.ConfigHolder.DirectoryConfig)
	if err != nil {
		return err
	}

	r.Workspace = workspace

	// Allow --group to override the scope
	scope, err := cli.RequireScope(cmd, *workspace)
	if err != nil {
		return err
	}

	// We don't need to explicitly validate the existence of the scope, because we'll validate the existence
	// of the environment later. That will give an appropriate error message for the case where the group
	// does not exist.
	workspace.Scope = scope

	r.EnvironmentNameOrID, err = cli.RequireEnvironmentNameOrID(cmd, args, *workspace)
	if err != nil {
		return err
	}

	// This might be empty, and that's fine!
	r.ApplicationName, err = cli.ReadApplicationName(cmd, *workspace)
	if err != nil {
		return err
	}
	r.Providers = &clients.Providers{}
	r.Providers.Radius = &clients.RadiusProvider{}

	_, parseErr := resources.Parse(r.EnvironmentNameOrID)
	if parseErr != nil {
		envName := r.EnvironmentNameOrID
		// Treat it as name, not an ID - check both providers
		checkResult, err := r.CheckEnvironmentExistence(cmd.Context(), r.EnvironmentNameOrID, cmd, args)
		if err != nil {
			return err
		}

		// If no environment found and user didn't specify environment name, that's ok
		if checkResult == nil {
			if cli.DidSpecifyEnvironmentName(cmd, args) {
				return clierrors.Message("The environment %q does not exist in scope %q. Run `rad env create` first. You could also provide the environment ID if the environment exists in a different group.", envName, r.Workspace.Scope)
			}
			// Environment not required, continue without setting up providers
			return nil
		}

		r.EnvCheckResult = checkResult

		err = r.setupProvidersFromCheckResult()
		if err != nil {
			return err
		}
	} else {
		// It's an ID - use the existing logic
		err = r.setupProvidersFromID(cmd.Context(), cmd, args)
		if err != nil {
			return err
		}
	}

	r.FilePath = args[0]

	parameterArgs, err := cmd.Flags().GetStringArray("parameters")

	if err != nil {
		return err
	}

	parser := bicep.ParameterParser{FileSystem: filesystem.NewOSFS()}
	r.Parameters, err = parser.Parse(parameterArgs...)

	if err != nil {
		return err
	}

	return nil
}

// Run runs the `rad deploy` command.
//

// Run deploys a Bicep template into an environment from a workspace, optionally creating an application if
// specified, and displays progress and completion messages. It returns an error if any of the operations fail.
func (r *Runner) Run(ctx context.Context) error {
	template, err := r.Bicep.PrepareTemplate(r.FilePath)
	if err != nil {
		return err
	}

	// This is the earliest point where we can inject parameters, we have
	// to wait until the template is prepared.
	err = r.injectAutomaticParameters(template)
	if err != nil {
		return err
	}

	// This is the earliest point where we can report missing parameters, we have
	// to wait until the template is prepared.
	err = r.reportMissingParameters(template)
	if err != nil {
		return err
	}

	// Create application if specified. This supports the case where the application resource
	// is not specified in Bicep. Creating the application automatically helps us "bootstrap" in a new environment.
	if r.ApplicationName != "" {
		client, err := r.ConnectionFactory.CreateApplicationsManagementClient(ctx, *r.Workspace)
		if err != nil {
			return err
		}

		// Environment validation has already happened, so only create application if we have an environment
		if r.Workspace.Environment != "" {
			err = client.CreateApplicationIfNotFound(ctx, r.ApplicationName, &v20231001preview.ApplicationResource{
				Location: to.Ptr(v1.LocationGlobal),
				Properties: &v20231001preview.ApplicationProperties{
					Environment: &r.Workspace.Environment,
				},
			})
			if err != nil {
				return err
			}
		}
	}

	progressText := ""
	if r.ApplicationName == "" {
		progressText = fmt.Sprintf(
			"Deploying template '%v' into environment '%v' from workspace '%v'...\n\n"+
				"Deployment In Progress...", r.FilePath, r.EnvironmentNameOrID, r.Workspace.Name)
	} else {
		progressText = fmt.Sprintf(
			"Deploying template '%v' for application '%v' and environment '%v' from workspace '%v'...\n\n"+
				"Deployment In Progress... ", r.FilePath, r.ApplicationName, r.EnvironmentNameOrID, r.Workspace.Name)
	}

	_, err = r.Deploy.DeployWithProgress(ctx, deploy.Options{
		ConnectionFactory: r.ConnectionFactory,
		Workspace:         *r.Workspace,
		Template:          template,
		Parameters:        r.Parameters,
		ProgressText:      progressText,
		CompletionText:    "Deployment Complete",
		Providers:         r.Providers,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Runner) injectAutomaticParameters(template map[string]any) error {
	if r.Providers.Radius.EnvironmentID != "" {
		err := bicep.InjectEnvironmentParam(template, r.Parameters, r.Providers.Radius.EnvironmentID)
		if err != nil {
			return err
		}
	}

	if r.Providers.Radius.ApplicationID != "" {
		err := bicep.InjectApplicationParam(template, r.Parameters, r.Providers.Radius.ApplicationID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) reportMissingParameters(template map[string]any) error {
	declaredParameters, err := bicep.ExtractParameters(template)
	if err != nil {
		return err
	}

	errors := map[string]string{}
	for parameter := range declaredParameters {
		// Case-invariant lookup on the user-provided values
		match := false
		for provided := range r.Parameters {
			if strings.EqualFold(parameter, provided) {
				match = true
				break
			}
		}

		if match {
			// Has user-provided value
			continue
		}

		if _, ok := bicep.DefaultValue(declaredParameters[parameter]); ok {
			// Has default value
			continue
		}

		// Special case the parameters that are automatically injected
		if strings.EqualFold(parameter, "environment") {
			errors[parameter] = "The template requires an environment. Use --environment to specify the environment name."
		} else if strings.EqualFold(parameter, "application") {
			errors[parameter] = "The template requires an application. Use --application to specify the application name."
		} else {
			errors[parameter] = fmt.Sprintf("The template requires a parameter %q. Use --parameters %s=<value> to specify the value.", parameter, parameter)
		}
	}

	if len(errors) == 0 {
		return nil
	}

	keys := maps.Keys(errors)
	sort.Strings(keys)

	details := []string{}
	for _, key := range keys {
		details = append(details, fmt.Sprintf("  - %v", errors[key]))
	}

	return clierrors.Message("The template %q could not be deployed because of the following errors:\n\n%v", r.FilePath, strings.Join(details, "\n"))
}

// setupApplicationsCoreProviders validates and configures providers for Applications Core environment
func (r *Runner) setupApplicationsCoreProviders(ctx context.Context, cmd *cobra.Command, args []string) error {
	env, err := r.getApplicationsCoreEnvironment(ctx, cmd, args)
	if err != nil {
		return err
	}

	return r.configureProviders(env, strings.ToLower(appCoreProviderName))
}

// setupRadiusCoreProviders validates and configures providers for Radius Core environment
func (r *Runner) setupRadiusCoreProviders(ctx context.Context, cmd *cobra.Command, args []string) error {
	env, err := r.getRadiusCoreEnvironment(ctx, cmd, args)
	if err != nil {
		return err
	}

	return r.configureProviders(env, strings.ToLower(radiusCoreProviderName))
}

// setupCloudProviders sets up AWS and Azure providers based on environment properties
func (r *Runner) setupCloudProviders(properties any) {
	switch props := properties.(type) {
	case *v20231001preview.EnvironmentProperties:
		if props != nil && props.Providers != nil {
			if props.Providers.Aws != nil {
				r.Providers.AWS = &clients.AWSProvider{
					Scope: *props.Providers.Aws.Scope,
				}
			}
			if props.Providers.Azure != nil {
				r.Providers.Azure = &clients.AzureProvider{
					Scope: *props.Providers.Azure.Scope,
				}
			}
		}
	case *v20250801preview.EnvironmentProperties:
		if props != nil && props.Providers != nil {
			if props.Providers.Aws != nil {
				r.Providers.AWS = &clients.AWSProvider{
					Scope: *props.Providers.Aws.Scope,
				}
			}
			if props.Providers.Azure != nil {
				r.Providers.Azure = &clients.AzureProvider{
					Scope: "/planes/azure/azure/" + "Subscriptions/" + *props.Providers.Azure.SubscriptionID + "/ResourceGroups/" + *props.Providers.Azure.ResourceGroupName,
				}
			}
		}
	}
}

// determineEnvironmentProvider determines which provider to use based on the environment ID
func (r *Runner) determineEnvironmentProvider() bool {
	useApplicationsCoreEnv := true

	// Check if EnvironmentNameOrID is an ID and parse it
	parsedID, parseErr := resources.Parse(r.EnvironmentNameOrID)
	if parseErr == nil {
		// Use ProviderNamespace() to check the provider namespace
		providerNamespace := parsedID.ProviderNamespace()
		if strings.EqualFold(providerNamespace, appCoreProviderName) {
			useApplicationsCoreEnv = true
		} else if strings.EqualFold(providerNamespace, radiusCoreProviderName) {
			useApplicationsCoreEnv = false
		}
	}

	return useApplicationsCoreEnv
}

// handleEnvironmentError handles common error patterns for environment retrieval
func (r *Runner) handleEnvironmentError(err error, command *cobra.Command, args []string) error {
	// If the error is not a 404, return it
	if !clients.Is404Error(err) {
		return err
	}

	// If the environment doesn't exist, but the user specified its name or resource id as
	// a command-line option, return an error
	if cli.DidSpecifyEnvironmentName(command, args) {
		return clierrors.Message("The environment %q does not exist in scope %q. Run `rad env create` first. You could also provide the environment ID if the environment exists in a different group.", r, r.Workspace.Scope)
	}

	// If we got here, it means that the error was a 404 and the user did not specify the environment name.
	// This is fine, because an environment is not required.
	return nil
}

// handleEnvironmentErrorWithNilReturn handles environment errors and returns nil on 404 with no specification
func (r *Runner) handleEnvironmentErrorWithNilReturn(err error, command *cobra.Command, args []string) (bool, error) {
	handleErr := r.handleEnvironmentError(err, command, args)
	if handleErr != nil {
		return false, handleErr
	}
	// Return true to indicate we should return nil (environment not found but that's ok)
	return true, nil
}

// setupEnvironmentID sets up the environment ID and workspace environment
func (r *Runner) setupEnvironmentID(envID *string) {
	if envID != nil && r.Providers != nil && r.Providers.Radius != nil {
		r.Providers.Radius.EnvironmentID = *envID
		r.Workspace.Environment = r.Providers.Radius.EnvironmentID
	}
}

// getApplicationsCoreEnvironment retrieves environment using Applications Core client
func (r *Runner) getApplicationsCoreEnvironment(ctx context.Context, command *cobra.Command, args []string) (*v20231001preview.EnvironmentResource, error) {
	client, err := r.ConnectionFactory.CreateApplicationsManagementClient(ctx, *r.Workspace)
	if err != nil {
		return nil, err
	}
	env, err := client.GetEnvironment(ctx, r.EnvironmentNameOrID)
	if err != nil {
		return nil, err
	}
	return &env, nil
}

// getRadiusCoreEnvironment retrieves environment using Radius Core client and returns as Applications.Core format
func (r *Runner) getRadiusCoreEnvironment(ctx context.Context, command *cobra.Command, args []string) (*v20250801preview.EnvironmentResource, error) {
	if r.RadiusCoreClientFactory == nil {
		clientFactory, err := cmd.InitializeRadiusCoreClientFactory(ctx, r.Workspace, r.Workspace.Scope)
		if err != nil {
			return nil, err
		}
		r.RadiusCoreClientFactory = clientFactory
	}

	environmentClient := r.RadiusCoreClientFactory.NewEnvironmentsClient()
	// If ID, parse and get the name.
	envName := r.EnvironmentNameOrID
	parsedID, err := resources.Parse(r.EnvironmentNameOrID)
	if err == nil {
		envName = parsedID.Name()
	}
	env, err := environmentClient.Get(ctx, envName, nil)
	if err != nil {
		return nil, err
	}

	return &env.EnvironmentResource, nil
}

// constructEnvironmentID constructs an environment ID from a name and provider type
func (r *Runner) constructEnvironmentID(envName, providerType string) string {
	return r.Workspace.Scope + "/providers/" + providerType + "/environments/" + envName
}

// ConstructApplicationsCoreEnvironmentID constructs an Applications.Core environment ID from a name
func (r *Runner) ConstructApplicationsCoreEnvironmentID(envName string) string {
	return r.constructEnvironmentID(envName, appCoreProviderName)
}

// ConstructRadiusCoreEnvironmentID constructs a Radius.Core environment ID from a name
func (r *Runner) ConstructRadiusCoreEnvironmentID(envName string) string {
	return r.constructEnvironmentID(envName, radiusCoreProviderName)
}

// EnvironmentCheckResult holds the result of checking for environments
type EnvironmentCheckResult struct {
	UseApplicationsCore bool
	ApplicationsCoreEnv *v20231001preview.EnvironmentResource
	RadiusCoreEnv       *v20250801preview.EnvironmentResource
}

// CheckEnvironmentExistence checks both Applications.Core and Radius.Core environments for a given name
func (r *Runner) CheckEnvironmentExistence(ctx context.Context, envName string, command *cobra.Command, args []string) (*EnvironmentCheckResult, error) {
	result := &EnvironmentCheckResult{}

	// Check Applications.Core environment
	appCoreEnvID := r.ConstructApplicationsCoreEnvironmentID(envName)
	tempRunner := &Runner{
		EnvironmentNameOrID: appCoreEnvID,
		Workspace:           r.Workspace,
		ConnectionFactory:   r.ConnectionFactory,
	}
	appCoreEnv, err := tempRunner.getApplicationsCoreEnvironment(ctx, command, args)
	if err != nil {
		appCoreEnv = nil
		// Only return error if it's not a 404
		if !clients.Is404Error(err) {
			return nil, err
		}
	}
	if appCoreEnv != nil {
		result.ApplicationsCoreEnv = appCoreEnv
	}

	// Check Radius.Core environment
	radiusCoreEnvID := r.ConstructRadiusCoreEnvironmentID(envName)
	tempRadiusRunner := &Runner{
		EnvironmentNameOrID:     radiusCoreEnvID,
		Workspace:               r.Workspace,
		RadiusCoreClientFactory: r.RadiusCoreClientFactory,
	}

	// Initialize RadiusCoreClientFactory if needed
	if tempRadiusRunner.RadiusCoreClientFactory == nil {
		clientFactory, err := cmd.InitializeRadiusCoreClientFactory(ctx, r.Workspace, r.Workspace.Scope)
		if err != nil {
			// If we can't initialize the client, we'll assume Radius.Core is not available
			// but we won't fail the whole operation
		} else {
			tempRadiusRunner.RadiusCoreClientFactory = clientFactory
		}
	}

	if tempRadiusRunner.RadiusCoreClientFactory != nil {
		radiusCoreEnv, err := tempRadiusRunner.getRadiusCoreEnvironment(ctx, command, args)
		if err != nil {
			radiusCoreEnv = nil
			// Only return error if it's not a 404
			if !clients.Is404Error(err) {
				return nil, err
			}
		}
		if radiusCoreEnv != nil {
			result.RadiusCoreEnv = radiusCoreEnv
		}
	}

	// Determine which one to use and check for conflicts
	if result.ApplicationsCoreEnv != nil && result.RadiusCoreEnv != nil {
		return nil, clierrors.Message("Conflict detected: Environment '%s' exists in both Applications.Core and Radius.Core providers. Please specify the full resource ID to disambiguate:\n  Applications.Core: %s\n  Radius.Core: %s",
			envName, appCoreEnvID, radiusCoreEnvID)
	}

	if result.ApplicationsCoreEnv != nil {
		result.UseApplicationsCore = true
		r.EnvironmentNameOrID = appCoreEnvID
	} else if result.RadiusCoreEnv != nil {
		result.UseApplicationsCore = false
		r.EnvironmentNameOrID = radiusCoreEnvID
		r.RadiusCoreClientFactory = tempRadiusRunner.RadiusCoreClientFactory
	} else {
		// Neither found, treat as environment not found case
		return nil, nil
	}

	return result, nil
}

// configureProviders configures environment and cloud providers based on the environment and provider type
func (r *Runner) configureProviders(env any, providerType string) error {
	if env != nil {
		switch e := env.(type) {
		case *v20231001preview.EnvironmentResource:
			if e != nil && e.ID != nil {
				r.setupEnvironmentID(e.ID)
				r.setupCloudProviders(e.Properties)
			}
		case *v20250801preview.EnvironmentResource:
			if e != nil && e.ID != nil {
				r.setupEnvironmentID(e.ID)
				r.setupCloudProviders(e.Properties)
			}
		}
	}

	if r.ApplicationName != "" && r.Providers != nil && r.Providers.Radius != nil {
		r.Providers.Radius.ApplicationID = r.Workspace.Scope + "/providers/" + providerType + "/applications/" + r.ApplicationName
	}

	return nil
}

// setupApplicationsCoreProvidersFromResult validates and configures providers for Applications Core environment from check result
func (r *Runner) setupApplicationsCoreProvidersFromResult() error {
	var env any
	if r.EnvCheckResult != nil && r.EnvCheckResult.ApplicationsCoreEnv != nil {
		env = r.EnvCheckResult.ApplicationsCoreEnv
	}

	return r.configureProviders(env, strings.ToLower(appCoreProviderName))
}

// setupRadiusCoreProvidersFromResult validates and configures providers for Radius Core environment from check result
func (r *Runner) setupRadiusCoreProvidersFromResult() error {
	var env any
	if r.EnvCheckResult != nil && r.EnvCheckResult.RadiusCoreEnv != nil {
		env = r.EnvCheckResult.RadiusCoreEnv
	}

	return r.configureProviders(env, strings.ToLower(radiusCoreProviderName))
}

// setupProvidersFromCheckResult sets up providers based on environment check result
func (r *Runner) setupProvidersFromCheckResult() error {
	if r.EnvCheckResult.UseApplicationsCore {
		return r.setupApplicationsCoreProvidersFromResult()
	} else {
		return r.setupRadiusCoreProvidersFromResult()
	}
}

// setupProvidersFromID sets up providers based on environment ID
func (r *Runner) setupProvidersFromID(ctx context.Context, cmd *cobra.Command, args []string) error {
	isAppCoreEnv := r.determineEnvironmentProvider()
	if isAppCoreEnv {
		return r.setupApplicationsCoreProviders(ctx, cmd, args)
	} else {
		return r.setupRadiusCoreProviders(ctx, cmd, args)
	}
}
