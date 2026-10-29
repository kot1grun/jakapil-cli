/*
Copyright © 2026 Kot1gruN
*/

// Package testutil provides a shared fake OpenStack networking v2 server used
// by the VPC command tests. It reuses gophercloud's own test infrastructure
// (testhelper.FakeServer via th.SetupHTTP) and the ServiceClient helper from
// openstack/networking/v2/common, so the tests match how production clients
// are wired up instead of hand-rolling an HTTP server.
package testutil

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Resource describes the subset of a networking v2 resource the fake server
// serves for a particular VPC command package (network, subnet or port).
type Resource struct {
	// Path is the collection path without the /v2.0/ prefix, e.g. "networks".
	Path string
	// ItemKey is the JSON key wrapping a single item, e.g. "network" or "subnet".
	ItemKey string
	// RequireField is the create body field that must be non-empty, e.g. "name"
	// for networks or "network_id" for subnets and ports.
	RequireField string

	// List are the records returned by GET on the collection.
	List []map[string]any
	// Get is the record returned by GET on a single item. Leave nil to
	// disable the single-item GET handler (used where "show" is unsupported).
	Get map[string]any
	// Updated is the record returned by PUT on a single item.
	Updated map[string]any
	// Created builds the create response. It receives the decoded item body
	// (so the response can echo back, say, the submitted name) and must return
	// the item to place under ItemKey.
	Created func(in map[string]any) map[string]any
}

// Neutron starts a fake networking v2 server serving a single resource.
func Neutron(t *testing.T, res Resource) th.FakeServer {
	t.Helper()
	fs := th.SetupHTTP()
	t.Cleanup(fs.Teardown)

	mux := fs.Mux
	mux.HandleFunc("/v2.0/"+res.Path, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{res.Path: res.List})
		case http.MethodPost:
			item, ok := decodeItem(w, r, res)
			if !ok {
				return
			}
			if item[res.RequireField] == "" {
				http.Error(w, res.RequireField+" required", http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{res.ItemKey: res.Created(item)})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2.0/"+res.Path+"/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if res.Get == nil {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{res.ItemKey: res.Get})
		case http.MethodPut:
			writeJSON(w, http.StatusOK, map[string]any{res.ItemKey: res.Updated})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return fs
}

// NewClient wires a networking v2 ServiceClient to the fake server using
// gophercloud's own test helper (openstack/networking/v2/common), which sets
// the base path to /v2.0/ exactly like production client construction does.
func NewClient(fs th.FakeServer) *gophercloud.ServiceClient {
	return fake.ServiceClient(fs)
}

// decodeItem extracts the item payload from a create request body shaped as
// {<ItemKey>: {...}}.
func decodeItem(w http.ResponseWriter, r *http.Request, res Resource) (map[string]any, bool) {
	var body map[string]map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, false
	}
	return body[res.ItemKey], true
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
