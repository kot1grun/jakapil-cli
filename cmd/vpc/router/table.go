/*
Copyright © 2026 Kot1gruN
*/

package router

import (
	"fmt"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// routerHeader lists the compact columns shown for the table format.
var routerHeader = []string{"ID", "Name", "Status", "Admin Up", "Distributed", "GW Network", "Routes"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOfValues(v, routerRow); rows != nil {
		return routerHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func routerRow(r routers.Router) []string {
	return []string{
		r.ID,
		r.Name,
		r.Status,
		output.YesNo(r.AdminStateUp),
		output.YesNo(r.Distributed),
		r.GatewayInfo.NetworkID,
		strconv.Itoa(len(r.Routes)),
	}
}
