/*
Copyright © 2026 Kot1gruN
*/
package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists images and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchImages(cmd.Context(), c, imageListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runShow fetches a single image by its ID and renders it through the --output
// flag.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	image, err := images.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderImage(cmd, image)
}

// runListAllRegions lists images in every region the service advertises and
// renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only GETs.
// A region whose endpoint is unavailable is skipped with a warning on stderr;
// the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewImageV2ServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := imageListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchImages(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, img := range list {
			rows = append(rows, imageRow(img))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, imageHeader, results)
}

// imageListOpts builds the images list options from the list flags.
func imageListOpts(cmd *cobra.Command) images.ListOpts {
	opts := images.ListOpts{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		opts.Name = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		opts.Status = images.ImageStatus(v)
	}
	if v, _ := cmd.Flags().GetString("visibility"); v != "" {
		opts.Visibility = images.ImageVisibility(v)
	}
	if v, _ := cmd.Flags().GetString("owner"); v != "" {
		opts.Owner = v
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	if v, _ := cmd.Flags().GetString("sort"); v != "" {
		opts.Sort = v
	}
	if v, _ := cmd.Flags().GetStringArray("tag"); len(v) > 0 {
		opts.Tags = v
	}
	if v, _ := cmd.Flags().GetInt64("min-size"); v > 0 {
		opts.SizeMin = v
	}
	if v, _ := cmd.Flags().GetInt64("max-size"); v > 0 {
		opts.SizeMax = v
	}
	if v, _ := cmd.Flags().GetString("disk-format"); v != "" {
		opts.DiskFormat = v
	}
	if v, _ := cmd.Flags().GetString("container-format"); v != "" {
		opts.ContainerFormat = v
	}
	if hidden, _ := cmd.Flags().GetBool("hidden"); hidden {
		opts.Hidden = true
	}
	return opts
}

// fetchImages lists images through the paginator and returns them flat.
// The --limit option caps the total number of returned images: the server
// treats it as a per-page size and, via the "next" link, would otherwise feed
// every remaining page, defeating the user's intent to get only N results.
func fetchImages(ctx context.Context, c *gophercloud.ServiceClient, opts images.ListOpts) ([]images.Image, error) {
	var list []images.Image
	pager := images.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageImages, err := images.ExtractImages(page)
		if err != nil {
			return false, err
		}
		for _, img := range pageImages {
			list = append(list, img)
			// Stop walking the pages as soon as the limit is reached.
			if opts.Limit > 0 && len(list) >= opts.Limit {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		return nil, err
	}
	// Guard against a server that ignores limit and returns more than opts.Limit
	// in a single page.
	if opts.Limit > 0 && len(list) > opts.Limit {
		list = list[:opts.Limit]
	}
	return list, nil
}

// runCreate creates an image and renders the result through the --output flag.
// When --file is set, the local image data is uploaded to Glance right after the
// record is created, and the freshly-loaded image (status "active") is then
// fetched and rendered.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	opts, err := imageCreateOpts(cmd)
	if err != nil {
		return err
	}

	image, err := images.Create(cmd.Context(), c, opts).Extract()
	if err != nil {
		return err
	}

	if file := mustString(cmd, "file"); file != "" {
		if err := uploadImageData(cmd.Context(), cmd, c, image, file); err != nil {
			return err
		}
		// Re-fetch to reflect the post-upload status (queued -> saving -> active)
		// instead of the record's initial "queued" state.
		image, err = images.Get(cmd.Context(), c, image.ID).Extract()
		if err != nil {
			return fmt.Errorf("fetch freshly uploaded image %s: %w", image.ID, err)
		}
	}
	return renderImage(cmd, image)
}

// liftHTTPTimeout temporarily removes the total http.Client.Timeout from the
// provider HTTP client and returns a restore func to call via defer. The client
// carries a 30s cap tuned for fast control-plane calls, but Go's
// http.Client.Timeout bounds the whole request, including reading the response
// body — far too short for streaming multi-GB image payloads in either
// direction. Only the timeout is dropped; Transport/Jar/redirects are kept.
func liftHTTPTimeout(c *gophercloud.ServiceClient) func() {
	pc := c.ProviderClient
	saved := pc.HTTPClient
	noTimeout := saved
	noTimeout.Timeout = 0
	pc.HTTPClient = noTimeout
	return func() { pc.HTTPClient = saved }
}

// uploadImageData uploads a local file as the payload of the given image. The
// call is long-running for large files, so a progress bar is rendered on the
// command's stderr (kept separate from stdout so --output json/yaml stays clean).
func uploadImageData(ctx context.Context, cmd *cobra.Command, c *gophercloud.ServiceClient, img *images.Image, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open image file: %w", err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat image file: %w", err)
	}

	// The provider HTTP client carries a short total timeout (30s) tuned for
	// fast discovery calls. Streaming a large image can take far longer, and
	// http.Client.Timeout aborts the request mid-stream. Lift it for the upload.
	defer liftHTTPTimeout(c)()

	bar := progressbar.NewOptions64(
		st.Size(),
		progressbar.OptionSetDescription("Uploading image data"),
		progressbar.OptionSetWriter(cmd.ErrOrStderr()),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(20),
		progressbar.OptionClearOnFinish(),
	)

	// NewReader feeds bar.Add(n) on every chunk read from the file, so the bar
	// tracks real bytes pushed to Glance rather than a fake percentage.
	// Read has a pointer receiver, hence &reader.
	reader := progressbar.NewReader(f, bar)

	// The library only renders within Add() once the progress is > 0% (the
	// "currentPercent > 0" guard), so nothing would be drawn until the first
	// few KB are read. Render the empty 0% line up front so the bar is visible
	// from the very start of the upload.
	if err := bar.RenderBlank(); err != nil {
		return fmt.Errorf("render progress bar: %w", err)
	}

	if err := imagedata.Upload(ctx, c, img.ID, &reader).ExtractErr(); err != nil {
		return fmt.Errorf("upload image data for %s: %w", img.ID, err)
	}
	return nil
}

