/*
Copyright © 2026 Kot1gruN
*/

package cluster

import (
	"fmt"

	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
	mkstask "github.com/selectel/mks-go/pkg/v1/task"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// clusterHeader lists the compact columns shown for the cluster table format.
var clusterHeader = []string{"ID", "Name", "Status", "Kube Version", "Region", "Cluster Type", "CNI"}

// taskHeader lists the compact columns shown for the cluster task table format.
var taskHeader = []string{"ID", "Status", "Type", "Cluster ID", "Nodegroup ID"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOf(v, clusterListViewRow); rows != nil {
		return clusterHeader, rows, nil
	}
	if rows := output.RowsOf(v, clusterGetViewRow); rows != nil {
		return clusterHeader, rows, nil
	}
	if rows := output.RowsOf(v, taskRow); rows != nil {
		return taskHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

// clusterFields is a plain-value projection shared by the ListView and GetView
// row builders (both embed the SDK BaseView, but are distinct Go types).
type clusterFields struct {
	id, name, status, kubeVersion, region, clusterType, cni string
}

func clusterRow(f clusterFields) []string {
	return []string{f.id, f.name, f.status, f.kubeVersion, f.region, f.clusterType, f.cni}
}

func clusterListViewRow(c *mkscluster.ListView) []string {
	return clusterRow(clusterFields{c.ID, c.Name, string(c.Status), c.KubeVersion, c.Region,
		string(c.ClusterType), string(c.CNIType)})
}

func clusterGetViewRow(c *mkscluster.GetView) []string {
	return clusterRow(clusterFields{c.ID, c.Name, string(c.Status), c.KubeVersion, c.Region,
		string(c.ClusterType), string(c.CNIType)})
}

func taskRow(t *mkstask.View) []string {
	return []string{t.ID, string(t.Status), string(t.Type), t.ClusterID, t.NodeGroupID}
}
