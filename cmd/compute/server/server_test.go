/*
Copyright © 2026 Kot1gruN
*/
package server

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

// testCommand builds a bare command with every vm subcommand flag registered
// and a captured stdout buffer. The response bodies follow the fixtures from
// github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers/testing.
func testCommand(t *testing.T) (*cobra.Command, *strings.Builder) {
	t.Helper()
	cmd := &cobra.Command{Use: "vm"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")

	// list flags (also "name" is shared with create/update)
	fs.String("changes-since", "", "")
	fs.String("name", "", "")
	fs.String("status", "", "")
	fs.String("image", "", "")
	fs.String("flavor", "", "")
	fs.String("ip", "", "")
	fs.String("ip6", "", "")
	fs.String("host", "", "")
	fs.Int("limit", 0, "")
	fs.String("marker", "", "")
	fs.String("tags", "", "")
	fs.String("tags-any", "", "")
	fs.String("not-tags", "", "")
	fs.String("not-tags-any", "", "")
	fs.String("availability-zone", "", "")
	fs.Bool("all-regions", false, "")

	// create flags
	fs.String("flavor-ref", "", "")
	fs.String("image-ref", "", "")
	fs.StringArray("net", nil, "")
	fs.StringArray("port", nil, "")
	fs.String("network", "", "")
	fs.StringArray("security-group", nil, "")
	fs.String("user-data-file", "", "")
	fs.StringToString("metadata", nil, "")
	fs.String("admin-pass", "", "")
	fs.Bool("config-drive", false, "")
	fs.Bool("config-drive-off", false, "")
	fs.String("disk-config", "", "")
	fs.Int("min-count", 0, "")
	fs.Int("max-count", 0, "")
	fs.StringArray("tag", nil, "")

	// update / delete flags
	fs.String("access-ipv4", "", "")
	fs.String("access-ipv6", "", "")
	fs.String("hostname", "", "")
	fs.Bool("force", false, "")
	fs.Bool("hard", false, "")

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunList(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "servers": [
        {"id": "ef079b0c-e610-4dfb-b1aa-b49f07ac48e5", "name": "herp", "status": "ACTIVE",
         "flavor": {"id": "1"}, "image": {"id": "img-1"}, "addresses": {},
         "key_name": null, "created": "2014-09-25T13:10:02Z"},
        {"id": "9e5476bd-a4ec-4653-93d6-72c93aa682ba", "name": "derp", "status": "ACTIVE",
         "flavor": {"id": "1"}, "image": {"id": "img-2"}, "addresses": {},
         "key_name": null, "created": "2014-09-25T13:04:41Z"}
    ]
}`)
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"herp", "derp"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunShow(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/servers/9e5476bd-a4ec-4653-93d6-72c93aa682ba", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "server": {
        "id": "9e5476bd-a4ec-4653-93d6-72c93aa682ba",
        "name": "derp",
        "status": "ACTIVE",
        "flavor": {"id": "1"},
        "image": {"id": "img-1"},
        "addresses": {"private": [{"addr": "10.0.0.31", "version": 4}]},
        "key_name": null,
        "created": "2014-09-25T13:04:41Z",
        "updated": "2014-09-25T13:04:49Z",
        "OS-EXT-STS:vm_state": "active"
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := runShow(cmd, fake.ServiceClient(fakeServer), []string{"9e5476bd-a4ec-4653-93d6-72c93aa682ba"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	if !strings.Contains(out.String(), "derp") {
		t.Fatalf("show output missing server; got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "10.0.0.31") {
		t.Fatalf("show output missing IP; got:\n%s", out.String())
	}
}

func TestRunCreate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{
    "server": {
        "name": "new-server",
        "imageRef": "f90f6034-2570-4974-8351-6b49732ef2eb",
        "flavorRef": "1"
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{
    "server": {
        "id": "9e5476bd-a4ec-4653-93d6-72c93aa682ba",
        "name": "new-server",
        "status": "BUILD",
        "flavor": {"id": "1"},
        "image": {"id": "img-1"},
        "addresses": {},
        "key_name": null,
        "created": "2014-09-25T13:04:41Z"
    }
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "new-server")
	_ = cmd.Flags().Set("flavor-ref", "1")
	_ = cmd.Flags().Set("image-ref", "f90f6034-2570-4974-8351-6b49732ef2eb")

	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "new-server") {
		t.Fatalf("create output missing server; got:\n%s", out.String())
	}
}
func TestRunCreateRequiresName(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate without --name succeeded; want an error")
	}
}

func TestRunCreateRequiresFlavorRef(t *testing.T) {
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("name", "x")
	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate without --flavor-ref succeeded; want an error")
	}
}

func TestRunCreateNetworkConflict(t *testing.T) {
	cmd, _ := testCommand(t)
	_ = cmd.Flags().Set("name", "x")
	_ = cmd.Flags().Set("flavor-ref", "1")
	_ = cmd.Flags().Set("network", "auto")
	_ = cmd.Flags().Set("net", "net-1")
	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate with --network and --net succeeded; want an error")
	}
}

func TestRunUpdate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/servers/1234asdf", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PUT")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{ "server": { "name": "new-name" } }`)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "server": {
        "id": "1234asdf",
        "name": "new-name",
        "status": "ACTIVE",
        "flavor": {"id": "1"},
        "image": {"id": "img-1"},
        "addresses": {},
        "key_name": null,
        "created": "2014-09-25T13:04:41Z"
    }
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "new-name")
	if err := runUpdate(cmd, fake.ServiceClient(fakeServer), []string{"1234asdf"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "new-name") {
		t.Fatalf("update output missing server; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOption(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runUpdate(cmd, nil, []string{"1234asdf"}); err == nil {
		t.Fatal("runUpdate without an option succeeded; want an error")
	}
}

func TestRunDelete(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/servers/asdfasdfasdf", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.WriteHeader(http.StatusNoContent)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, fake.ServiceClient(fakeServer), []string{"asdfasdfasdf"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "asdfasdfasdf") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}

func TestRunDeleteForce(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/servers/asdfasdfasdf/action", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{ "forceDelete": "" }`)
		w.WriteHeader(http.StatusAccepted)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("force", "true")
	if err := runDelete(cmd, fake.ServiceClient(fakeServer), []string{"asdfasdfasdf"}); err != nil {
		t.Fatalf("runDelete --force: %v", err)
	}
	if !strings.Contains(out.String(), "asdfasdfasdf") {
		t.Fatalf("force-delete output missing confirmation; got:\n%s", out.String())
	}
}

// TestRunListTableOmitsDetailColumns verifies the table form of the list
// command keeps only the narrow columns, leaving the per-server detail
// (flavor/image ID and key name) out of the terminal-wide grid.
func TestRunListTableOmitsDetailColumns(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "servers": [
        {"id": "ef079b0c-e610-4dfb-b1aa-b49f07ac48e5", "name": "herp", "status": "ACTIVE",
         "flavor": {"id": "1"}, "image": {"id": "img-1"}, "key_name": "my-key",
         "addresses": {"private": [{"addr": "10.0.0.32", "version": 4}]},
         "created": "2014-09-25T13:10:02Z"}
    ]
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("output", "table")
	if err := runList(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, removed := range []string{"Flavor ID", "Image ID", "Key Name"} {
		if strings.Contains(out.String(), removed) {
			t.Fatalf("list table still contains %q; got:\n%s", removed, out.String())
		}
	}
	for _, kept := range []string{"ID", "Name", "Status"} {
		if !strings.Contains(out.String(), kept) {
			t.Fatalf("list table missing %q; got:\n%s", kept, out.String())
		}
	}
}

// setupAction registers the server /action endpoint responding to the given
// expected JSON payload with a 202 Accepted status.
func setupAction(fakeServer th.FakeServer, t *testing.T, payload string) {
	fakeServer.Mux.HandleFunc("/servers/1234asdf/action", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, payload)
		w.WriteHeader(http.StatusAccepted)
	})
}

func TestRunPause(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)
	setupAction(fakeServer, t, `{ "pause": null }`)

	cmd, out := testCommand(t)
	if err := runPause(cmd, fake.ServiceClient(fakeServer), []string{"1234asdf"}); err != nil {
		t.Fatalf("runPause: %v", err)
	}
	if !strings.Contains(out.String(), "paused") {
		t.Fatalf("pause output missing confirmation; got:\n%s", out.String())
	}
}

