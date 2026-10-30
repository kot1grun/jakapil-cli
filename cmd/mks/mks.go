/*
Copyright © 2026 Kot1gruN
*/
package mks

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
	"jakapil-cli/cmd/mks/cluster"
	"jakapil-cli/cmd/mks/kubeoptions"
	"jakapil-cli/cmd/mks/kubeversion"
	"jakapil-cli/cmd/mks/node"
	"jakapil-cli/cmd/mks/nodegroup"
	"jakapil-cli/internal/output"
)

// NewMksCommand returns the parent "mks" command with its subcommands:
// cluster, kubeoptions, kubeversion, node and nodegroup.
func NewMksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mks",
		Short: "Manage Selectel Managed Kubernetes Service",
		Long: `Manage Selectel Managed Kubernetes Service (MKS).
The following objects are supported: clusters, nodegroups, nodes,
Kubernetes versions and options.`,
	}

	// Persistent flags are shared by all mks subcommands.
	cmd.PersistentFlags().StringP(client.FlagToken, "t", "", "Selectel Keystone API token (default $SLT_KS_TOKEN)")
	cmd.PersistentFlags().StringP(client.FlagEndpoint, "e", client.DefaultEndpoint, "Selectel Keystone Identity v3 auth URL for authentication and service discovery (default $OS_AUTH_URL)")
	cmd.PersistentFlags().String(client.FlagRegion, "", "target region for MKS endpoint discovery, e.g. ru-1 (default $OS_REGION_NAME)")
	cmd.PersistentFlags().Bool(client.FlagDebug, false, "print the discovered service catalog and MKS endpoints to stderr")
	cmd.PersistentFlags().StringP("output", "o", output.YAML, "output format: json, yaml or table")

	cmd.AddCommand(
		cluster.NewClusterCommand(),
		kubeoptions.NewKubeOptionsCommand(),
		kubeversion.NewKubeVersionCommand(),
		node.NewNodeCommand(),
		nodegroup.NewNodeGroupCommand(),
	)

	return cmd
}
