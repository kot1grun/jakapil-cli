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

func testTypeCommand(t *testing.T) (*cobra.Command, *strings.Builder) {
	t.Helper()
	cmd := &cobra.Command{Use: "type"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	// list flags
	fs.String("name", "", "")
	fs.String("description", "", "")
	fs.String("visibility", "", "")
	fs.Int("limit", 0, "")
	fs.String("sort", "", "")
	fs.Bool("all-regions", false, "")

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunTypeList(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/types", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "volume_types": [
        {
            "name": "SSD",
            "qos_specs_id": "qos-1",
            "extra_specs": {"volume_backend_name": "lvmdriver-1"},
            "is_public": true,
            "id": "6685584b-1eac-4da6-b5c3-555430cf68ff",
            "description": null
        },
        {
            "name": "SATA",
            "qos_specs_id": null,
            "extra_specs": {"volume_backend_name": "lvmdriver-1"},
            "is_public": true,
            "id": "8eb69a46-df97-4e41-9586-9a40a7533803",
            "description": null
        }
    ]
}`)
	})

	cmd, out := testTypeCommand(t)
	if err := runTypeList(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runTypeList: %v", err)
	}
	for _, want := range []string{"SSD", "SATA"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("type list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunTypeShow(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/types/d32019d3-bc6e-4319-9c1d-6722fc136a22", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "volume_type": {
        "id": "d32019d3-bc6e-4319-9c1d-6722fc136a22",
        "name": "vol-type-001",
        "description": "high performance",
        "os-volume-type-access:is_public": true,
        "extra_specs": {"volume_backend_name": "lvmdriver-1"},
        "is_public": true,
        "qos_specs_id": null
    }
}`)
	})

	cmd, out := testTypeCommand(t)
	_ = cmd.Flags().Set("output", "table")
	if err := runTypeShow(cmd, fake.ServiceClient(fakeServer), []string{"d32019d3-bc6e-4319-9c1d-6722fc136a22"}); err != nil {
		t.Fatalf("runTypeShow: %v", err)
	}
	for _, want := range []string{"vol-type-001", "high performance", "volume_backend_name=lvmdriver-1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("type show output missing %q; got:\n%s", want, out.String())
		}
	}
}