// runDownload streams the raw image payload into the --file target (or stdout
// when --file is omitted) and renders a download progress bar on stderr.
func runDownload(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	// http.Client.Timeout covers reading the response body too, so a large image
	// download can be cut off long after the headers arrive. Lift the limit while
	// the stream is drained (restored on return by the defer).
	defer liftHTTPTimeout(c)()

	res := imagedata.Download(cmd.Context(), c, args[0])
	defer res.Body.Close()
	if res.Err != nil {
		return res.Err
	}

	var w io.Writer = cmd.OutOrStdout()
	if file := mustString(cmd, "file"); file != "" {
		// Never clobber an existing file without an explicit --force: a large
		// download would otherwise silently truncate and destroy its content.
		if _, err := os.Stat(file); err == nil {
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				return fmt.Errorf("download file %q already exists: remove it or pass --force to overwrite", file)
			}
		}
		f, err := os.Create(file)
		if err != nil {
			return fmt.Errorf("create download file %q: %w", file, err)
		}
		defer f.Close()
		w = f
	}

	if err := streamImageData(cmd, res.Body, res.Header.Get("Content-Length"), w); err != nil {
		return fmt.Errorf("download image data for %s: %w", args[0], err)
	}
	return nil
}

// streamImageData copies body into w while rendering a progress bar. maximum, when
// provided by the server via the Content-Length header, yields a determinate
// bar; otherwise the bar runs in an indeterminate (spinner) mode.
func streamImageData(cmd *cobra.Command, body io.Reader, contentLength string, w io.Writer) error {
	max := int64(-1)
	if cl := strings.TrimSpace(contentLength); cl != "" {
		if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
			max = n
		}
	}

	bar := progressbar.NewOptions64(
		max,
		progressbar.OptionSetDescription("Downloading image data"),
		progressbar.OptionSetWriter(cmd.ErrOrStderr()),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(20),
		progressbar.OptionClearOnFinish(),
	)
	// Render the empty bar up front: the library only draws within Add() once
	// progress is > 0%, so without this nothing appears before the first chunk.
	if err := bar.RenderBlank(); err != nil {
		return fmt.Errorf("render progress bar: %w", err)
	}
	reader := progressbar.NewReader(body, bar)
	if _, err := io.Copy(w, &reader); err != nil {
		return err
	}
	return nil
}

// runUpload uploads local image data into an existing image, which must be in
// the "queued" state (Glance forbids uploading into an active image). The
// progress bar and timeout handling are shared with the create --file path.
func runUpload(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	file := mustString(cmd, "file")
	if file == "" {
		return fmt.Errorf("image file is required: pass --file")
	}

	img, err := images.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	if img.Status != images.ImageStatusQueued {
		return fmt.Errorf("cannot upload data to image %s in status %q: only images in %q can accept data",
			img.ID, img.Status, images.ImageStatusQueued)
	}

	if err := uploadImageData(cmd.Context(), cmd, c, img, file); err != nil {
		return err
	}

	// Re-fetch to reflect the post-upload status (queued -> saving -> active)
	// instead of the pre-upload "queued" state.
	img, err = images.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return fmt.Errorf("fetch freshly uploaded image %s: %w", img.ID, err)
	}
	return renderImage(cmd, img)
}

