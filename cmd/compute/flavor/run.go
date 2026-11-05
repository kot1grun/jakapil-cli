/*
Copyright © 2026 Kot1gruN
*/
package flavor

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists flavors and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchFlavors(cmd.Context(), c, flavorsListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runListAllRegions lists flavors in every region the service advertises and
// renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only GETs.
// A region whose endpoint is unavailable is skipped with a warning on stderr;
// the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewComputeServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := flavorsListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchFlavors(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, f := range list {
			rows = append(rows, flavorRow(f))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, flavorHeader, results)
}

// flavorsListOpts builds the flavors list options from the list flags.
func flavorsListOpts(cmd *cobra.Command) flavors.ListOpts {
	opts := flavors.ListOpts{}
	if v, _ := cmd.Flags().GetString("changes-since"); v != "" {
		opts.ChangesSince = v
	}
	if v, _ := cmd.Flags().GetInt("min-disk"); v > 0 {
		opts.MinDisk = v
	}
	if v, _ := cmd.Flags().GetInt("min-ram"); v > 0 {
		opts.MinRAM = v
	}
	if v, _ := cmd.Flags().GetString("sort-dir"); v != "" {
		opts.SortDir = v
	}
	if v, _ := cmd.Flags().GetString("sort-key"); v != "" {
		opts.SortKey = v
	}
	if v, _ := cmd.Flags().GetString("is-public"); v != "" {
		opts.AccessType = flavors.AccessType(v)
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	return opts
}

// fetchFlavors lists flavors through the paginator and returns them flat.
func fetchFlavors(ctx context.Context, c *gophercloud.ServiceClient, opts flavors.ListOpts) ([]flavors.Flavor, error) {
	var list []flavors.Flavor
	pager := flavors.ListDetail(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageFlavors, err := flavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageFlavors...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// flavorDetail is a standalone mirror of the API's single-flavor object. The
// Selectel compute API does not inline extra-specs in the flavor body; they are
// served by the dedicated /flavors/{id}/os-extra_specs sub-resource (gophercloud
// flavors.Flavor maps extra-specs only from the "extra_specs" body key, so a
// plain flavors.Get().Extract() yields ExtraSpecs: nil). flavorDetail keeps the
// captured specs in Properties, which is what openstackclient renders as the
// flavor's "properties". The struct is flat to also sidestep flavors.Flavor's
// custom UnmarshalJSON, which would swallow the object on embedded decoding.
type flavorDetail struct {
	ID          string            `json:"id"`
	Disk        int               `json:"disk"`
	RAM         int               `json:"ram"`
	Name        string            `json:"name"`
	RxTxFactor  float64           `json:"rxtx_factor"`
	Swap        any               `json:"swap"`
	VCPUs       int               `json:"vcpus"`
	IsPublic    bool              `json:"os-flavor-access:is_public"`
	Ephemeral   int               `json:"OS-FLV-EXT-DATA:ephemeral"`
	Description string            `json:"description"`
	Properties  map[string]string `json:"properties"`
}

// toFlavor converts a decoded flavorDetail back into a gophercloud flavors.Flavor
// so the shared table render path can be reused. The captured "properties" map
// is placed in ExtraSpecs (the canonical gophercloud name for flavor extra-specs).
func (d flavorDetail) toFlavor() flavors.Flavor {
	return flavors.Flavor{
		ID:          d.ID,
		Disk:        d.Disk,
		RAM:         d.RAM,
		Name:        d.Name,
		RxTxFactor:  d.RxTxFactor,
		Swap:        d.swapInt(),
		VCPUs:       d.VCPUs,
		IsPublic:    d.IsPublic,
		Ephemeral:   d.Ephemeral,
		Description: d.Description,
		ExtraSpecs:  d.Properties,
	}
}

// swapInt normalizes the API's "swap" value, which may arrive either as a JSON
// number or as a string ("" or "1024"), back into an int.
func (d flavorDetail) swapInt() int {
	switch v := d.Swap.(type) {
	case float64:
		return int(v)
	case string:
		if v == "" {
			return 0
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

// runShow performs two raw GET requests: one on /flavors/{id} for the flavor
// record and one on /flavors/{id}/os-extra_specs for its specs (Selectel does
// not inline them in the flavor body). For table output it renders
// flavors.Flavor (with a dedicated Properties column); for long-form (json/yaml)
// it prints flavorDetail directly so the specs appear under "properties" exactly
// like openstackclient.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	var body struct {
		Flavor flavorDetail `json:"flavor"`
	}
	resp, err := c.Get(cmd.Context(), c.ServiceURL("flavors", args[0]), &body, nil)
	if _, _, err := gophercloud.ParseResponse(resp, err); err != nil {
		return err
	}

	if specs, err := getExtraSpecs(cmd, c, args[0]); err == nil {
		body.Flavor.Properties = specs
	}

	if output.IsTable(cmd) {
		f := body.Flavor.toFlavor()
		return render(cmd, &f)
	}
	return output.Print(cmd, body.Flavor)
}

// getExtraSpecs returns the flavor's extra-specs from the dedicated
// /flavors/{id}/os-extra_specs sub-resource. Selectel serves them there (and
// nowhere in the flavor body); openstackclient renders this exact map as the
// flavor's "properties". An error here is returned untouched so the caller can
// decide how to surface it.
func getExtraSpecs(cmd *cobra.Command, c *gophercloud.ServiceClient, id string) (map[string]string, error) {
	var body struct {
		ExtraSpecs map[string]string `json:"extra_specs"`
	}
	resp, err := c.Get(cmd.Context(), c.ServiceURL("flavors", id, "os-extra_specs"), &body, nil)
	if _, _, err := gophercloud.ParseResponse(resp, err); err != nil {
		return nil, err
	}
	return body.ExtraSpecs, nil
}

// runCreate builds the create options from the flags and renders the created
// flavor.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	name, _ := cmd.Flags().GetString("name")
	ram, _ := cmd.Flags().GetInt("ram")
	vcpus, _ := cmd.Flags().GetInt("vcpus")
	disk, _ := cmd.Flags().GetInt("disk")
	if name == "" {
		return fmt.Errorf("flavor name is required: pass --name")
	}
	if ram <= 0 || vcpus <= 0 {
		return fmt.Errorf("--ram and --vcpus must be greater than zero")
	}
	// High-frequency flavors impose a minimum root disk that grows with the
	// vCPU count: at least 4 GB per vCPU up to 14 vCPUs, and 8 GB per vCPU
	// beyond that. The disk flag must always be provided explicitly, since its
	// default of 0 cannot satisfy the requirement.
	highfreq, _ := cmd.Flags().GetBool("highfreq")
	if cmd.Flags().Changed("highfreq") && highfreq {
		if !cmd.Flags().Changed("disk") {
			return fmt.Errorf("--disk is required when --highfreq is set")
		}
		minDisk := 4 * vcpus
		if vcpus > 14 {
			minDisk = 8 * vcpus
		}
		if disk < minDisk {
			return fmt.Errorf("--disk must be at least %d GB when --highfreq is set (%d vCPU)", minDisk, vcpus)
		}
	}

	opts := flavors.CreateOpts{
		Name:  name,
		RAM:   ram,
		VCPUs: vcpus,
		Disk:  &disk,
	}
	if v, _ := cmd.Flags().GetString("id"); v != "" {
		opts.ID = v
	}
	if v, _ := cmd.Flags().GetInt("swap"); v > 0 {
		swap := v
		opts.Swap = &swap
	}
	if v, _ := cmd.Flags().GetFloat64("rxtx-factor"); v != 0 {
		opts.RxTxFactor = v
	}
	// jakapil is a client-side tool that creates flavors for the caller's own
	// project, so flavors are always private: os-flavor-access:is_public is
	// pinned to false (omitting it would let the server default to public).
	isPublic := false
	opts.IsPublic = &isPublic
	if v, _ := cmd.Flags().GetInt("ephemeral"); v > 0 {
		ephemeral := v
		opts.Ephemeral = &ephemeral
	}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		opts.Description = v
	}

	flavor, err := flavors.Create(cmd.Context(), c, createOptsBuilder(opts, cmd)).Extract()
	if err != nil {
		return err
	}
	return render(cmd, flavor)
}

// selectelFlavorOpts extends flavors.CreateOpts through Go struct embedding to
// add Selectel-specific parameters to the created flavor. Only the fields set
// in the sent struct are injected, so a plain create keeps the body
// byte-for-byte unchanged:
//
//   - highFreq (from --highfreq) adds both "highfreq" and its companion
//     "dedicated_cpus".
//
// vcpusVendor wiring is disabled for the time being: the active API
// microversion does not accept the vcpus_vendor property on flavor create.
// See the commented-out code below for the intended behavior.
type selectelFlavorOpts struct {
	flavors.CreateOpts
	highFreq *bool
	// Disabled for now (--vcpus-vendor): re-enable once the API supports it.
	// vcpusVendor *string
}

// ToFlavorCreateMap builds the request body from the embedded flavors.CreateOpts
// and injects the Selectel-specific fields that were explicitly requested.
func (o selectelFlavorOpts) ToFlavorCreateMap() (map[string]any, error) {
	body, err := o.CreateOpts.ToFlavorCreateMap()
	if err != nil {
		return nil, err
	}
	if flavorMap, ok := body["flavor"].(map[string]any); ok {
		if o.highFreq != nil {
			flavorMap["highfreq"] = *o.highFreq
			flavorMap["dedicated_cpus"] = *o.highFreq
		}
		// Disabled for now (--vcpus-vendor):
		// if o.vcpusVendor != nil && *o.vcpusVendor != "" {
		// 	flavorMap["vcpus_vendor"] = *o.vcpusVendor
		// }
	}
	return body, nil
}

// createOptsBuilder returns the flavors.CreateOptsBuilder to feed to
// flavors.Create. When no Selectel-specific flag is set it returns the plain
// opts, keeping the body unchanged. Otherwise it wraps opts in selectelFlavorOpts
// so the corresponding Selectel fields are emitted with the request.
func createOptsBuilder(opts flavors.CreateOpts, cmd *cobra.Command) flavors.CreateOptsBuilder {
	highfreq, _ := cmd.Flags().GetBool("highfreq")

	var highFreq *bool
	if cmd.Flags().Changed("highfreq") {
		highFreq = &highfreq
	}
	// Disabled for now: keep reading --vcpus-vendor out of the payload until
	// the active API microversion accepts vcpus_vendor on flavor create.
	//
	// vcpusVendor, _ := cmd.Flags().GetString("vcpus-vendor")
	// var vendor *string
	// if vcpusVendor != "" {
	// 	vendor = &vcpusVendor
	// }
	// if highFreq == nil && vendor == nil {
	// 	return opts
	// }
	if highFreq == nil {
		return opts
	}
	return selectelFlavorOpts{CreateOpts: opts, highFreq: highFreq}
}

// runUpdate applies at least one update option and renders the updated flavor.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if !cmd.Flags().Changed("description") {
		return fmt.Errorf("at least one update option is required: --description")
	}

	opts := flavors.UpdateOpts{
		Description: mustString(cmd, "description"),
	}

	flavor, err := flavors.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, flavor)
}

// runDelete removes a flavor by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := flavors.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "flavor %s deleted\n", args[0])
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
