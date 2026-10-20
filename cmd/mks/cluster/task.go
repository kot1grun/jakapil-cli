/*
Copyright © 2026 Kot1gruN
*/
package cluster

import (
	"context"

	mkstask "github.com/selectel/mks-go/pkg/v1/task"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
	"jakapil-cli/cmd/mks/output"
)

// newTaskCommand returns the "cluster task" subcommand group.
// Without arguments it shows the command help; "list" lists all tasks of a
// cluster; "show" returns a single task by its identifier.
func newTaskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage cluster tasks (list, show)",
		Long: `Inspect tasks of a Kubernetes cluster.

Subcommands:
  list  List all tasks of a cluster
  show  Show details of a single task by its identifier`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newTaskListCommand(), newTaskShowCommand())

	return cmd
}

// newTaskListCommand returns the "cluster task list" subcommand.
func newTaskListCommand() *cobra.Command {
	var clusterID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks of a cluster",
		Long:  "List all tasks of a Kubernetes cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return cmd.Help()
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			tasks, _, err := mkstask.List(ctx, mksClient, clusterID)
			if err != nil {
				return err
			}
			return output.Print(cmd, tasks)
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")

	return cmd
}

// newTaskShowCommand returns the "cluster task show" subcommand.
func newTaskShowCommand() *cobra.Command {
	var (
		clusterID string
		taskID    string
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show a single task",
		Long:  "Show details of a single cluster task by its identifier.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return cmd.Help()
			}
			if taskID == "" {
				return cmd.Help()
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			task, _, err := mkstask.Get(ctx, mksClient, clusterID, taskID)
			if err != nil {
				return err
			}
			return output.Print(cmd, task)
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVarP(&taskID, "id", "i", "", "task ID to show details for")

	return cmd
}