// imageCreateOpts builds the images.CreateOpts from the create flags and
// validates the required fields.
func imageCreateOpts(cmd *cobra.Command) (images.CreateOpts, error) {
	opts := images.CreateOpts{
		Name:            mustString(cmd, "name"),
		ID:              mustString(cmd, "id"),
		Tags:            mustStringArray(cmd, "tag"),
		ContainerFormat: mustString(cmd, "container-format"),
		DiskFormat:      mustString(cmd, "disk-format"),
		MinDisk:         mustInt(cmd, "min-disk"),
		MinRAM:          mustInt(cmd, "min-ram"),
	}
	if opts.Name == "" {
		return images.CreateOpts{}, fmt.Errorf("image name is required: pass --name")
	}
	if opts.ContainerFormat == "" {
		return images.CreateOpts{}, fmt.Errorf("container format is required: pass --container-format")
	}
	if opts.DiskFormat == "" {
		return images.CreateOpts{}, fmt.Errorf("disk format is required: pass --disk-format")
	}

	if v, _ := cmd.Flags().GetString("visibility"); v != "" {
		vis := images.ImageVisibility(v)
		opts.Visibility = &vis
	}
	if hidden, _ := cmd.Flags().GetBool("hidden"); hidden {
		h := hidden
		opts.Hidden = &h
	}
	if protected, _ := cmd.Flags().GetBool("protected"); protected {
		p := protected
		opts.Protected = &p
	}
	if props, _ := cmd.Flags().GetStringToString("property"); len(props) > 0 {
		opts.Properties = props
	}
	return opts, nil
}

// runUpdate applies the requested image updates and renders the updated image.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts, err := imageUpdateOpts(cmd)
	if err != nil {
		return err
	}

	image, err := images.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return renderImage(cmd, image)
}

// imageUpdateOpts assembles the images.UpdateOpts (a JSON patch document) from
// the update flags, returning an error when no update was requested.
func imageUpdateOpts(cmd *cobra.Command) (images.UpdateOpts, error) {
	var patches images.UpdateOpts

	if cmd.Flags().Changed("name") {
		patches = append(patches, images.ReplaceImageName{NewName: mustString(cmd, "name")})
	}
	if cmd.Flags().Changed("visibility") {
		patches = append(patches, images.UpdateVisibility{
			Visibility: images.ImageVisibility(mustString(cmd, "visibility")),
		})
	}
	if cmd.Flags().Changed("protected") {
		protected, _ := cmd.Flags().GetBool("protected")
		patches = append(patches, images.ReplaceImageProtected{NewProtected: protected})
	}
	if cmd.Flags().Changed("hidden") {
		hidden, _ := cmd.Flags().GetBool("hidden")
		patches = append(patches, images.ReplaceImageHidden{NewHidden: hidden})
	}
	if cmd.Flags().Changed("min-disk") {
		patches = append(patches, images.ReplaceImageMinDisk{NewMinDisk: mustInt(cmd, "min-disk")})
	}
	if cmd.Flags().Changed("min-ram") {
		patches = append(patches, images.ReplaceImageMinRam{NewMinRam: mustInt(cmd, "min-ram")})
	}
	if tags := mustStringArray(cmd, "tag"); len(tags) > 0 {
		patches = append(patches, images.ReplaceImageTags{NewTags: tags})
	}
	if props, _ := cmd.Flags().GetStringToString("property"); len(props) > 0 {
		// Deterministic order: map key iteration is random in Go.
		for _, k := range sortedMapKeys(props) {
			patches = append(patches, images.UpdateImageProperty{
				Op:    images.AddOp,
				Name:  k,
				Value: props[k],
			})
		}
	}
	for _, key := range mustStringArray(cmd, "remove-property") {
		patches = append(patches, images.UpdateImageProperty{
			Op:   images.RemoveOp,
			Name: key,
		})
	}

	if len(patches) == 0 {
		return nil, fmt.Errorf("at least one update option is required: --name, --visibility, --protected, --hidden, --min-disk, --min-ram, --tag, --property or --remove-property")
	}
	return patches, nil
}

// runDelete removes an image by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := images.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "image %s deleted\n", args[0])
	return nil
}

func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func mustInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}

func mustStringArray(cmd *cobra.Command, name string) []string {
	v, _ := cmd.Flags().GetStringArray(name)
	return v
}

// sortedMapKeys returns the keys of m in ascending order.
func sortedMapKeys(m map[string]string) []string {
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
