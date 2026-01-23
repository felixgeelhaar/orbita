package project

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/projects/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var milestoneCmd = &cobra.Command{
	Use:   "milestone",
	Short: "Manage project milestones",
	Long:  `Add, update, and delete milestones within a project.`,
}

var (
	milestoneDescription string
	milestoneDueDate     string
	milestoneName        string
)

var addMilestoneCmd = &cobra.Command{
	Use:   "add [project-id] [name]",
	Short: "Add a milestone to a project",
	Long: `Add a new milestone to a project.

Examples:
  orbita project milestone add abc123 "Alpha Release" --due 2024-03-15
  orbita project milestone add abc123 "MVP Complete" -d "First working version" --due 2024-04-30`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.AddMilestoneHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		if milestoneDueDate == "" {
			return fmt.Errorf("due date is required (use --due YYYY-MM-DD)")
		}

		dueDate, err := time.Parse("2006-01-02", milestoneDueDate)
		if err != nil {
			return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
		}

		addCmd := commands.AddMilestoneCommand{
			ProjectID:   projectID,
			UserID:      app.CurrentUserID,
			Name:        args[1],
			Description: milestoneDescription,
			DueDate:     dueDate,
		}

		ctx := cmd.Context()
		result, err := app.AddMilestoneHandler.Handle(ctx, addCmd)
		if err != nil {
			return fmt.Errorf("failed to add milestone: %w", err)
		}

		fmt.Printf("Milestone added: %s\n", result.MilestoneID)
		fmt.Printf("  name: %s\n", args[1])
		fmt.Printf("  due: %s\n", milestoneDueDate)

		return nil
	},
}

var updateMilestoneCmd = &cobra.Command{
	Use:   "update [project-id] [milestone-id]",
	Short: "Update a milestone",
	Long: `Update milestone properties.

Examples:
  orbita project milestone update abc123 def456 --name "New Name"
  orbita project milestone update abc123 def456 --due 2024-04-30`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.UpdateMilestoneHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		milestoneID, err := uuid.Parse(args[1])
		if err != nil {
			return fmt.Errorf("invalid milestone ID: %w", err)
		}

		updateCmd := commands.UpdateMilestoneCommand{
			MilestoneID: milestoneID,
			ProjectID:   projectID,
			UserID:      app.CurrentUserID,
		}

		if milestoneName != "" {
			updateCmd.Name = &milestoneName
		}
		if milestoneDescription != "" {
			updateCmd.Description = &milestoneDescription
		}
		if milestoneDueDate != "" {
			parsed, err := time.Parse("2006-01-02", milestoneDueDate)
			if err != nil {
				return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
			}
			updateCmd.DueDate = &parsed
		}

		ctx := cmd.Context()
		if err := app.UpdateMilestoneHandler.Handle(ctx, updateCmd); err != nil {
			return fmt.Errorf("failed to update milestone: %w", err)
		}

		fmt.Println("Milestone updated successfully.")
		return nil
	},
}

var deleteMilestoneCmd = &cobra.Command{
	Use:   "delete [project-id] [milestone-id]",
	Short: "Delete a milestone",
	Long: `Delete a milestone from a project.

Examples:
  orbita project milestone delete abc123 def456`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.DeleteMilestoneHandler == nil {
			return fmt.Errorf("application not initialized - database connection required")
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		milestoneID, err := uuid.Parse(args[1])
		if err != nil {
			return fmt.Errorf("invalid milestone ID: %w", err)
		}

		deleteCmd := commands.DeleteMilestoneCommand{
			MilestoneID: milestoneID,
			ProjectID:   projectID,
			UserID:      app.CurrentUserID,
		}

		ctx := cmd.Context()
		if err := app.DeleteMilestoneHandler.Handle(ctx, deleteCmd); err != nil {
			return fmt.Errorf("failed to delete milestone: %w", err)
		}

		fmt.Println("Milestone deleted successfully.")
		return nil
	},
}

func init() {
	// Add milestone command
	addMilestoneCmd.Flags().StringVarP(&milestoneDescription, "description", "d", "", "milestone description")
	addMilestoneCmd.Flags().StringVar(&milestoneDueDate, "due", "", "due date (YYYY-MM-DD) - required")

	// Update milestone command
	updateMilestoneCmd.Flags().StringVar(&milestoneName, "name", "", "new milestone name")
	updateMilestoneCmd.Flags().StringVarP(&milestoneDescription, "description", "d", "", "new milestone description")
	updateMilestoneCmd.Flags().StringVar(&milestoneDueDate, "due", "", "new due date (YYYY-MM-DD)")

	// Add subcommands
	milestoneCmd.AddCommand(addMilestoneCmd)
	milestoneCmd.AddCommand(updateMilestoneCmd)
	milestoneCmd.AddCommand(deleteMilestoneCmd)
}
