// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package graph

import (
	"context"

	"github.com/radius-project/radius/pkg/cli"
	"github.com/radius-project/radius/pkg/cli/clients"
	"github.com/radius-project/radius/pkg/cli/clierrors"
	"github.com/radius-project/radius/pkg/cli/cmd/commonflags"
	"github.com/radius-project/radius/pkg/cli/connections"
	"github.com/radius-project/radius/pkg/cli/framework"
	"github.com/radius-project/radius/pkg/cli/output"
	"github.com/radius-project/radius/pkg/cli/workspaces"
	"github.com/spf13/cobra"
)

// NewCommand creates an instance of the command and runner for the `rad app graph` command.
//
// When the positional argument is a Bicep file (*.bicep), the command generates
// a static application graph from the compiled template without deployment.
// Otherwise, it queries the Radius API for the deployed application graph.
func NewCommand(factory framework.Factory) (*cobra.Command, framework.Runner) {
	runner := NewRunner(factory)
	cmd := &cobra.Command{
		Use:   "graph [application or file.bicep]",
		Short: "Shows the application graph for an application.",
		Long: `Shows the application graph for an application.

When a Bicep file path is provided (e.g., app.bicep), the command generates a
static application graph by compiling the template locally. No deployment or
Radius connection is required.

When an application name is provided (or no argument), the command queries the
Radius API for the deployed application graph.`,
		Args: cobra.MaximumNArgs(1),
		Example: `
# Show graph for current application
rad app graph

# Show graph for specified application
rad app graph my-application

# Show static graph from a Bicep file
rad app graph app.bicep

# Show static graph with parameters
rad app graph app.bicep --parameters params.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Detect whether the argument is a Bicep file
			if len(args) > 0 && IsBicepFile(args[0]) {
				paramFile, _ := cmd.Flags().GetString("parameters")
				stdoutFlag, _ := cmd.Flags().GetBool("stdout")
				outputPath, _ := cmd.Flags().GetString("output")
				format, _ := cmd.Flags().GetString("format")
				noGit, _ := cmd.Flags().GetBool("no-git")
				staticRunner := &StaticRunner{
					Output:        factory.GetOutput(),
					Bicep:         factory.GetBicep(),
					FilePath:      args[0],
					ParameterFile: paramFile,
					Stdout:        stdoutFlag,
					OutputPath:    outputPath,
					Format:        format,
					NoGit:         noGit,
				}
				return framework.RunCommand(staticRunner)(cmd, args)
			}

			// Default: query the Radius API for the deployed graph
			return framework.RunCommand(runner)(cmd, args)
		},
	}

	commonflags.AddWorkspaceFlag(cmd)
	commonflags.AddResourceGroupFlag(cmd)
	commonflags.AddApplicationNameFlag(cmd)

	cmd.Flags().String("parameters", "", "Path to a Bicep parameter file (used with .bicep file input)")
	cmd.Flags().Bool("stdout", false, "Write JSON output to stdout instead of a file (static graph only)")
	cmd.Flags().StringP("output", "o", "", "Custom file path for JSON output (static graph only)")
	cmd.Flags().String("format", "", "Additional output format: 'markdown' (static graph only)")
	cmd.Flags().Bool("no-git", false, "Disable git metadata enrichment (static graph only)")

	return cmd, runner
}

// Runner is the runner implementation for the `rad app graph` command.
type Runner struct {
	ConfigHolder      *framework.ConfigHolder
	ConnectionFactory connections.Factory
	Output            output.Interface

	ApplicationName string
	Workspace       *workspaces.Workspace
}

// NewRunner creates a new instance of the `rad app graph` runner.
func NewRunner(factory framework.Factory) *Runner {
	return &Runner{
		ConfigHolder:      factory.GetConfigHolder(),
		Output:            factory.GetOutput(),
		ConnectionFactory: factory.GetConnectionFactory(),
	}
}

// Validate runs validation for the `rad app graph` command.
func (r *Runner) Validate(cmd *cobra.Command, args []string) error {
	workspace, err := cli.RequireWorkspace(cmd, r.ConfigHolder.Config, r.ConfigHolder.DirectoryConfig)
	if err != nil {
		return err
	}
	r.Workspace = workspace

	r.Workspace.Scope, err = cli.RequireScope(cmd, *r.Workspace)
	if err != nil {
		return err
	}

	r.ApplicationName, err = cli.RequireApplicationArgs(cmd, args, *r.Workspace)
	if err != nil {
		return err
	}

	client, err := r.ConnectionFactory.CreateApplicationsManagementClient(cmd.Context(), *r.Workspace)
	if err != nil {
		return err
	}

	// Validate that the application exists
	_, err = client.GetApplication(cmd.Context(), r.ApplicationName)
	if clients.Is404Error(err) {
		return clierrors.Message("Application %q does not exist or has been deleted.", r.ApplicationName)
	} else if err != nil {
		return err
	}

	return nil
}

// Run runs the `rad app graph` command.
func (r *Runner) Run(ctx context.Context) error {
	client, err := r.ConnectionFactory.CreateApplicationsManagementClient(ctx, *r.Workspace)
	if err != nil {
		return err
	}

	applicationGraphResponse, err := client.GetApplicationGraph(ctx, r.ApplicationName)
	if err != nil {
		return err
	}
	graph := applicationGraphResponse.Resources
	display := display(graph, r.ApplicationName)
	r.Output.LogInfo(display)

	return nil
}
