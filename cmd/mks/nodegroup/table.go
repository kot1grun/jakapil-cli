/*
Copyright © 2026 Kot1gruN
*/

package nodegroup

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	mksnode "github.com/selectel/mks-go/pkg/v1/node"
	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// nodegroupHeader lists the compact columns shown for the table format.
var nodegroupHeader = []string{"ID", "Status", "Cluster ID", "Flavor", "Volume", "Nodes", "Type"}

// nodegroupShowHeader is the two-column header used for the show table of a
// single nodegroup: a vertical parameter-per-row layout reads better than a wide
// single-row table.
var nodegroupShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow prints a single nodegroup with its full attribute set. Long-form
// (json/yaml) output serializes the raw GetView value; the table format uses a
// vertical Parameter/Value layout.
func renderShow(cmd *cobra.Command, g *mksnodegroup.GetView) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, g)
	}
	if g == nil {
		return output.PrintTable(cmd, nodegroupShowHeader, nil)
	}
	return output.PrintTable(cmd, nodegroupShowHeader, nodegroupShowRows(*g))
}

// nodegroupShowRows renders a single nodegroup as vertical Parameter/Value rows
// covering the most relevant fields MKS exposes for a nodegroup.
func nodegroupShowRows(g mksnodegroup.GetView) [][]string {
	yesNo := output.YesNo
	autoscale := "-"
	if g.EnableAutoscale {
		autoscale = fmt.Sprintf("min=%d max=%d", g.AutoscaleMinNodes, g.AutoscaleMaxNodes)
	}
	return [][]string{
		{"ID", g.ID},
		{"Status", string(g.Status)},
		{"Cluster ID", g.ClusterID},
		{"Flavor", g.FlavorID},
		{"Volume", volumeFor(g.LocalVolume, g.VolumeGB, g.VolumeType)},
		{"Node Count", strconv.Itoa(len(g.Nodes))},
		{"Nodes", nodeHostnames(g.Nodes)},
		{"Nodegroup Type", g.NodegroupType},
		{"Availability Zone", g.AvailabilityZone},
		{"Preemptible", yesNo(g.Preemptible)},
		{"Autoscale", autoscale},
		{"Nvidia Plugin", yesNo(g.InstallNvidiaDevicePlugin)},
		{"Labels", joinLabels(g.Labels)},
		{"Taints", joinTaints(g.Taints)},
		{"Created At", formatNGTime(g.CreatedAt)},
		{"Updated At", formatNGTime(g.UpdatedAt)},
	}
}

// nodeHostnames lists the hostnames of the nodegroup's nodes, one per line, so
// the show table enumerates them instead of counting them. It falls back to the
// node ID when the hostname is empty, and renders "-" when there are no nodes.
func nodeHostnames(nodes []*mksnode.View) string {
	if len(nodes) == 0 {
		return "-"
	}
	names := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if n == nil {
			continue
		}
		h := n.Hostname
		if h == "" {
			h = n.ID
		}
		names = append(names, h)
	}
	if len(names) == 0 {
		return "-"
	}
	return strings.Join(names, "\n")
}

// joinLabels renders a label map as a deterministic comma-separated "k=v" list,
// or "-" when empty.
func joinLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, ", ")
}

// joinTaints renders taints as a deterministic comma-separated "k=v:effect"
// list, or "-" when empty.
func joinTaints(taints []mksnodegroup.Taint) string {
	if len(taints) == 0 {
		return "-"
	}
	sorted := append([]mksnodegroup.Taint(nil), taints...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})
	parts := make([]string, 0, len(sorted))
	for _, t := range sorted {
		parts = append(parts, t.Key+"="+t.Value+":"+string(t.Effect))
	}
	return strings.Join(parts, ", ")
}

// formatNGTime renders a nullable timestamp, or "-" when nil.
func formatNGTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format(time.RFC3339)
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
