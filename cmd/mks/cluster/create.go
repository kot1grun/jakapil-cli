/*
Copyright © 2026 Kot1gruN
*/
package cluster

import (
	"context"
	"fmt"
	"strings"

	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// newCreateCommand returns the "cluster create" subcommand that creates a
// new Kubernetes cluster together with its initial worker nodegroup.
func newCreateCommand() *cobra.Command {
	var opts createOptions

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new cluster",
		Long: `Create a new Kubernetes cluster in the Selectel Managed Kubernetes
Service, together with an initial worker nodegroup.

At minimum you must provide a cluster name and nodegroup sizing: either
"--flavor-id" or both "--cpus" and "--ram-mb", plus a volume ("--volume-gb"
and "--volume-type", unless "--local-volume" is used).

Example:
  jakapil-cli mks cluster create \
    --name my-cluster --kube-version 1.29.1 --region ru-3 \
    --cpus 4 --ram-mb 8192 --volume-gb 50 --volume-type fast.ru-3a \
    --nodegroup-count 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Name == "" {
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

			cl, _, err := mkscluster.Create(ctx, mksClient, opts.clusterOpts())
			if err != nil {
				return err
			}
			return render(cmd, cl)
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "cluster name")
	cmd.Flags().StringVar(&opts.KubeVersion, "kube-version", "", "Kubernetes version (x.y.z)")
	cmd.Flags().StringVar(&opts.Region, "region", "", "region of the cluster")
	cmd.Flags().StringVar(&opts.NetworkID, "network-id", "", "ID of a pre-created network")
	cmd.Flags().StringVar(&opts.SubnetID, "subnet-id", "", "ID of a pre-created subnet")
	cmd.Flags().StringVar(&opts.ClusterType, "cluster-type", "",
		"cluster type: BASIC, HIGH_AVAILABILITY or HIGH_AVAILABILITY_MULTI_AZ")
	cmd.Flags().StringVar(&opts.CNIType, "cni-type", "", "CNI type: CALICO or CILIUM")
	cmd.Flags().BoolVar(&opts.PrivateKubeAPI, "private-kube-api", false,
		"do not expose the Kubernetes API from the Internet")
	cmd.Flags().StringVar(&opts.MaintenanceWindowStart, "maintenance-window-start", "",
		"maintenance window start in hh:mm:ss UTC")

	cmd.Flags().IntVar(&opts.Count, "nodegroup-count", 1, "number of nodes in the initial nodegroup")
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

// createOptions holds the flag values for creating a cluster.
type createOptions struct {
	// cluster-level options
	Name                   string
	KubeVersion            string
	Region                 string
	NetworkID              string
	SubnetID               string
	ClusterType            string
	CNIType                string
	PrivateKubeAPI         bool
	MaintenanceWindowStart string

	// initial worker nodegroup options
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

// validate checks that the provided flags describe a creatable cluster.
func (o *createOptions) validate() error {
	if o.ClusterType != "" && !isValidClusterType(o.ClusterType) {
		return fmt.Errorf("invalid --cluster-type %q: use BASIC, HIGH_AVAILABILITY or HIGH_AVAILABILITY_MULTI_AZ", o.ClusterType)
	}
	if o.CNIType != "" && strings.ToUpper(o.CNIType) != string(mkscluster.CNITypeCalico) && strings.ToUpper(o.CNIType) != string(mkscluster.CNITypeCilium) {
		return fmt.Errorf("invalid --cni-type %q: use CALICO or CILIUM", o.CNIType)
	}

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

// clusterOpts builds the mks-go cluster CreateOpts from the flag values.
func (o *createOptions) clusterOpts() *mkscluster.CreateOpts {
	ng := &mksnodegroup.CreateOpts{
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

	opts := &mkscluster.CreateOpts{
		Name:                   o.Name,
		KubeVersion:            o.KubeVersion,
		Region:                 o.Region,
		NetworkID:              o.NetworkID,
		SubnetID:               o.SubnetID,
		MaintenanceWindowStart: o.MaintenanceWindowStart,
		PrivateKubeAPI:         boolPtr(o.PrivateKubeAPI),
		Nodegroups:             []*mksnodegroup.CreateOpts{ng},
	}

	if o.ClusterType != "" {
		ct := mkscluster.ClusterType(strings.ToUpper(o.ClusterType))
		opts.ClusterType = &ct
	}
	if o.CNIType != "" {
		opts.CNIType = mkscluster.CNIType(strings.ToUpper(o.CNIType))
	}

	return opts
}

func isValidClusterType(v string) bool {
	switch strings.ToUpper(v) {
	case string(mkscluster.ClusterTypeBasic),
		string(mkscluster.ClusterTypeHighAvailability),
		string(mkscluster.ClusterTypeHighAvailabilityMultiAZ):
		return true
	default:
		return false
	}
}

func boolPtr(b bool) *bool { return &b }
