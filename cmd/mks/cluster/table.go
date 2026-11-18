/*
Copyright © 2026 Kot1gruN
*/

package cluster

import (
	"fmt"
	"time"

	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
	mkstask "github.com/selectel/mks-go/pkg/v1/task"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// clusterHeader lists the compact columns shown for the cluster table format.
var clusterHeader = []string{"ID", "Name", "Status", "Kube Version", "Region", "Cluster Type", "CNI"}

// taskHeader lists the compact columns shown for the cluster task table format.
var taskHeader = []string{"ID", "Status", "Type", "Cluster ID", "Nodegroup ID"}

// clusterShowHeader is the two-column header used for the show table of a
// single cluster: a vertical parameter-per-row layout reads better than a wide
// single-row table.
var clusterShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow prints a single cluster with its full attribute set. Long-form
// (json/yaml) output serializes the raw GetView value; the table format uses a
// vertical Parameter/Value layout.
func renderShow(cmd *cobra.Command, c *mkscluster.GetView) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, c)
	}
	if c == nil {
		return output.PrintTable(cmd, clusterShowHeader, nil)
	}
	return output.PrintTable(cmd, clusterShowHeader, clusterShowRows(*c))
}

// clusterShowRows renders a single cluster as vertical Parameter/Value rows
// covering the most relevant fields MKS exposes for a cluster.
func clusterShowRows(c mkscluster.GetView) [][]string {
	yesNo := output.YesNo
	return [][]string{
		{"ID", c.ID},
		{"Name", c.Name},
		{"Status", string(c.Status)},
		{"Region", c.Region},
		{"Kube Version", c.KubeVersion},
		{"Cluster Type", string(c.ClusterType)},
		{"CNI", string(c.CNIType)},
		{"Project ID", c.ProjectID},
		{"Network ID", c.NetworkID},
		{"Subnet ID", c.SubnetID},
		{"Kube API IP", c.KubeAPIIP},
		{"Private Kube API", yesNo(c.PrivateKubeAPI)},
		{"Autorepair", yesNo(c.EnableAutorepair)},
		{"Patch Auto Upgrade", yesNo(c.EnablePatchVersionAutoUpgrade)},
		{"Maintenance Window", maintenanceWindow(c.MaintenanceWindowStart, c.MaintenanceWindowEnd)},
		{"Created At", formatTime(c.CreatedAt)},
		{"Updated At", formatTime(c.UpdatedAt)},
	}
}

// maintenanceWindow joins the start/end of the maintenance window into a single
// label, or "-" when the cluster reports neither bound.
func maintenanceWindow(start, end string) string {
	if start == "" && end == "" {
		return "-"
	}
	if start == "" {
		return end
	}
	if end == "" {
		return start
	}
	return start + " - " + end
}

// formatTime renders a nullable timestamp, or "-" when nil.
func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format(time.RFC3339)
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
