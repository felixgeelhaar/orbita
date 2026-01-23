package project

import (
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/projects/application/commands"
	"github.com/felixgeelhaar/orbita/internal/projects/application/queries"
	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage project tasks",
	Long:  `Link, unlink, and list tasks associated with a project.`,
}

var (
	taskRole string
)

var linkTaskCmd = &cobra.Command{
	Use:   "link [project-id] [task-id]",
	Short: "Link a task to a project",
	Long: `Link an existing task to a project with an optional role.

Roles:
  blocker     - Task that blocks project progress
  dependency  - Task that project depends on
  deliverable - Task that represents a project deliverable
  subtask     - General subtask (default)

Examples:
  orbita project task link abc123 def456
  orbita project task link abc123 def456 --role blocker`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.LinkTaskHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		taskID, err := uuid.Parse(args[1])
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		// Validate role if provided
		role := domain.RoleSubtask
		if taskRole != "" {
			role = domain.TaskRole(taskRole)
			if !role.IsValid() {
				return fmt.Errorf("invalid role: %s (valid: blocker, dependency, deliverable, subtask)", taskRole)
			}
		}

		linkCmd := commands.LinkTaskCommand{
			ProjectID: projectID,
			TaskID:    taskID,
			UserID:    app.CurrentUserID,
			Role:      string(role),
		}

		ctx := cmd.Context()
		if err := app.LinkTaskHandler.Handle(ctx, linkCmd); err != nil {
			return fmt.Errorf("failed to link task: %w", err)
		}

		fmt.Printf("Task linked to project successfully.\n")
		fmt.Printf("  task: %s\n", taskID)
		fmt.Printf("  role: %s\n", role)
		return nil
	},
}

var unlinkTaskCmd = &cobra.Command{
	Use:   "unlink [project-id] [task-id]",
	Short: "Unlink a task from a project",
	Long: `Remove a task's association with a project.

Examples:
  orbita project task unlink abc123 def456`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.UnlinkTaskHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		taskID, err := uuid.Parse(args[1])
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		unlinkCmd := commands.UnlinkTaskCommand{
			ProjectID: projectID,
			TaskID:    taskID,
			UserID:    app.CurrentUserID,
		}

		ctx := cmd.Context()
		if err := app.UnlinkTaskHandler.Handle(ctx, unlinkCmd); err != nil {
			return fmt.Errorf("failed to unlink task: %w", err)
		}

		fmt.Println("Task unlinked from project successfully.")
		return nil
	},
}

var listTasksCmd = &cobra.Command{
	Use:   "list [project-id]",
	Short: "List tasks linked to a project",
	Long: `Display all tasks associated with a project.

Examples:
  orbita project task list abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.GetProjectHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		ctx := cmd.Context()
		query := queries.GetProjectQuery{
			ProjectID: projectID,
			UserID:    app.CurrentUserID,
		}
		project, err := app.GetProjectHandler.Handle(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		if len(project.Tasks) == 0 {
			fmt.Println("No tasks linked to this project.")
			return nil
		}

		fmt.Printf("Tasks in %s (%d):\n\n", project.Name, len(project.Tasks))

		// Group by role
		blockers := []string{}
		dependencies := []string{}
		deliverables := []string{}
		subtasks := []string{}

		for _, t := range project.Tasks {
			line := fmt.Sprintf("  %s", t.TaskID[:8])
			switch domain.TaskRole(t.Role) {
			case domain.RoleBlocker:
				blockers = append(blockers, line)
			case domain.RoleDependency:
				dependencies = append(dependencies, line)
			case domain.RoleDeliverable:
				deliverables = append(deliverables, line)
			default:
				subtasks = append(subtasks, line)
			}
		}

		if len(blockers) > 0 {
			fmt.Println("🚫 Blockers:")
			for _, b := range blockers {
				fmt.Println(b)
			}
			fmt.Println()
		}

		if len(dependencies) > 0 {
			fmt.Println("🔗 Dependencies:")
			for _, d := range dependencies {
				fmt.Println(d)
			}
			fmt.Println()
		}

		if len(deliverables) > 0 {
			fmt.Println("📦 Deliverables:")
			for _, d := range deliverables {
				fmt.Println(d)
			}
			fmt.Println()
		}

		if len(subtasks) > 0 {
			fmt.Println("📝 Subtasks:")
			for _, s := range subtasks {
				fmt.Println(s)
			}
		}

		return nil
	},
}

func init() {
	// Link task command
	linkTaskCmd.Flags().StringVar(&taskRole, "role", "", "task role (blocker, dependency, deliverable, subtask)")

	// Add subcommands
	taskCmd.AddCommand(linkTaskCmd)
	taskCmd.AddCommand(unlinkTaskCmd)
	taskCmd.AddCommand(listTasksCmd)
}
