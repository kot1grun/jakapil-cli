/*
Copyright © 2026 Kot1gruN
*/
package image

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	fake "github.com/gophercloud/gophercloud/v2/testhelper/client"
	"github.com/spf13/cobra"
)

// testCommand builds a bare command with every image subcommand flag registered
// and a captured stdout buffer. The response bodies follow the fixtures from
// github.com/gophercloud/gophercloud/v2/openstack/image/v2/images/testing.
func testCommand(t *testing.T) (*cobra.Command, *strings.Builder) {
	t.Helper()
	cmd := &cobra.Command{Use: "image"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")

	// list flags
	fs.String("name", "", "")
	fs.String("status", "", "")
	fs.String("visibility", "", "")
	fs.String("owner", "", "")
	fs.Int("limit", 0, "")
	fs.String("sort", "", "")
	fs.StringArray("tag", nil, "")
	fs.Int64("min-size", 0, "")
	fs.Int64("max-size", 0, "")
	fs.String("disk-format", "", "")
	fs.String("container-format", "", "")
	fs.Bool("hidden", false, "")

	// create / update flags
	fs.String("id", "", "")
	fs.Int("min-disk", 0, "")
	fs.Int("min-ram", 0, "")
	fs.Bool("protected", false, "")
	fs.StringToString("property", nil, "")
	fs.StringArray("remove-property", nil, "")
	fs.StringP("file", "f", "", "")
	fs.Bool("force", false, "")
	fs.Bool("no-progress", false, "")

	var out strings.Builder
	cmd.SetOut(&out)
	// Route progress-bar output (stderr) to a throwaway buffer so tests don't
	// spam the runner with \r-rewriting progress lines.
	cmd.SetErr(&strings.Builder{})
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunList(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "images": [
        {
            "status": "active",
            "name": "cirros-0.3.4-x86_64-uec",
            "tags": [],
            "container_format": "ami",
            "created_at": "2015-07-15T11:43:35Z",
            "disk_format": "ami",
            "updated_at": "2015-07-15T11:43:35Z",
            "visibility": "public",
            "min_disk": 0,
            "protected": false,
            "id": "07aa21a9-fa1a-430e-9a33-185be5982431",
            "size": 25165824,
            "checksum": "eb9139e4942121f22bbc2afc0400b2a4",
            "owner": "cba624273b8344e59dd1fd18685183b0",
            "min_ram": 0
        },
        {
            "status": "active",
            "name": "cirros-0.3.4-x86_64-uec-kernel",
            "tags": [],
            "container_format": "aki",
            "created_at": "2015-07-15T11:43:29Z",
            "disk_format": "aki",
            "updated_at": "2015-07-15T11:43:30Z",
            "visibility": "public",
            "min_disk": 0,
            "protected": false,
            "id": "e1b6edd4-bd9b-40ac-b010-8a6c16de4ba4",
            "size": 4979632,
            "checksum": "8a40c862b5735975d82605c1dd395796",
            "owner": "cba624273b8344e59dd1fd18685183b0",
            "min_ram": 0
        }
    ]
}`)
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"cirros-0.3.4-x86_64-uec", "cirros-0.3.4-x86_64-uec-kernel"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}
func TestRunCreate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{
    "name": "cirros-0.4.0-x86_64-disk",
    "container_format": "bare",
    "disk_format": "qcow2"
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
    "status": "queued",
    "name": "cirros-0.4.0-x86_64-disk",
    "tags": [],
    "container_format": "bare",
    "created_at": "2015-11-12T13:33:30Z",
    "disk_format": "qcow2",
    "updated_at": "2015-11-12T13:33:30Z",
    "visibility": "private",
    "id": "2aa8abd3-4f4d-4e99-a744-f0e5730ea101",
    "size": 0,
    "owner": "",
    "min_ram": 0,
    "min_disk": 0
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "cirros-0.4.0-x86_64-disk")
	_ = cmd.Flags().Set("container-format", "bare")
	_ = cmd.Flags().Set("disk-format", "qcow2")

	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "cirros-0.4.0-x86_64-disk") {
		t.Fatalf("create output missing created image; got:\n%s", out.String())
	}
}

// TestRunCreateWithFile verifies that --file uploads the local payload into
// Glance right after the record is created, and that the freshly-loaded image is
// then rendered.
func TestRunCreateWithFile(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	payload := []byte("fake-qcow2-image-bytes")

	// 1. Create the record (status queued).
	fakeServer.Mux.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{
    "status": "queued",
    "name": "cirros-0.4.0-x86_64-disk",
    "tags": [],
    "container_format": "bare",
    "created_at": "2015-11-12T13:33:30Z",
    "disk_format": "qcow2",
    "updated_at": "2015-11-12T13:33:30Z",
    "visibility": "private",
    "id": %q,
    "size": 0,
    "owner": "",
    "min_ram": 0,
    "min_disk": 0
}`, imageID)
	})

	// 2. Upload the payload.
	fakeServer.Mux.HandleFunc("/images/"+imageID+"/file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PUT")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestHeader(t, r, "Content-Type", "application/octet-stream")
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upload body: %v", err)
		}
		if string(got) != string(payload) {
			t.Fatalf("upload body = %q; want %q", got, payload)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// 3. Refresh the image (now active).
	fakeServer.Mux.HandleFunc("/images/"+imageID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "status": "active",
    "name": "cirros-0.4.0-x86_64-disk",
    "tags": [],
    "container_format": "bare",
    "created_at": "2015-11-12T13:33:30Z",
    "disk_format": "qcow2",
    "updated_at": "2015-11-12T13:33:33Z",
    "visibility": "private",
    "id": "`+imageID+`",
    "size": 22,
    "owner": "",
    "min_ram": 0,
    "min_disk": 0
}`)
	})

	tmp := t.TempDir()
	file := tmp + "/cirros.qcow2"
	if err := os.WriteFile(file, payload, 0o600); err != nil {
		t.Fatalf("write temp image file: %v", err)
	}

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "cirros-0.4.0-x86_64-disk")
	_ = cmd.Flags().Set("container-format", "bare")
	_ = cmd.Flags().Set("disk-format", "qcow2")
	_ = cmd.Flags().Set("file", file)

	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate --file: %v", err)
	}
	if !strings.Contains(out.String(), "active") {
		t.Fatalf("create --file output missing refreshed 'active' status; got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "cirros-0.4.0-x86_64-disk") {
		t.Fatalf("create --file output missing image name; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresName(t *testing.T) {
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("container-format", "bare")
	_ = cmd.Flags().Set("disk-format", "qcow2")

	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate without --name succeeded; want an error")
	}
}

// TestRunUpload uploads data into a queued image and verifies the bar passes
// the payload and the refreshed "active" image is rendered.
func TestRunUpload(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	payload := []byte("fake-qcow2-image-bytes")

	// 1. Fetch the existing (queued) image.
	queueCalls := 0
	fakeServer.Mux.HandleFunc("/images/"+imageID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		queueCalls++
		w.Header().Add("Content-Type", "application/json")
		status := "queued"
		if queueCalls > 1 {
			status = "active"
		}
		fmt.Fprintf(w, `{
    "status": %q,
    "name": "cirros-0.4.0-x86_64-disk",
    "tags": [],
    "container_format": "bare",
    "created_at": "2015-11-12T13:33:30Z",
    "disk_format": "qcow2",
    "updated_at": "2015-11-12T13:33:35Z",
    "visibility": "private",
    "id": %q,
    "size": 22,
    "owner": "",
    "min_ram": 0,
    "min_disk": 0
}`, status, imageID)
	})

	// 2. Upload the payload.
	fakeServer.Mux.HandleFunc("/images/"+imageID+"/file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PUT")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestHeader(t, r, "Content-Type", "application/octet-stream")
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upload body: %v", err)
		}
		if string(got) != string(payload) {
			t.Fatalf("upload body = %q; want %q", got, payload)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	tmp := t.TempDir()
	file := tmp + "/cirros.qcow2"
	if err := os.WriteFile(file, payload, 0o600); err != nil {
		t.Fatalf("write temp image file: %v", err)
	}

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("file", file)
	if err := runUpload(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err != nil {
		t.Fatalf("runUpload: %v", err)
	}
	if queueCalls != 2 {
		t.Fatalf("expected 2 GET calls (pre + post), got %d", queueCalls)
	}
	if !strings.Contains(out.String(), "active") {
		t.Fatalf("upload output missing refreshed 'active' status; got:\n%s", out.String())
	}
}

// TestRunUploadNoProgress verifies the upload path still streams the payload
// correctly when the progress bar is suppressed with --no-progress.
func TestRunUploadNoProgress(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	payload := []byte("raw-upload-no-progress")

	fakeServer.Mux.HandleFunc("/images/"+imageID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "status": "queued",
    "name": "cirros-0.4.0-x86_64-disk",
    "container_format": "bare",
    "disk_format": "qcow2",
    "visibility": "private",
    "id": "`+imageID+`"
}`)
	})
	fakeServer.Mux.HandleFunc("/images/"+imageID+"/file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PUT")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upload body: %v", err)
		}
		if string(got) != string(payload) {
			t.Fatalf("upload body = %q; want %q", got, payload)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	tmp := t.TempDir()
	file := tmp + "/no-progress.qcow2"
	if err := os.WriteFile(file, payload, 0o600); err != nil {
		t.Fatalf("write temp image file: %v", err)
	}

	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("file", file)
	_ = cmd.Flags().Set("no-progress", "true")
	if err := runUpload(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err != nil {
		t.Fatalf("runUpload --no-progress: %v", err)
	}
}
func TestRunUploadRequiresFile(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runUpload(cmd, nil, []string{"some-id"}); err == nil {
		t.Fatal("runUpload without --file succeeded; want an error")
	}
}

func TestRunUploadNotQueued(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	fakeServer.Mux.HandleFunc("/images/"+imageID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "status": "active",
    "name": "cirros-0.4.0-x86_64-disk",
    "container_format": "bare",
    "disk_format": "qcow2",
    "visibility": "private",
    "id": "`+imageID+`",
    "size": 22,
    "owner": "",
    "min_ram": 0,
    "min_disk": 0
}`)
	})

	cmd, _ := testCommand(t)
	tmp := t.TempDir()
	file := tmp + "/img.qcow2"
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_ = cmd.Flags().Set("file", file)

	err := runUpload(cmd, fake.ServiceClient(fakeServer), []string{imageID})
	if err == nil {
		t.Fatal("runUpload on an active image succeeded; want an error")
	}
	if !strings.Contains(err.Error(), "only images in") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRunDownloadToFile verifies that the raw payload is streamed into --file
// and byte-for-byte matches the served body.
func TestRunDownloadToFile(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	payload := []byte("fake-qcow2-download-payload")
	fakeServer.Mux.HandleFunc("/images/"+imageID+"/file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		_, _ = w.Write(payload)
	})

	out := t.TempDir() + "/downloaded.qcow2"
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("file", out)
	if err := runDownload(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err != nil {
		t.Fatalf("runDownload: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("downloaded content = %q; want %q", got, payload)
	}
}

// TestRunDownloadNoProgress verifies the payload streams correctly with the
// progress bar disabled via --no-progress.
func TestRunDownloadNoProgress(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	payload := []byte("raw-download-no-progress")
	fakeServer.Mux.HandleFunc("/images/"+imageID+"/file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		_, _ = w.Write(payload)
	})

	out := t.TempDir() + "/no-progress.qcow2"
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("file", out)
	_ = cmd.Flags().Set("no-progress", "true")
	if err := runDownload(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err != nil {
		t.Fatalf("runDownload --no-progress: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("downloaded content = %q; want %q", got, payload)
	}
}

// TestProgressEnabled verifies the toggle produced by --no-progress.
func TestProgressEnabled(t *testing.T) {
	cmd, _ := testCommand(t)

	_ = cmd.Flags().Set("no-progress", "true")
	if progressEnabled(cmd) {
		t.Fatal("progressEnabled = true with --no-progress set; want false")
	}

	_ = cmd.Flags().Set("no-progress", "false")
	if !progressEnabled(cmd) {
		t.Fatal("progressEnabled = false without --no-progress; want true")
	}
}

// TestRunDownloadRefusesOverwrite verifies an existing target file is not
// silently clobbered without --force.
func TestRunDownloadRefusesOverwrite(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	fakeServer.Mux.HandleFunc("/images/"+imageID+"/file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		_, _ = w.Write([]byte("new-payload"))
	})

	out := t.TempDir() + "/existing.qcow2"
	if err := os.WriteFile(out, []byte("old-data"), 0o600); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}

	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("file", out)
	err := runDownload(cmd, fake.ServiceClient(fakeServer), []string{imageID})
	if err == nil {
		t.Fatal("runDownload over an existing file succeeded; want an error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
	// The file must be left untouched.
	got, _ := os.ReadFile(out)
	if string(got) != "old-data" {
		t.Fatalf("existing file was modified: %q", got)
	}
}

// TestRunDownloadForceOverwrite verifies --force allows clobbering the target.
func TestRunDownloadForceOverwrite(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	newPayload := []byte("new-payload")
	fakeServer.Mux.HandleFunc("/images/"+imageID+"/file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		_, _ = w.Write(newPayload)
	})

	out := t.TempDir() + "/existing.qcow2"
	if err := os.WriteFile(out, []byte("old-data"), 0o600); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}

	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("file", out)
	_ = cmd.Flags().Set("force", "true")
	if err := runDownload(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err != nil {
		t.Fatalf("runDownload --force: %v", err)
	}
	got, _ := os.ReadFile(out)
	if string(got) != string(newPayload) {
		t.Fatalf("downloaded content = %q; want %q", got, newPayload)
	}
}

// TestRunDownloadToStdout verifies the payload lands on stdout when --file is
// omitted (the bar itself goes to stderr, which the test routes to a buffer).
func TestRunDownloadToStdout(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	fakeServer.Mux.HandleFunc("/images/"+imageID+"/file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		_, _ = w.Write([]byte("raw-bytes"))
	})

	cmd, out := testCommand(t)
	if err := runDownload(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err != nil {
		t.Fatalf("runDownload stdout: %v", err)
	}
	if out.String() != "raw-bytes" {
		t.Fatalf("stdout content = %q; want %q", out.String(), "raw-bytes")
	}
}

// TestRunDownloadErrorForwards surfaces transport/API errors from Download.
func TestRunDownloadErrorForwards(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "missing-image-id"
	fakeServer.Mux.HandleFunc("/images/"+imageID+"/file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		http.NotFound(w, r)
	})

	cmd, _ := testCommand(t)
	if err := runDownload(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err == nil {
		t.Fatal("runDownload on a missing image succeeded; want an error")
	}
}

func TestRunShow(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "2aa8abd3-4f4d-4e99-a744-f0e5730ea101"
	fakeServer.Mux.HandleFunc("/images/"+imageID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "status": "active",
    "name": "cirros-0.4.0-x86_64-disk",
    "tags": [],
    "container_format": "bare",
    "created_at": "2015-11-12T13:33:30Z",
    "disk_format": "qcow2",
    "updated_at": "2015-11-12T13:33:35Z",
    "visibility": "private",
    "id": "2aa8abd3-4f4d-4e99-a744-f0e5730ea101",
    "size": 13287936,
    "owner": "41553c2845531ef9daae337c3a741f6b",
    "min_ram": 0,
    "min_disk": 0,
    "os_hash_algo": "sha512",
    "hw_disk_bus": "virtio",
    "os_type": "linux",
    "owner_project_name": "kot1grun"
}`)
	})

	cmd, out := testCommand(t)
	if err := runShow(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"cirros-0.4.0-x86_64-disk", imageID, "qcow2"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q; got:\n%s", want, out.String())
		}
	}
	// Extra Glance properties must be surfaced (in YAML they land under a
	// top-level "properties" key), not silently dropped.
	for _, want := range []string{"os_hash_algo", "sha512", "hw_disk_bus", "virtio", "owner_project_name", "kot1grun"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing extra property %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunShowNotFound(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "missing-image-id"
	fakeServer.Mux.HandleFunc("/images/"+imageID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": {"message": "Image not found", "code": 404}}`)
	})

	cmd, _ := testCommand(t)
	if err := runShow(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err == nil {
		t.Fatal("runShow for a missing image succeeded; want an error")
	}
}

func TestRunUpdate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	imageID := "da3b75d9-3f4a-40e7-8a2c-bfab23927dea"
	fakeServer.Mux.HandleFunc("/images/"+imageID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `[
    {
        "op": "replace",
        "path": "/name",
        "value": "Fedora 17"
    }
]`)
		th.AssertEquals(t, "application/openstack-images-v2.1-json-patch", r.Header.Get("Content-Type"))
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "id": "da3b75d9-3f4a-40e7-8a2c-bfab23927dea",
    "name": "Fedora 17",
    "status": "active",
    "visibility": "public",
    "size": 2254249,
    "tags": ["fedora", "beefy"],
    "created_at": "2012-08-10T19:23:50Z",
    "updated_at": "2012-08-12T11:11:33Z",
    "disk_format": "",
    "container_format": ""
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "Fedora 17")

	if err := runUpdate(cmd, fake.ServiceClient(fakeServer), []string{imageID}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "Fedora 17") {
		t.Fatalf("update output missing updated image; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOption(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runUpdate(cmd, nil, []string{"da3b75d9-3f4a-40e7-8a2c-bfab23927dea"}); err == nil {
		t.Fatal("runUpdate without any option succeeded; want an error")
	}
}

