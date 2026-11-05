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
	fs.Bool("highfreq", false, "")
	fs.Bool("dedicated-vcpus", false, "")
	// Disabled for now (--vcpus-vendor), see flavor.go.
	// fs.String("vcpus-vendor", "", "")

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
        "os-flavor-access:is_public": false,
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

// TestRunCreateZeroDisk verifies that a flavor with disk = 0 is allowed and the
// request body carries "disk": 0.
func TestRunCreateZeroDisk(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{
    "flavor": {
        "name": "m1.nodisk",
        "ram": 512,
        "vcpus": 1,
        "disk": 0,
        "os-flavor-access:is_public": false
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
    "flavor": {
        "id": "7",
        "name": "m1.nodisk",
        "vcpus": 1,
        "disk": 0,
        "ram": 512
    }
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "m1.nodisk")
	_ = cmd.Flags().Set("ram", "512")
	_ = cmd.Flags().Set("vcpus", "1")
	_ = cmd.Flags().Set("disk", "0")

	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate with disk=0: %v", err)
	}
	if !strings.Contains(out.String(), "m1.nodisk") {
		t.Fatalf("create output missing created flavor; got:\n%s", out.String())
	}
}

func TestRunCreateHighFreq(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{
    "flavor": {
        "name": "m1.highfreq",
        "ram": 4096,
        "vcpus": 4,
        "disk": 40,
        "os-flavor-access:is_public": false,
        "highfreq": true
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
    "flavor": {
        "id": "hf-1",
        "name": "m1.highfreq",
        "vcpus": 4,
        "disk": 40,
        "ram": 4096,
        "os-flavor-access:is_public": true,
        "OS-FLV-EXT-DATA:ephemeral": 0,
        "highfreq": true
    }
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "m1.highfreq")
	_ = cmd.Flags().Set("ram", "4096")
	_ = cmd.Flags().Set("vcpus", "4")
	_ = cmd.Flags().Set("disk", "40")
	_ = cmd.Flags().Set("highfreq", "true")

	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate with --highfreq: %v", err)
	}
	if !strings.Contains(out.String(), "m1.highfreq") {
		t.Fatalf("create output missing created flavor; got:\n%s", out.String())
	}
}

// TestRunCreateHighFreqRequiresDisk verifies that --highfreq mandates an
// explicitly set --disk.
func TestRunCreateHighFreqRequiresDisk(t *testing.T) {
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("name", "m1.highfreq")
	_ = cmd.Flags().Set("ram", "4096")
	_ = cmd.Flags().Set("vcpus", "4")
	_ = cmd.Flags().Set("highfreq", "true")

	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate with --highfreq but no --disk succeeded; want an error")
	}
}

// TestRunCreateHighFreqDiskTooSmall verifies that --highfreq rejects a disk
// below the minimum of 4 GB per vCPU (vcpus <= 14).
func TestRunCreateHighFreqDiskTooSmall(t *testing.T) {
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("name", "m1.highfreq")
	_ = cmd.Flags().Set("ram", "4096")
	_ = cmd.Flags().Set("vcpus", "4")
	_ = cmd.Flags().Set("disk", "8")
	_ = cmd.Flags().Set("highfreq", "true")

	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate with --highfreq and disk below 4*vcpu succeeded; want an error")
	}
}

// TestRunCreateHighFreqLargeVcpusRequiresDisk verifies the 8 GB per vCPU
// minimum once --vcpus exceeds 14.
func TestRunCreateHighFreqLargeVcpusRequiresDisk(t *testing.T) {
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("name", "m1.highfreq")
	_ = cmd.Flags().Set("ram", "65536")
	_ = cmd.Flags().Set("vcpus", "16")
	_ = cmd.Flags().Set("disk", "64")
	_ = cmd.Flags().Set("highfreq", "true")

	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate with --highfreq, 16 vCPU and disk below 8*16 succeeded; want an error")
	}
}

