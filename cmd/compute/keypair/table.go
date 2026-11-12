/*
Copyright © 2026 Kot1gruN
*/

package keypair

import (
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// keypairHeader lists the compact columns shown for the table format of the
// list command. Plain JSON/YAML render the full record.
var keypairHeader = []string{"Name", "Fingerprint", "Type", "User ID"}

// keypairShowHeader is the two-column header used for a single-keypair table
// result (show, create). A keypair is rendered vertically, one parameter per
// row, so the (potentially long) public key fits without stretching a single
// wide row.
var keypairShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// toTable renders a slice of keypairs as the compact list table and a single
// keypair (backend returns *keypairs.KeyPair from Create/Get) as a two-column
// Parameter/Value table.
func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]keypairs.KeyPair); ok {
		rows := make([][]string, 0, len(list))
		for _, k := range list {
			rows = append(rows, keypairRow(k))
		}
		return keypairHeader, rows, nil
	}
	if single, ok := v.(*keypairs.KeyPair); ok {
		if single == nil {
			return keypairShowHeader, [][]string{}, nil
		}
		return keypairShowHeader, keypairShowRows(*single), nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func keypairRow(k keypairs.KeyPair) []string {
	return []string{
		k.Name,
		k.Fingerprint,
		k.Type,
		k.UserID,
	}
}

// keypairShowRows renders a single keypair as vertical Parameter/Value rows.
// The private key is included only when the API returned it, which happens
// exactly on a fresh create (the service keeps only the public form of a
// generated keypair).
func keypairShowRows(k keypairs.KeyPair) [][]string {
	rows := [][]string{
		{"Name", k.Name},
		{"Fingerprint", k.Fingerprint},
		{"Type", k.Type},
		{"User ID", k.UserID},
		{"Public Key", k.PublicKey},
	}
	if k.PrivateKey != "" {
		rows = append(rows, []string{"Private Key", k.PrivateKey})
	}
	return rows
}
