/*
Copyright © 2026 Kot1gruN
*/
package kubeoptions

import (
	"context"
	"fmt"

	mkskubeoptions "github.com/selectel/mks-go/pkg/v1/kubeoptions"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// NewKubeOptionsCommand returns the "kubeoptions" subcommand.
// By default it lists available feature gates by Kubernetes version;
// with "--admission-controllers" it lists available admission controllers.
func NewKubeOptionsCommand() *cobra.Command {
	var admissionControllers bool

	cmd := &cobra.Command{
		Use:   "kubeoptions",
		Short: "List available Kubernetes feature gates and admission controllers",
		Long: `List available Kubernetes feature gates (by default) or admission
controllers (with "--admission-controllers") grouped by Kubernetes version.

Examples:
  jakapil-cli mks kubeoptions
  jakapil-cli mks kubeoptions --admission-controllers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			if admissionControllers {
				available, _, err := mkskubeoptions.ListAdmissionControllers(ctx, mksClient)
				if err != nil {
					return err
				}
				for _, v := range available {
					fmt.Printf("%s: %v\n", v.KubeVersion, v.Names)
				}
				return nil
			}

			available, _, err := mkskubeoptions.ListFeatureGates(ctx, mksClient)
			if err != nil {
				return err
			}
			for _, v := range available {
				fmt.Printf("%s: %v\n", v.KubeVersion, v.Names)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&admissionControllers, "admission-controllers", false,
		"list available admission controllers instead of feature gates")

	return cmd
}