// TestRunCreateDedicatedCPUs verifies that --dedicated-vcpus is an independent
// flag: it sends "dedicated_cpus" in the body without implying --highfreq.
func TestRunCreateDedicatedCPUs(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{
    "flavor": {
        "name": "m1.dedicated",
        "ram": 2048,
        "vcpus": 4,
        "disk": 20,
        "os-flavor-access:is_public": false,
        "dedicated_cpus": true
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
    "flavor": {
        "id": "ded-1",
        "name": "m1.dedicated",
        "vcpus": 4,
        "disk": 20,
        "ram": 2048,
        "os-flavor-access:is_public": true,
        "OS-FLV-EXT-DATA:ephemeral": 0
    }
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "m1.dedicated")
	_ = cmd.Flags().Set("ram", "2048")
	_ = cmd.Flags().Set("vcpus", "4")
	_ = cmd.Flags().Set("disk", "20")
	_ = cmd.Flags().Set("dedicated-vcpus", "true")

	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate with --dedicated-vcpus: %v", err)
	}
	if !strings.Contains(out.String(), "m1.dedicated") {
		t.Fatalf("create output missing created flavor; got:\n%s", out.String())
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

func TestRunShow(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors/2", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "flavor": {
        "id": "2",
        "name": "m1.small",
        "vcpus": 1,
        "disk": 20,
        "ram": 2048,
        "swap": "",
        "os-flavor-access:is_public": true,
        "OS-FLV-EXT-DATA:ephemeral": 0,
        "description": "small"
    }
}`)
	})
	fakeServer.Mux.HandleFunc("/flavors/2/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "extra_specs": {"hw:cpu_policy": "dedicated", "hw:cpu_thread_policy": "isolate"}
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("output", "table")
	if err := runShow(cmd, fake.ServiceClient(fakeServer), []string{"2"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"Parameter", "Value", "Properties", "hw:cpu_policy=dedicated", "hw:cpu_thread_policy=isolate"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q; got:\n%s", want, out.String())
		}
	}
	// The specs must be grouped under a single "Properties" Parameter row: the
	// line right after it is a continuation cell, not a new key/value pair.
	lines := strings.Split(out.String(), "\n")
	for i, l := range lines {
		if strings.Contains(l, "Properties") {
			if i+1 >= len(lines) || !strings.Contains(lines[i+1], "hw:cpu_thread_policy=isolate") {
				t.Fatalf("show output properties not a single newline-separated value; got:\n%s", out.String())
			}
		}
	}
}

// TestRunShowJSON verifies that the long-form output surfaces the spec data
// under the "properties" key (like openstackclient) and does not leak the empty
// gophercloud "extra_specs" field.
func TestRunShowJSON(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors/2", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "flavor": {
        "id": "2",
        "name": "m1.small",
        "vcpus": 1,
        "disk": 20,
        "ram": 2048,
        "swap": "",
        "os-flavor-access:is_public": true,
        "OS-FLV-EXT-DATA:ephemeral": 0,
        "description": "small"
    }
}`)
	})
	fakeServer.Mux.HandleFunc("/flavors/2/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "extra_specs": {"hw:cpu_policy": "dedicated", "hw:cpu_thread_policy": "isolate"}
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("output", "json")
	if err := runShow(cmd, fake.ServiceClient(fakeServer), []string{"2"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}

	if !strings.Contains(out.String(), `"properties"`) {
		t.Fatalf("show json output missing \"properties\" key; got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), `"hw:cpu_policy": "dedicated"`) {
		t.Fatalf("show json output missing property data; got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "extra_specs") {
		t.Fatalf("show json output leaked gophercloud extra_specs; got:\n%s", out.String())
	}
}

func TestRunShowNotFound(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/flavors/999", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"itemNotFound": {"message": "Flavor 999 could not be found", "code": 404}}`)
	})

	cmd, _ := testCommand(t)
	if err := runShow(cmd, fake.ServiceClient(fakeServer), []string{"999"}); err == nil {
		t.Fatal("runShow for a missing flavor succeeded; want an error")
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