func TestRunUnpause(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)
	setupAction(fakeServer, t, `{ "unpause": null }`)

	cmd, out := testCommand(t)
	if err := runUnpause(cmd, fake.ServiceClient(fakeServer), []string{"1234asdf"}); err != nil {
		t.Fatalf("runUnpause: %v", err)
	}
	if !strings.Contains(out.String(), "unpaused") {
		t.Fatalf("unpause output missing confirmation; got:\n%s", out.String())
	}
}

func TestRunRebootSoft(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)
	setupAction(fakeServer, t, `{ "reboot": { "type": "SOFT" } }`)

	cmd, out := testCommand(t)
	if err := runReboot(cmd, fake.ServiceClient(fakeServer), []string{"1234asdf"}); err != nil {
		t.Fatalf("runReboot: %v", err)
	}
	if !strings.Contains(out.String(), "rebooted") {
		t.Fatalf("reboot output missing confirmation; got:\n%s", out.String())
	}
}

func TestRunRebootHard(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)
	setupAction(fakeServer, t, `{ "reboot": { "type": "HARD" } }`)

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("hard", "true")
	if err := runReboot(cmd, fake.ServiceClient(fakeServer), []string{"1234asdf"}); err != nil {
		t.Fatalf("runReboot --hard: %v", err)
	}
	if !strings.Contains(out.String(), "HARD") {
		t.Fatalf("hard reboot output missing method; got:\n%s", out.String())
	}
}

func TestRunStop(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)
	setupAction(fakeServer, t, `{ "os-stop": null }`)

	cmd, out := testCommand(t)
	if err := runStop(cmd, fake.ServiceClient(fakeServer), []string{"1234asdf"}); err != nil {
		t.Fatalf("runStop: %v", err)
	}
	if !strings.Contains(out.String(), "stopped") {
		t.Fatalf("stop output missing confirmation; got:\n%s", out.String())
	}
}

func TestRunStart(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)
	setupAction(fakeServer, t, `{ "os-start": null }`)

	cmd, out := testCommand(t)
	if err := runStart(cmd, fake.ServiceClient(fakeServer), []string{"1234asdf"}); err != nil {
		t.Fatalf("runStart: %v", err)
	}
	if !strings.Contains(out.String(), "started") {
		t.Fatalf("start output missing confirmation; got:\n%s", out.String())
	}
}
