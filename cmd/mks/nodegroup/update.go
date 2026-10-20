/*
Copyright © 2026 Kot1gruN
*/
package nodegroup

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// newUpdateCommand returns the "nodegroup update" subcommand that updates a
// nodegroup of a cluster: labels, taints and autoscaling settings.
//
// The autoscale pointers (*bool/*int) are only sent when their corresponding
// flag is actually set, so an unset flag leaves the field untouched.
func newUpdateCommand() *cobra.Command {
	var (
		clusterID string
		groupID   string

		labels []string
		taints []string

		autoscale    string
		autoscaleMin int
		autoscaleMax int
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a nodegroup",
		Long: `Update a nodegroup of a Kubernetes cluster (labels, taints and
autoscaling settings).

The operation changes the state of the cluster.

Examples:
  jakapil-cli mks nodegroup update --cluster-id <cid> --id <nid> \
    --label env=prod --label team=platform

  jakapil-cli mks nodegroup update --cluster-id <cid> --id <nid> \
    --autoscale enable --autoscale-min-nodes 2 --autoscale-max-nodes 10

  jakapil-cli mks nodegroup update --cluster-id <cid> --id <nid> \
    --taint dedicated=gpu:NoSchedule`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}
			if groupID == "" {
				return cmd.Help()
			}

			opts, err := buildUpdateOpts(cmd, labels, taints, autoscale, autoscaleMin, autoscaleMax)
			if err != nil {
				return err
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			res, err := mksnodegroup.Update(ctx, mksClient, clusterID, groupID, opts)
			if err != nil {
				return err
			}
			// Update is async and commonly returns 200, 202 Accepted or 204 No
			// Content; treat any 2xx status as success.
			if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("unexpected HTTP status %d while updating nodegroup %q", res.StatusCode, groupID)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "nodegroup %s updated\n", groupID)
			return err
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVarP(&groupID, "id", "i", "", "nodegroup ID to update")

	cmd.Flags().StringArrayVar(&labels, "label", nil, "label key=value to apply (repeatable)")
	cmd.Flags().StringArrayVar(&taints, "taint", nil, "taint key[=value]:effect to apply (repeatable)")
	cmd.Flags().StringVar(&autoscale, "autoscale", "", "enable or disable autoscaling: enable, disable")
	cmd.Flags().IntVar(&autoscaleMin, "autoscale-min-nodes", 0, "minimum number of worker nodes (only with autoscale)")
	cmd.Flags().IntVar(&autoscaleMax, "autoscale-max-nodes", 0, "maximum number of worker nodes (only with autoscale)")

	return cmd
}

// buildUpdateOpts converts the parsed flags into nodegroup UpdateOpts.
// It returns an error if no update option was provided or a value is invalid.
func buildUpdateOpts(cmd *cobra.Command, labelFlags, taintFlags []string, autoscale string, autoscaleMin, autoscaleMax int) (*mksnodegroup.UpdateOpts, error) {
	labels, err := parseLabels(labelFlags)
	if err != nil {
		return nil, err
	}
	taints, err := parseTaints(taintFlags)
	if err != nil {
		return nil, err
	}

	opts := &mksnodegroup.UpdateOpts{}

	if len(labels) > 0 {
		opts.Labels = labels
	}
	if len(taints) > 0 {
		opts.Taints = taints
	}

	switch autoscale {
	case "":
	case "enable", "true":
		opts.EnableAutoscale = boolPtr(true)
	case "disable", "false":
		opts.EnableAutoscale = boolPtr(false)
	default:
		return nil, fmt.Errorf("invalid --autoscale %q: use enable or disable", autoscale)
	}

	if cmd.Flags().Changed("autoscale-min-nodes") {
		v := autoscaleMin
		opts.AutoscaleMinNodes = &v
	}
	if cmd.Flags().Changed("autoscale-max-nodes") {
		v := autoscaleMax
		opts.AutoscaleMaxNodes = &v
	}

	if opts.Labels == nil && opts.Taints == nil && opts.EnableAutoscale == nil &&
		opts.AutoscaleMinNodes == nil && opts.AutoscaleMaxNodes == nil {
		return nil, fmt.Errorf("nothing to update: pass at least one of --label, --taint, --autoscale, --autoscale-min-nodes or --autoscale-max-nodes")
	}

	return opts, nil
}

// parseLabels converts a list of "key=value" flags into a labels map.
func parseLabels(flags []string) (map[string]string, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	labels := make(map[string]string, len(flags))
	for _, f := range flags {
		key, value, ok := strings.Cut(f, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid label %q: use key=value", f)
		}
		labels[key] = value
	}
	return labels, nil
}

// parseTaints converts a list of "key[=value]:effect" flags into taints.
func parseTaints(flags []string) ([]mksnodegroup.Taint, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	taints := make([]mksnodegroup.Taint, 0, len(flags))
	for _, f := range flags {
		keyPart, effect, ok := strings.Cut(f, ":")
		if !ok || keyPart == "" || effect == "" {
			return nil, fmt.Errorf("invalid taint %q: use key[=value]:effect", f)
		}

		var taint mksnodegroup.Taint
		if key, value, hasValue := strings.Cut(keyPart, "="); hasValue {
			taint.Key = key
			taint.Value = value
		} else {
			taint.Key = keyPart
		}

		effect = strings.ToUpper(strings.TrimSpace(effect))
		if !isValidTaintEffect(effect) {
			return nil, fmt.Errorf("invalid taint effect %q: use NoSchedule, PreferNoSchedule or NoExecute", effect)
		}
		taint.Effect = mksnodegroup.TaintEffect(effect)

		taints = append(taints, taint)
	}
	return taints, nil
}

func isValidTaintEffect(v string) bool {
	switch v {
	case "NOSCHEDULE", "PREFERNOSCHEDULE", "NOEXECUTE":
		return true
	default:
		return false
	}
}

func boolPtr(b bool) *bool { return &b }
