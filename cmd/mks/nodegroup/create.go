/*
Copyright © 2026 Kot1gruN
*/
package nodegroup

import (
	"context"
	"fmt"
	"net/http"

	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// newCreateCommand returns the "nodegroup create" subcommand that creates a
// new worker nodegroup in an existing cluster.
//
// At minimum you must provide nodegroup sizing: either "--flavor-id" or both
// "--cpus" and "--ram-mb", plus a volume ("--volume-gb" and "--volume-type",
// unless "--local-volume" is used).
func newCreateCommand() *cobra.Command {
	var (
		clusterID string
		opts      createOptions
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a nodegroup",
		Long: `Create a new worker nodegroup in an existing Kubernetes cluster.

At minimum you must provide nodegroup sizing: either "--flavor-id" or both
"--cpus" and "--ram-mb", plus a volume ("--volume-gb" and "--volume-type",
unless "--local-volume" is used).

Example:
  jakapil-cli mks nodegroup create --cluster-id <cluster-id> \
    --cpus 4 --ram-mb 8192 --volume-gb 50 --volume-type fast.ru-3a \
    --count 2`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}
			if opts.Count == 0 && opts.FlavorID == "" && opts.CPUs == 0 && opts.RAMMB == 0 {
				return cmd.Help()
			}
			if err := opts.validate(); err != nil {
				return err
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			res, err := mksnodegroup.Create(ctx, mksClient, clusterID, opts.nodegroupOpts())
			if err != nil {
				return err
			}
			// Create is async and commonly returns 200, 202 Accepted or 204 No
			// Content; treat any 2xx status as success.
			if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("unexpected HTTP status %d while creating nodegroup in cluster %q", res.StatusCode, clusterID)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "nodegroup created in cluster %s\n", clusterID)
			return err
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().IntVar(&opts.Count, "count", 1, "number of nodes in the nodegroup")
	cmd.Flags().StringVar(&opts.FlavorID, "flavor-id", "", "OpenStack flavor ID for the nodes")
	cmd.Flags().IntVar(&opts.CPUs, "cpus", 0, "CPU count per node (used when --flavor-id is empty)")
	cmd.Flags().IntVar(&opts.RAMMB, "ram-mb", 0, "RAM in MB per node (used when --flavor-id is empty)")
	cmd.Flags().IntVar(&opts.VolumeGB, "volume-gb", 0, "volume size in GB per node")
	cmd.Flags().StringVar(&opts.VolumeType, "volume-type", "", "block storage volume type per node")
	cmd.Flags().BoolVar(&opts.LocalVolume, "local-volume", false, "use local volumes for the nodes")
	cmd.Flags().StringVar(&opts.KeypairName, "keypair-name", "", "SSH keypair name added to the nodes")
	cmd.Flags().StringVar(&opts.AvailabilityZone, "availability-zone", "", "availability zone for the nodes")

	return cmd
}

// createOptions holds the flag values for creating a nodegroup.
type createOptions struct {
	Count            int
	FlavorID         string
	CPUs             int
	RAMMB            int
	VolumeGB         int
	VolumeType       string
	LocalVolume      bool
	KeypairName      string
	AvailabilityZone string
}

// validate checks that the provided flags describe a creatable nodegroup.
func (o *createOptions) validate() error {
	hasFlavor := o.FlavorID != ""
	hasSize := o.CPUs > 0 && o.RAMMB > 0
	if !hasFlavor && !hasSize {
		return fmt.Errorf("nodegroup sizing is required: pass --flavor-id or both --cpus and --ram-mb")
	}
	if !o.LocalVolume && (o.VolumeGB <= 0 || o.VolumeType == "") {
		return fmt.Errorf("nodegroup volume is required: pass --volume-gb and --volume-type, or --local-volume")
	}

	return nil
}

// nodegroupOpts builds the mks-go nodegroup CreateOpts from the flag values.
func (o *createOptions) nodegroupOpts() *mksnodegroup.CreateOpts {
	return &mksnodegroup.CreateOpts{
		Count:            o.Count,
		FlavorID:         o.FlavorID,
		CPUs:             o.CPUs,
		RAMMB:            o.RAMMB,
		VolumeGB:         o.VolumeGB,
		VolumeType:       o.VolumeType,
		LocalVolume:      o.LocalVolume,
		KeypairName:      o.KeypairName,
		AvailabilityZone: o.AvailabilityZone,
	}
}
