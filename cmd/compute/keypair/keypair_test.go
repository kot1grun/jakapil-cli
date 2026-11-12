/*
Copyright © 2026 Kot1gruN
*/
package keypair

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

// testCommand builds a bare command with every keypair subcommand flag
// registered and a captured stdout buffer. The response bodies follow the
// fixtures from
// github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs/testing.
func testCommand(t *testing.T) (*cobra.Command, *strings.Builder) {
	t.Helper()
	cmd := &cobra.Command{Use: "keypair"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.Bool("all-regions", false, "")
	fs.String("name", "", "")
	fs.String("public-key", "", "")
	fs.String("type", "", "")

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunList(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/os-keypairs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "keypairs": [
        {"keypair": {
            "fingerprint": "15:b0:f8:b3:f9:48:63:71:cf:7b:5b:38:6d:44:2d:4a",
            "name": "firstkey",
            "type": "ssh",
            "public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQAB generated",
            "user_id": "fake"
        }},
        {"keypair": {
            "fingerprint": "35:9d:d0:c3:4a:80:d3:d8:86:f1:ca:f7:df:c4:f9:d8",
            "name": "secondkey",
            "type": "ssh",
            "public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQAB2 generated",
            "user_id": "fake"
        }}
    ]
}`)
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"firstkey", "secondkey"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunListTableOmitsPublicKey(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/os-keypairs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "keypairs": [
        {"keypair": {
            "fingerprint": "15:b0:f8:b3:f9:48:63:71:cf:7b:5b:38:6d:44:2d:4a",
            "name": "firstkey",
            "public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQAB generated",
            "user_id": "fake"
        }}
    ]
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("output", "table")
	if err := runList(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if strings.Contains(out.String(), "Public Key") {
		t.Fatalf("list table leaked the public key column; got:\n%s", out.String())
	}
	for _, kept := range []string{"Name", "Fingerprint"} {
		if !strings.Contains(out.String(), kept) {
			t.Fatalf("list table missing %q; got:\n%s", kept, out.String())
		}
	}
}

func TestRunShow(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/os-keypairs/firstkey", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
    "keypair": {
        "public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQAB generated",
        "name": "firstkey",
        "type": "ssh",
        "fingerprint": "15:b0:f8:b3:f9:48:63:71:cf:7b:5b:38:6d:44:2d:4a",
        "user_id": "fake"
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := runShow(cmd, fake.ServiceClient(fakeServer), []string{"firstkey"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	if !strings.Contains(out.String(), "firstkey") || !strings.Contains(out.String(), "15:b0:f8:b3") {
		t.Fatalf("show output missing expected key data; got:\n%s", out.String())
	}
}

func TestRunCreate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/os-keypairs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, `{ "keypair": { "name": "createdkey" } }`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "keypair": {
        "fingerprint": "35:9d:d0:c3:4a:80:d3:d8:86:f1:ca:f7:df:c4:f9:d8",
        "name": "createdkey",
        "private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIICXA==",
        "public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQAB generated",
        "user_id": "fake"
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "createdkey"); err != nil {
		t.Fatal(err)
	}
	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "createdkey") {
		t.Fatalf("create output missing keypair name; got:\n%s", out.String())
	}
	// The freshly generated private key must be surfaced to the user once.
	if !strings.Contains(out.String(), "PRIVATE KEY") {
		t.Fatalf("create output missing generated private key; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresName(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate without --name succeeded; want an error")
	}
}

func TestRunCreateImport(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	pub := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQAB generated-import"
	reqBody := `{ "keypair": { "name": "importedkey", "public_key": "` + pub + `" } }`
	fakeServer.Mux.HandleFunc("/os-keypairs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestJSONRequest(t, r, reqBody)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "keypair": {
        "fingerprint": "1e:2c:9b:56:79:4b:45:77:f9:ca:7a:98:2c:b0:d5:3c",
        "name": "importedkey",
        "public_key": "`+pub+`",
        "user_id": "fake"
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "importedkey"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("public-key", pub); err != nil {
		t.Fatal(err)
	}
	if err := runCreate(cmd, fake.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate import: %v", err)
	}
	if !strings.Contains(out.String(), "importedkey") {
		t.Fatalf("import output missing keypair name; got:\n%s", out.String())
	}
}

func TestRunDelete(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/os-keypairs/deletedkey", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.WriteHeader(http.StatusAccepted)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, fake.ServiceClient(fakeServer), []string{"deletedkey"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "keypair deletedkey deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
