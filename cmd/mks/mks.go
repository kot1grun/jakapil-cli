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
	"jakapil-cli/cmd/mks/output"
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
	cmd.PersistentFlags().StringP(client.FlagToken, "t", "", "MKS API token (default $MKS_TOKEN)")
	cmd.PersistentFlags().StringP(client.FlagEndpoint, "e", client.DefaultEndpoint, "MKS API endpoint")
	cmd.PersistentFlags().StringP("output", "o", output.YAML, "output format: json or yaml")

	cmd.AddCommand(
		cluster.NewClusterCommand(),
		kubeoptions.NewKubeOptionsCommand(),
		kubeversion.NewKubeVersionCommand(),
		node.NewNodeCommand(),
		nodegroup.NewNodeGroupCommand(),
	)

	return cmd
}