func TestRunDelete(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/images/2aa8abd3-4f4d-4e99-a744-f0e5730ea101", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.WriteHeader(http.StatusNoContent)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, fake.ServiceClient(fakeServer), []string{"2aa8abd3-4f4d-4e99-a744-f0e5730ea101"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "2aa8abd3-4f4d-4e99-a744-f0e5730ea101") {
		t.Fatalf("delete output missing image id; got:\n%s", out.String())
	}
}

// imageJSON returns a minimal image object fixture for the given id.
func imageJSON(id string) string {
	return `{
        "status": "active",
        "name": "img-` + id + `",
        "tags": [],
        "container_format": "bare",
        "created_at": "2015-07-15T11:43:35Z",
        "disk_format": "qcow2",
        "updated_at": "2015-07-15T11:43:35Z",
        "visibility": "private",
        "min_disk": 0,
        "protected": false,
        "id": "` + id + `",
        "size": 1024,
        "owner": "cba624273b8344e59dd1fd18685183b0",
        "min_ram": 0
    }`
}

// TestFetchImagesRespectsLimit verifies two things: the server does receive a
// limit query parameter, and fetchImages stops walking the pages once the
// limit is reached instead of following the "next" link to the bitter end.
func TestFetchImagesRespectsLimit(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	var calls int
	fakeServer.Mux.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		calls++
		th.TestMethod(t, r, "GET")
		// The API does receive the limit parameter on the first page.
		if calls == 1 && r.URL.Query().Get("limit") != "5" {
			t.Fatalf("list did not send limit in query; got %q", r.URL.Query().Get("limit"))
		}
		w.Header().Add("Content-Type", "application/json")
		if calls == 1 {
			// First page: 6 images (more than the limit) plus a next link.
			fmt.Fprintf(w, `{
    "images": [
        %s,%s,%s,%s,%s,%s
    ],
    "next": "/images?marker=img-00000006"
}`, imageJSON("00000001"), imageJSON("00000002"), imageJSON("00000003"),
				imageJSON("00000004"), imageJSON("00000005"), imageJSON("00000006"))
			return
		}
		// Second page: 3 more images, no next link.
		fmt.Fprintf(w, `{
    "images": [
        %s,%s,%s
    ]
}`, imageJSON("00000004"), imageJSON("00000005"), imageJSON("00000006"))
	})

	list, err := fetchImages(context.Background(), fake.ServiceClient(fakeServer), images.ListOpts{Limit: 5})
	if err != nil {
		t.Fatalf("fetchImages: %v", err)
	}
	if len(list) != 5 {
		t.Fatalf("fetchImages with limit=5 returned %d images; want 5 (pages walked: %d)", len(list), calls)
	}
	// Because the client stops at the limit, the second page must never be fetched.
	if calls != 1 {
		t.Fatalf("fetchImages walked %d pages but should have stopped at the limit", calls)
	}
}
