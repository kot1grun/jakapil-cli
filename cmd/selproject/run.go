/*
Copyright © 2026 Kot1gruN
*/
package selproject

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/spf13/cobra"

	sdk "github.com/selectel/go-selvpcclient/v5/selvpcclient"
	"github.com/selectel/go-selvpcclient/v5/selvpcclient/resell/v2/projects"
	"jakapil-cli/cmd/selproject/client"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List cloud platform projects",
		Long:  `List all cloud platform projects available to the account.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a cloud platform project",
		Long:  `Create a new cloud platform project. The name is required.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("name", "", "project name (required, max 64 characters)")
	cmd.Flags().Bool("skip-quotas-init", false, "skip the default quotas initialization for the new project")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <project-id>",
		Short: "Update a cloud platform project",
		Long:  `Update attributes of an existing project by its ID. At least one update option is required.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args[0])
		},
	}
	cmd.Flags().String("name", "", "new project name (max 64 characters)")
	cmd.Flags().String("custom-url", "", "custom public URL for the project")
	cmd.Flags().String("theme-color", "", "hex color code for the project theme top panel, e.g. ffffff")
	cmd.Flags().String("theme-logo", "", "URL of the project theme logo")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <project-id>",
		Short:   "Delete a cloud platform project",
		Long:    `Delete a project by its ID.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runDelete(cmd, c, args[0])
		},
	}
	return cmd
}

// runList lists all projects and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *sdk.Client) error {
	list, _, err := projects.List(c)
	if err != nil {
		return resellError(err)
	}
	return render(cmd, list)
}

// runCreate builds the create options from the flags and renders the created
// project.
func runCreate(cmd *cobra.Command, c *sdk.Client) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return fmt.Errorf("project name is required: pass --name")
	}
	skipQuotas, _ := cmd.Flags().GetBool("skip-quotas-init")

	opts := projects.CreateOpts{Name: name, SkipQuotasInit: skipQuotas}

	project, _, err := projects.Create(c, opts)
	if err != nil {
		return resellError(err)
	}
	return render(cmd, project)
}

// runUpdate applies at least one update option and renders the updated project.
func runUpdate(cmd *cobra.Command, c *sdk.Client, id string) error {
	opts := projects.UpdateOpts{}
	applied := false

	if cmd.Flags().Changed("name") {
		val, _ := cmd.Flags().GetString("name")
		opts.Name = val
		applied = true
	}
	if cmd.Flags().Changed("custom-url") {
		val, _ := cmd.Flags().GetString("custom-url")
		opts.CustomURL = &val
		applied = true
	}
	if themeSet(cmd) {
		theme := &projects.ThemeUpdateOpts{}
		if cmd.Flags().Changed("theme-color") {
			val, _ := cmd.Flags().GetString("theme-color")
			theme.Color = &val
		}
		if cmd.Flags().Changed("theme-logo") {
			val, _ := cmd.Flags().GetString("theme-logo")
			theme.Logo = &val
		}
		opts.Theme = theme
		applied = true
	}
	if !applied {
		return fmt.Errorf("at least one update option is required: --name, --custom-url, --theme-color or --theme-logo")
	}

	project, _, err := projects.Update(c, id, opts)
	if err != nil {
		return resellError(err)
	}
	return render(cmd, project)
}

// runDelete removes a project by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *sdk.Client, id string) error {
	if _, err := projects.Delete(c, id); err != nil {
		return resellError(err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "project %s deleted\n", id)
	return nil
}

// themeSet reports whether at least one theme option was passed on the command
// line (and thus should be applied, rather than left at its zero value).
func themeSet(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("theme-color") || cmd.Flags().Changed("theme-logo")
}

// resellError wraps a failed Cloud Platform Projects API call with actionable
// context derived from the HTTP response body, keeping the original error
// wrapped so callers can still type-assert or errors.Is against it.
//
// Selectel enforces token-scope policies server side. The OpenAPI specification
// documents the Projects endpoints as accepting an account-scoped token
// (iam_token_account_scoped), yet some pools (for example new-pool accounts that
// resolve a "resell" endpoint on api.selectel.ru) reject that exact scope with
// "policy_does_not_allow_this_request ... Scope.account scope token". When that
// signature is seen a hint is added instead of silently re-issuing the same
// forbidden request.
func resellError(err error) error {
	if err == nil {
		return nil
	}

	var httpErr gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &httpErr) {
		return err
	}

	body := string(httpErr.Body)
	if !strings.Contains(body, "policy_does_not_allow_this_request") || !strings.Contains(body, "Scope.account") {
		return err
	}

	var b strings.Builder
	b.WriteString("The Projects API rejected this account-scoped token even though the OpenAPI spec documents iam_token_account_scoped for it. This usually means the Resell endpoint resolved from the service catalog points at the wrong pool host.")
	b.WriteString("\nCheck the request against the base URL documented for this pool (https://docs.selectel.ru/api/urls/): some accounts must use a pool-specific host such as <pool>.cloud.api.selcloud.ru instead of api.selectel.ru. Re-run with --debug to print the resolved endpoint, and retry with the correct --auth-url and --region.")
	return fmt.Errorf("%w\n\n%s", err, b.String())
}
