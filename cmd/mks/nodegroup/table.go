/*
Copyright © 2026 Kot1gruN
*/

package nodegroup

import (
	"fmt"
	"strconv"

	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// nodegroupHeader lists the compact columns shown for the table format.
var nodegroupHeader = []string{"ID", "Status", "Cluster ID", "Flavor", "Volume", "Nodes", "Type"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOf(v, nodegroupListViewRow); rows != nil {
		return nodegroupHeader, rows, nil
	}
	if rows := output.RowsOf(v, nodegroupGetViewRow); rows != nil {
		return nodegroupHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func nodegroupRow(id, status, clusterID, flavor, volume, ngType string, nodes int) []string {
	return []string{id, status, clusterID, flavor, volume, strconv.Itoa(nodes), ngType}
}

func volumeFor(local bool, gb int, vtype string) string {
	if local {
		return "local"
	}
	return fmt.Sprintf("%dGB %s", gb, vtype)
}

func nodegroupListViewRow(ng *mksnodegroup.ListView) []string {
	return nodegroupRow(ng.ID, string(ng.Status), ng.ClusterID, ng.FlavorID,
		volumeFor(ng.LocalVolume, ng.VolumeGB, ng.VolumeType), ng.NodegroupType, len(ng.Nodes))
}

func nodegroupGetViewRow(ng *mksnodegroup.GetView) []string {
	return nodegroupRow(ng.ID, string(ng.Status), ng.ClusterID, ng.FlavorID,
		volumeFor(ng.LocalVolume, ng.VolumeGB, ng.VolumeType), ng.NodegroupType, len(ng.Nodes))
}
