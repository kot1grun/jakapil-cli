/*
Copyright © 2026 Kot1gruN
*/
package flavor

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
	cmd := &cobra.Command{Use: "flavor"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.String("changes-since", "", "")
	fs.Int("min-disk", 0, "")
	fs.Int("min-ram", 0, "")
	fs.String("sort-dir", "", "")
	fs.String("sort-key", "", "")
	fs.String("is-public", "", "")
	fs.Int("limit", 0, "")
	fs.String("name", "", "")
	fs.Int("ram", 0, "")
	fs.Int("vcpus", 0, "")
	fs.Int("disk", 0, "")
	fs.String("id", "", "")
	fs.Int("swap", 0, "")
	fs.Float64("rxtx-factor", 0, "")
	fs.Int("ephemeral", 0, "")
	fs.String("description", "", "")

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunList(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors/detail", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "flavors": [
        {"id": "1", "name": "m1.tiny", "vcpus": 1, "disk": 1, "ram": 512,
         "os-flavor-access:is_public": true, "OS-FLV-EXT-DATA:ephemeral": 0,
         "description": "tiny"},
        {"id": "2", "name": "m1.small", "vcpus": 1, "disk": 20, "ram": 2048,
         "os-flavor-access:is_public": true, "OS-FLV-EXT-DATA:ephemeral": 0,
         "description": "small"}
    ]
}`)
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"m1.tiny", "m1.small"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{
    "flavor": {
        "name": "m1.tiny",
        "ram": 512,
        "vcpus": 1,
        "disk": 1,
        "description": "foo"
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
    "flavor": {
        "id": "1",
        "name": "m1.tiny",
        "vcpus": 1,
        "disk": 1,
        "ram": 512,
        "description": "foo"
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "m1.tiny"); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Flags().Set("ram", "512")
	_ = cmd.Flags().Set("vcpus", "1")
	_ = cmd.Flags().Set("disk", "1")
	_ = cmd.Flags().Set("description", "foo")

	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "m1.tiny") {
		t.Fatalf("create output missing created flavor; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresName(t *testing.T) {
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("ram", "512")
	_ = cmd.Flags().Set("vcpus", "1")
	_ = cmd.Flags().Set("disk", "1")

	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate without --name succeeded; want an error")
	}
}

func TestRunCreateRequiresSizes(t *testing.T) {
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("name", "m1.tiny")
	_ = cmd.Flags().Set("ram", "0")

	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate with zero --ram succeeded; want an error")
	}
}

func TestRunUpdate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors/1", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PUT")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{
    "flavor": {
        "description": "renamed"
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "flavor": {
        "id": "1",
        "name": "m1.tiny",
        "vcpus": 1,
        "disk": 1,
        "ram": 512,
        "description": "renamed"
    }
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("description", "renamed")

	if err := runUpdate(cmd, fake.ServiceClient(fakeServer), []string{"1"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Fatalf("update output missing updated description; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresDescription(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runUpdate(cmd, nil, []string{"1"}); err == nil {
		t.Fatal("runUpdate without --description succeeded; want an error")
	}
}

func TestRunDelete(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors/1", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.WriteHeader(http.StatusAccepted)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, fake.ServiceClient(fakeServer), []string{"1"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "flavor 1 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
