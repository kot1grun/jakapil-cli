/*
Copyright © 2026 Kot1gruN
*/
package cluster

import (
	"context"

	v1 "github.com/selectel/mks-go/pkg/v1"
	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// newShowCommand returns the "cluster show" subcommand that renders the full
// details of a single cluster by its ID. In table mode it prints a vertical
// Parameter/Value table; json/yaml serialize the raw GetView value.
func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <cluster-id>",
		Short: "Show details of a cluster",
		Long:  `Show the full information about a Kubernetes cluster by its ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runShow(cmd, mksClient, args[0])
		},
	}
	return cmd
}

// runShow fetches a single cluster by its ID and renders it through the
// --output flag. cluster.Get requires a service-scoped client, so the --region
// flag (or its default) must identify the cluster's region.
func runShow(cmd *cobra.Command, mksClient *v1.ServiceClient, id string) error {
	ctx := context.Background()
	cl, _, err := mkscluster.Get(ctx, mksClient, id)
	if err != nil {
		return err
	}
	return renderShow(cmd, cl)
}
