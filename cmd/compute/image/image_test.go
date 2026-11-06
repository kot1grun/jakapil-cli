/*
Copyright © 2026 Kot1gruN
*/
package image

import (
	"context"
	"fmt"
	"net/http"
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

	var out strings.Builder
	cmd.SetOut(&out)
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

func TestRunCreateRequiresName(t *testing.T) {
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("container-format", "bare")
	_ = cmd.Flags().Set("disk-format", "qcow2")

	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate without --name succeeded; want an error")
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
