package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		file        string
		sortBy      string
		keepSecrets bool
	)

	cmd := &cobra.Command{
		Use:           "zsh-history-tidy",
		Short:         "Back up, deduplicate, and sort a zsh extended history file in place",
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			backupDir, err := defaultBackupDir()
			if err != nil {
				return err
			}
			stats, err := Run(Options{
				InputPath:   file,
				BackupDir:   backupDir,
				Sort:        SortMode(sortBy),
				KeepSecrets: keepSecrets,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(),
				"read=%d skipped=%d redacted=%d kept=%d backup=%s\n",
				stats.Read, stats.Skipped, stats.Redacted, stats.Kept, stats.BackupPath)
			fmt.Fprintln(cmd.OutOrStdout(), file)
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", defaultFile(), "zsh history file to tidy in place")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", string(SortByTime), "sort field: time|command")
	cmd.Flags().BoolVar(&keepSecrets, "keep-secrets", false, "do not drop entries that look like they contain inline credentials")
	return cmd
}

func defaultFile() string {
	if env := os.Getenv("ZSH_HISTORY_FILE"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".zsh_history"
	}
	return filepath.Join(home, ".zsh_history")
}

func defaultBackupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".cache", "zsh-history-tidy", "backups"), nil
}
