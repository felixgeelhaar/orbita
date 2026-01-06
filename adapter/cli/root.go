package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	logger  *slog.Logger
)

type commandContext struct {
	correlationID uuid.UUID
	startedAt     time.Time
}

type commandContextKey struct{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "orbita",
	Short: "Orbita - Adaptive Productivity OS",
	Long: `Orbita is a CLI-first adaptive productivity operating system
that orchestrates tasks, calendars, habits, and meetings.

	It replaces manual planning with autonomous orchestration,
	adapting continuously as reality changes.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if logger == nil {
			logger = slog.Default()
		}
		ctx := cmd.Context()
		info := commandContext{
			correlationID: uuid.New(),
			startedAt:     time.Now(),
		}
		cmd.SetContext(context.WithValue(ctx, commandContextKey{}, info))
		logger.Info("command start",
			"command", cmd.CommandPath(),
			"correlation_id", info.correlationID.String(),
		)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logger == nil {
			logger = slog.Default()
		}
		info, ok := cmd.Context().Value(commandContextKey{}).(commandContext)
		if !ok {
			return
		}
		logger.Info("command end",
			"command", cmd.CommandPath(),
			"correlation_id", info.correlationID.String(),
			"duration_ms", time.Since(info.startedAt).Milliseconds(),
		)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// AddCommand adds a command to the root command.
func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

// SetLogger sets the CLI logger.
func SetLogger(l *slog.Logger) {
	logger = l
}
