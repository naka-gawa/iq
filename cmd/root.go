package cmd

import (
	"github.com/spf13/cobra"
)

var (
	version  string
	revision string
)

func NewRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "iq",
		Short: "iq is an INI parser tool",
		Long:  `iq is a fast and flexible CLI tool for parsing INI files.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
}

// Execute executes the root command
func Execute() error {
	rootCmd := NewRootCommand()
	rootCmd.AddCommand(newVersionCommand())

	return rootCmd.Execute()
}

// SetVersionInfo sets the version and revision information
func SetVersionInfo(v, r string) {
	version = v
	revision = r
}

// newVersionCommand creates the version command
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of iq",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("iq version %s, revision %s\n", version, revision)
		},
	}
}
