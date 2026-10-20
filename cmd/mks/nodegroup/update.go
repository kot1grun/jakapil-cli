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
// Since the MKS update endpoint replaces the whole label set with the one sent
// in a request, this command fetches the current labels first (via
// nodegroup.Get) and merges the requested additions and removals, so existing
// labels are not wiped out.
//
// The autoscale pointers (*bool/*int) are only sent when their corresponding
// flag is actually set, so an unset flag leaves the field untouched.
func newUpdateCommand() *cobra.Command {
	var (
		clusterID string
		groupID   string

		labels       []string
		removeLabels []string
		clearLabels  bool

		taints       []string
		removeTaints []string
		clearTaints  bool

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

Labels are updated incrementally: "--label" adds or changes a single label
without removing the existing ones, "--remove-label" deletes a single label
by its key, and "--clear-labels" removes all labels. Similarly, taints are
updated incrementally: "--taint" adds or changes a taint, "--remove-taint"
deletes one by its key, and "--clear-taints" removes all taints. The
"--label", "--remove-label", "--taint" and "--remove-taint" flags can be
repeated.

Examples:
  jakapil-cli mks nodegroup update --cluster-id <cid> --id <nid> \
    --label env=prod --label team=platform

  jakapil-cli mks nodegroup update --cluster-id <cid> --id <nid> \
    --remove-label env --remove-label team

  jakapil-cli mks nodegroup update --cluster-id <cid> --id <nid> --clear-labels

  jakapil-cli mks nodegroup update --cluster-id <cid> --id <nid> \
    --taint dedicated=gpu:NoSchedule --remove-taint legacy

  jakapil-cli mks nodegroup update --cluster-id <cid> --id <nid> --clear-taints

  jakapil-cli mks nodegroup update --cluster-id <cid> --id <nid> \
    --autoscale enable --autoscale-min-nodes 2 --autoscale-max-nodes 10
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}
			if groupID == "" {
				return cmd.Help()
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			// Fetch the current labels and taints so that adding or removing a single
			// entry does not wipe out the others (the API replaces the whole set).
			var currentLabels map[string]string
			var currentTaints []mksnodegroup.Taint
			if len(labels) > 0 || len(removeLabels) > 0 || len(taints) > 0 || len(removeTaints) > 0 {
				ng, _, err := mksnodegroup.Get(ctx, mksClient, clusterID, groupID)
				if err != nil {
					return err
				}
				currentLabels = ng.Labels
				currentTaints = ng.Taints
			}

			opts, err := buildUpdateOpts(cmd, currentLabels, currentTaints, labels, removeLabels, clearLabels,
				taints, removeTaints, clearTaints, autoscale, autoscaleMin, autoscaleMax)
			if err != nil {
				return err
			}

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

	cmd.Flags().StringArrayVar(&labels, "label", nil, "label key=value to add or change (repeatable)")
	cmd.Flags().StringArrayVar(&removeLabels, "remove-label", nil, "label key to remove (repeatable)")
	cmd.Flags().BoolVar(&clearLabels, "clear-labels", false, "remove all labels")
	cmd.Flags().StringArrayVar(&taints, "taint", nil, "taint key[=value]:effect to add or change (repeatable)")
	cmd.Flags().StringArrayVar(&removeTaints, "remove-taint", nil, "taint key to remove (repeatable)")
	cmd.Flags().BoolVar(&clearTaints, "clear-taints", false, "remove all taints")
	cmd.Flags().StringVar(&autoscale, "autoscale", "", "enable or disable autoscaling: enable, disable")
	cmd.Flags().IntVar(&autoscaleMin, "autoscale-min-nodes", 0, "minimum number of worker nodes (only with autoscale)")
	cmd.Flags().IntVar(&autoscaleMax, "autoscale-max-nodes", 0, "maximum number of worker nodes (only with autoscale)")

	return cmd
}

// buildUpdateOpts converts the parsed flags into nodegroup UpdateOpts.
// currentLabels/currentTaints hold the sets already owned by the nodegroup;
// when they are non-nil the corresponding entries are merged incrementally
// (add + remove), and the *clear flags reset the whole set first. Explicit
// additions are used only when the current state was not fetched. It returns
// an error if no update option was provided or a value is invalid.
func buildUpdateOpts(cmd *cobra.Command, currentLabels map[string]string, currentTaints []mksnodegroup.Taint,
	labelFlags, removeLabelFlags []string, clearLabels bool,
	taintFlags, removeTaintFlags []string, clearTaints bool,
	autoscale string, autoscaleMin, autoscaleMax int) (*mksnodegroup.UpdateOpts, error) {

	labelAdds, err := parseLabels(labelFlags)
	if err != nil {
		return nil, err
	}
	labelRemovals, err := parseLabelKeys(removeLabelFlags)
	if err != nil {
		return nil, err
	}
	taintAdds, err := parseTaints(taintFlags)
	if err != nil {
		return nil, err
	}
	taintRemovals, err := parseTaintKeys(removeTaintFlags)
	if err != nil {
		return nil, err
	}

	opts := &mksnodegroup.UpdateOpts{}

	// Labels: "clear" resets to an empty set; otherwise, when the nodegroup
	// labels were fetched, merge incrementally so unrelated entries survive.
	if clearLabels {
		opts.Labels = applyLabels(map[string]string{}, labelAdds, labelRemovals)
	} else if currentLabels != nil {
		opts.Labels = applyLabels(currentLabels, labelAdds, labelRemovals)
	} else if len(labelAdds) > 0 {
		opts.Labels = labelAdds
	}

	// Taints: the same incremental semantics as labels.
	if clearTaints {
		opts.Taints = applyTaints(nil, taintAdds, taintRemovals, true)
	} else if currentTaints != nil {
		opts.Taints = applyTaints(currentTaints, taintAdds, taintRemovals, false)
	} else if len(taintAdds) > 0 {
		opts.Taints = taintAdds
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

	hasLabel := clearLabels || len(labelFlags) > 0 || len(removeLabelFlags) > 0
	hasTaint := clearTaints || len(taintFlags) > 0 || len(removeTaintFlags) > 0
	hasAuto := autoscale != "" || cmd.Flags().Changed("autoscale-min-nodes") || cmd.Flags().Changed("autoscale-max-nodes")
	if !hasLabel && !hasTaint && !hasAuto {
		return nil, fmt.Errorf("nothing to update: pass at least one of --label, --remove-label, --clear-labels, --taint, --remove-taint, --clear-taints, --autoscale, --autoscale-min-nodes or --autoscale-max-nodes")
	}

	return opts, nil
}

// applyLabels merges additions into the current label set and then removes the
// requested keys, producing the full set to send to the API.
func applyLabels(current, adds map[string]string, removals []string) map[string]string {
	merged := make(map[string]string, len(current)+len(adds))
	for k, v := range current {
		merged[k] = v
	}
	for k, v := range adds {
		merged[k] = v
	}
	for _, k := range removals {
		delete(merged, k)
	}
	return merged
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

// parseLabelKeys converts a list of label keys to remove into a slice.
func parseLabelKeys(flags []string) ([]string, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(flags))
	for _, f := range flags {
		if f == "" {
			return nil, fmt.Errorf("invalid label key: must not be empty")
		}
		keys = append(keys, f)
	}
	return keys, nil
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

		effect = strings.TrimSpace(effect)
		canonical, err := canonicalTaintEffect(effect)
		if err != nil {
			return nil, err
		}
		taint.Effect = mksnodegroup.TaintEffect(canonical)

		taints = append(taints, taint)
	}
	return taints, nil
}

// canonicalTaintEffect maps a user-provided effect (accepted in any case and
// with or without hyphens, e.g. "prefer-no-schedule") to the canonical value
// the MKS API expects: NoSchedule, PreferNoSchedule or NoExecute.
func canonicalTaintEffect(effect string) (string, error) {
	switch {
	case strings.EqualFold(effect, "NoSchedule") || strings.EqualFold(effect, "No-Schedule"):
		return "NoSchedule", nil
	case strings.EqualFold(effect, "NoExecute") || strings.EqualFold(effect, "No-Execute"):
		return "NoExecute", nil
	case strings.EqualFold(effect, "PreferNoSchedule") || strings.EqualFold(effect, "Prefer-No-Schedule"):
		return "PreferNoSchedule", nil
	default:
		return "", fmt.Errorf("invalid taint effect %q: use NoSchedule, PreferNoSchedule or NoExecute", effect)
	}
}

// parseTaintKeys converts a list of taint keys to remove into a slice. Each
// flag may be a plain key ("gpu") or a full taint ("gpu:NoSchedule"); only the
// key part is kept and used for removal.
func parseTaintKeys(flags []string) ([]string, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(flags))
	for _, f := range flags {
		key := strings.SplitN(f, ":", 2)[0]
		key = strings.SplitN(key, "=", 2)[0]
		if key == "" {
			return nil, fmt.Errorf("invalid taint key %q: must not be empty", f)
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// applyTaints merges additions into the current taint set (by key), removes the
// requested keys and returns the full set to send to the API. When clear is
// true the current set is discarded and only the additions remain. An empty
// result is returned as an empty (non-nil) slice so it serializes as "[]".
func applyTaints(current []mksnodegroup.Taint, adds []mksnodegroup.Taint, removals []string, clear bool) []mksnodegroup.Taint {
	merged := map[string]mksnodegroup.Taint{}
	if !clear {
		for _, t := range current {
			merged[t.Key] = t
		}
	}
	for _, k := range removals {
		delete(merged, k)
	}
	for _, t := range adds {
		merged[t.Key] = t
	}
	if len(merged) == 0 {
		return []mksnodegroup.Taint{}
	}
	out := make([]mksnodegroup.Taint, 0, len(merged))
	for _, t := range merged {
		out = append(out, t)
	}
	return out
}

func boolPtr(b bool) *bool { return &b }
