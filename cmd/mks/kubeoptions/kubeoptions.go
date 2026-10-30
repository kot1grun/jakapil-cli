/*
Copyright © 2026 Kot1gruN
*/
package kubeoptions

import (
	"context"

	mkskubeoptions "github.com/selectel/mks-go/pkg/v1/kubeoptions"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// NewKubeOptionsCommand returns the "kubeoptions" subcommand.
// Without arguments it shows the command help; "list" lists feature gates
// (or admission controllers with "--admission-controllers").
func NewKubeOptionsCommand() *cobra.Command {
	var admissionControllers bool

	cmd := &cobra.Command{
		Use:   "kubeoptions",
		Short: "Show available Kubernetes feature gates and admission controllers",
		Long: `Show available Kubernetes options grouped by Kubernetes version.

Subcommands:
  list  List feature gates by Kubernetes version (use "--admission-controllers"
        to list admission controllers instead)

Flags:
  --admission-controllers  List admission controllers instead of feature gates

Examples:
  jakapil-cli mks kubeoptions list
  jakapil-cli mks kubeoptions list --admission-controllers
  jakapil-cli mks kubeoptions --admission-controllers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !admissionControllers {
				return cmd.Help()
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			available, _, err := mkskubeoptions.ListAdmissionControllers(ctx, mksClient)
			if err != nil {
				return err
			}
			return render(cmd, available)
		},
	}

	cmd.Flags().BoolVar(&admissionControllers, "admission-controllers", false,
		"list available admission controllers instead of feature gates")
	cmd.AddCommand(newListCommand())

	return cmd
}

// newListCommand returns the "kubeoptions list" subcommand.
func newListCommand() *cobra.Command {
	var admissionControllers bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List feature gates by Kubernetes version",
		Long: `List available Kubernetes feature gates by Kubernetes version,
or admission controllers with "--admission-controllers".`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			var available []*mkskubeoptions.View
			if admissionControllers {
				available, _, err = mkskubeoptions.ListAdmissionControllers(ctx, mksClient)
			} else {
				available, _, err = mkskubeoptions.ListFeatureGates(ctx, mksClient)
			}
			if err != nil {
				return err
			}

			return render(cmd, available)
		},
	}

	cmd.Flags().BoolVar(&admissionControllers, "admission-controllers", false,
		"list available admission controllers instead of feature gates")

	return cmd
}
