/*
Copyright © 2026 Kot1gruN
*/
package publicsubnet

import (
	"fmt"
	"strings"

	selvpcclient "github.com/selectel/go-selvpcclient/v5/selvpcclient"
	"github.com/selectel/go-selvpcclient/v5/selvpcclient/resell/v2/subnets"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List VPC public subnets",
		Long:  `List the public subnets in the current project through the Resell API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := NewResellClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [project-id]",
		Short: "Create a VPC public subnet",
		Long: `Create a new public subnet in the given project through the Resell API.

The project ID may be passed as a positional argument. When it is omitted, the
project is inferred from the project scope of the token used for
authentication: the Public Subnets API is project-scoped, so a project token
already carries the target project. A project ID passed explicitly always takes
precedence.

The region (--region), IP version (--type) and prefix length (--prefix-length)
define the requested subnet.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := NewResellClient(cmd)
			if err != nil {
				return err
			}
			projectID := ""
			if len(args) == 1 {
				projectID = args[0]
			}
			return runCreate(cmd, c, projectID)
		},
	}
	cmd.Flags().String("region", "", "region of the subnet, e.g. ru-1 (required)")
	cmd.Flags().String("type", "", "IP version: ipv4 or ipv6 (required)")
	cmd.Flags().Int("prefix-length", 0, "prefix length of the subnet, e.g. 29 (required)")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <subnet-id>",
		Short:   "Delete a VPC public subnet",
		Long:    `Delete a public subnet by its numeric Resell ID.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := NewResellClient(cmd)
			if err != nil {
				return err
			}
			return runDelete(cmd, c, args[0])
		},
	}
}

// runList lists all public subnets and renders the result through the --output
// flag.
func runList(cmd *cobra.Command, c *selvpcclient.Client) error {
	list, _, err := subnets.List(c, subnets.ListOpts{})
	if err != nil {
		return err
	}
	if len(list) == 0 && output.IsTable(cmd) {
		return output.PrintListEmpty(cmd, "")
	}
	return render(cmd, list)
}

// runCreate builds the create options from the flags and renders the created
// subnet. When projectID is empty it is inferred from the project scope of the
// token used for authentication.
func runCreate(cmd *cobra.Command, c *selvpcclient.Client, projectID string) error {
	region, _ := cmd.Flags().GetString("region")
	if region == "" {
		return fmt.Errorf("region is required: pass --region")
	}

	version := strings.ToLower(strings.TrimSpace(mustString(cmd, "type")))
	var ipType selvpcclient.IPVersion
	switch version {
	case "ipv4":
		ipType = selvpcclient.IPv4
	case "ipv6":
		ipType = selvpcclient.IPv6
	default:
		return fmt.Errorf("unsupported IP version %q: use ipv4 or ipv6", version)
	}

	prefixLength, _ := cmd.Flags().GetInt("prefix-length")
	if prefixLength <= 0 || prefixLength > 128 {
		return fmt.Errorf("invalid prefix length %d: must be in range 1..128", prefixLength)
	}

	if projectID == "" {
		inferred, err := resolveProjectID(cmd)
		if err != nil {
			return fmt.Errorf("inferring project ID from the token: %w", err)
		}
		if inferred == "" {
			return fmt.Errorf("project ID is required: the token used for authentication is not scoped to a project, " +
				"pass it explicitly as the positional argument or use a project-scoped token")
		}
		projectID = inferred
	}

	opts := subnets.SubnetOpts{
		Region:       region,
		Type:         ipType,
		PrefixLength: prefixLength,
	}

	subnet, _, err := subnets.Create(c, projectID, opts)
	if err != nil {
		return err
	}
	return render(cmd, subnet)
}

// runDelete removes a public subnet by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *selvpcclient.Client, id string) error {
	_, err := subnets.Delete(c, id)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "public subnet %s deleted\n", id)
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
