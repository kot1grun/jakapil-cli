/*
Copyright © 2026 Kot1gruN
*/
package volume

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	th "github.com/gophercloud/gophercloud/v2/testhelper"
	fake "github.com/gophercloud/gophercloud/v2/testhelper/client"
	"github.com/spf13/cobra"
)

func testCommand(t *testing.T) (*cobra.Command, *strings.Builder) {
	t.Helper()
	cmd := &cobra.Command{Use: "volume"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	// list flags
	fs.String("name", "", "")
	fs.String("status", "", "")
	fs.String("bootable", "", "")
	fs.Int("limit", 0, "")
	fs.String("sort", "", "")
	fs.Bool("all-regions", false, "")
	// create flags
	fs.Int("size", 0, "")
	fs.String("description", "", "")
	fs.String("snapshot-id", "", "")
	fs.String("source-vol-id", "", "")
	fs.String("image-ref", "", "")
	fs.String("backup-id", "", "")
	fs.StringToString("metadata", nil, "")
	// delete flags
	fs.Bool("cascade", false, "")

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunList(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "volumes": [
        {
            "id": "289da7f8-6440-407c-9fb4-7db01ec49164",
            "name": "vol-001",
            "status": "available",
            "size": 75,
            "volume_type": "lvmdriver-1",
            "availability_zone": "nova",
            "bootable": "false",
            "encrypted": false,
            "attachments": [],
            "metadata": {"foo": "bar"},
            "created_at": "2015-09-17T03:35:03.000000",
            "updated_at": "2015-09-17T06:00:00.000000"
        },
        {
            "id": "96c3bda7-c82a-4f50-be73-ca7621794835",
            "name": "vol-002",
            "status": "available",
            "size": 75,
            "volume_type": "lvmdriver-1",
            "availability_zone": "nova",
            "bootable": "false",
            "encrypted": true,
            "attachments": [],
            "metadata": {},
            "created_at": "2015-09-17T03:32:29.000000",
            "updated_at": "2015-09-17T06:00:00.000000"
        }
    ]
}`)
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"vol-001", "vol-002"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunShow(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/volumes/d32019d3-bc6e-4319-9c1d-6722fc136a22", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "volume": {
        "id": "d32019d3-bc6e-4319-9c1d-6722fc136a22",
        "name": "vol-001",
        "status": "available",
        "size": 75,
        "volume_type": "lvmdriver-1",
        "availability_zone": "nova",
        "bootable": "false",
        "encrypted": false,
        "description": "a test volume",
        "attachments": [],
        "user_id": "ff1ce52c03ab433aaba9108c2e3ef541",
        "metadata": {},
        "created_at": "2015-09-17T03:32:29.000000",
        "updated_at": "2015-09-17T06:00:00.000000"
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := runShow(cmd, fake.ServiceClient(fakeServer), []string{"d32019d3-bc6e-4319-9c1d-6722fc136a22"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"vol-001", "a test volume"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q; got:\n%s", want, out.String())
		}
	}
}
func TestRunCreate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{
    "volume": {
        "name": "vol-001",
        "size": 75
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{
    "volume": {
        "id": "d32019d3-bc6e-4319-9c1d-6722fc136a22",
        "name": "vol-001",
        "size": 75,
        "status": "creating",
        "volume_type": "lvmdriver-1",
        "availability_zone": "nova",
        "bootable": "false",
        "encrypted": false,
        "attachments": [],
        "metadata": {},
        "created_at": "2015-09-17T03:32:29.044216",
        "updated_at": "2015-09-17T03:32:29.044216"
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "vol-001"); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Flags().Set("size", "75")

	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "vol-001") {
		t.Fatalf("create output missing created volume; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresNameAndSize(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate without --name succeeded; want an error")
	}

	_ = cmd.Flags().Set("name", "vol-001")
	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate with zero --size succeeded; want an error")
	}
}

func TestRunUpdate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/volumes/d32019d3-bc6e-4319-9c1d-6722fc136a22", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PUT")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{
    "volume": {
        "name": "vol-002",
        "description": "renamed"
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "volume": {
        "id": "d32019d3-bc6e-4319-9c1d-6722fc136a22",
        "name": "vol-002",
        "description": "renamed",
        "status": "available",
        "size": 75,
        "volume_type": "lvmdriver-1",
        "attachments": []
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "vol-002"); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Flags().Set("description", "renamed")

	if err := runUpdate(cmd, fake.ServiceClient(fakeServer), []string{"d32019d3-bc6e-4319-9c1d-6722fc136a22"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "vol-002") {
		t.Fatalf("update output missing updated volume; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOption(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runUpdate(cmd, nil, []string{"d32019d3-bc6e-4319-9c1d-6722fc136a22"}); err == nil {
		t.Fatal("runUpdate without any option succeeded; want an error")
	}
}

func TestRunDelete(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/volumes/d32019d3-bc6e-4319-9c1d-6722fc136a22", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.WriteHeader(http.StatusAccepted)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, fake.ServiceClient(fakeServer), []string{"d32019d3-bc6e-4319-9c1d-6722fc136a22"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "volume d32019d3-bc6e-4319-9c1d-6722fc136a22 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
