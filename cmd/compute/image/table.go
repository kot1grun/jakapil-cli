/*
Copyright © 2026 Kot1gruN
*/
package image

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// imageHeader lists the compact columns shown for the table format of the
// list command. Plain JSON/YAML render the full record; the table keeps only
// the fields most useful for scanning multiple images.
var imageHeader = []string{"ID", "Name", "Status", "Owner", "Visibility", "Size (Bytes)", "Disk Format", "Container Format", "Protected"}

// imageShowHeader is the two-column header used for a single-image table
// result (show, create, update). An image is rendered vertically, one parameter
// per row, so long properties don't overload a single wide row.
var imageShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderImage renders a single image result (show, create, update). The table
// format uses the vertical Parameter/Value layout; the long formats carry the
// full record including the arbitrary Glance extra-properties, which gophercloud
// keeps in Image.Properties and whose json:"-" tag would otherwise drop them.
func renderImage(cmd *cobra.Command, img *images.Image) error {
	if output.IsTable(cmd) {
		return output.PrintTable(cmd, imageShowHeader, imageShowRows(*img))
	}
	return output.Print(cmd, newImageDisplay(img))
}

// imageDisplay wraps an image for JSON/YAML output so the arbitrary extra
// properties (kept by gophercloud in Image.Properties, tagged json:"-" and thus
// silently dropped from a plain Image serialization) surface as a "properties"
// object on the top level.
type imageDisplay struct {
	images.Image
	Properties map[string]any `json:"properties,omitempty"`
}

func newImageDisplay(img *images.Image) *imageDisplay {
	if img == nil {
		return nil
	}
	return &imageDisplay{Image: *img, Properties: img.Properties}
}

// toTable renders a slice of images as the compact list table and a single
// image (backend returns *images.Image from Create/Update) as a two-column
// Parameter/Value table.
func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]images.Image); ok {
		rows := make([][]string, 0, len(list))
		for _, img := range list {
			rows = append(rows, imageRow(img))
		}
		return imageHeader, rows, nil
	}
	if single, ok := v.(*images.Image); ok {
		if single == nil {
			return imageShowHeader, [][]string{}, nil
		}
		return imageShowHeader, imageShowRows(*single), nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func imageRow(img images.Image) []string {
	return []string{
		img.ID,
		img.Name,
		string(img.Status),
		img.Owner,
		string(img.Visibility),
		fmt.Sprintf("%d", img.SizeBytes),
		img.DiskFormat,
		img.ContainerFormat,
		output.YesNo(img.Protected),
	}
}

// imageShowRows renders a single image as vertical Parameter/Value rows.
func imageShowRows(img images.Image) [][]string {
	rows := [][]string{
		{"ID", img.ID},
		{"Name", img.Name},
		{"Status", string(img.Status)},
		{"Owner", img.Owner},
		{"Visibility", string(img.Visibility)},
		{"Size (Bytes)", fmt.Sprintf("%d", img.SizeBytes)},
		{"Min Disk (GB)", fmt.Sprintf("%d", img.MinDiskGigabytes)},
		{"Min RAM (MB)", fmt.Sprintf("%d", img.MinRAMMegabytes)},
		{"Disk Format", img.DiskFormat},
		{"Container Format", img.ContainerFormat},
		{"Protected", output.YesNo(img.Protected)},
		{"Hidden", output.YesNo(img.Hidden)},
		{"Tags", strings.Join(img.Tags, ", ")},
		{"Checksum", img.Checksum},
		{"Created At", img.CreatedAt.Format("2006-01-02 15:04:05")},
		{"Updated At", img.UpdatedAt.Format("2006-01-02 15:04:05")},
	}
	if len(img.Properties) > 0 {
		parts := make([]string, 0, len(img.Properties))
		for _, k := range sortedAnyKeys(img.Properties) {
			parts = append(parts, fmt.Sprintf("%s=%v", k, img.Properties[k]))
		}
		rows = append(rows, []string{"Properties", strings.Join(parts, "\n")})
	}
	return rows
}

// sortedAnyKeys returns the keys of m in ascending order.
func sortedAnyKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
