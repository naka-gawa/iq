package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"iq/internal/dialect"
	"iq/internal/mutation"
	"iq/internal/parser"
	"iq/internal/query"
	"iq/internal/serializer"
	"iq/internal/tui"
)

var (
	colorBool   = color.New(color.FgYellow)
	colorNumber = color.New(color.FgCyan)
	colorNull   = color.New(color.Faint)
)

// printResult writes a single query result to w with TTY-aware syntax highlighting.
// color is suppressed automatically when w is not a terminal (fatih/color behaviour).
func printResult(w io.Writer, v any, formatted string) {
	switch v.(type) {
	case bool:
		colorBool.Fprintln(w, formatted)
	case int, float64:
		colorNumber.Fprintln(w, formatted)
	default:
		if formatted == "null" {
			colorNull.Fprintln(w, formatted)
		} else {
			fmt.Fprintln(w, formatted)
		}
	}
}

var (
	version  string
	revision string
)

// NewRootCommand builds the root cobra command with all CLI flags registered.
func NewRootCommand() *cobra.Command {
	var (
		isInPlace   bool
		interactive bool
		outputFmt   string
		rawStrings  bool
	)

	cmd := &cobra.Command{
		Use:          "iq [flags] EXPR [FILE]",
		Short:        "iq is an INI query tool",
		Long:         `iq is a fast and flexible CLI tool for parsing INI files.`,
		SilenceUsage: true,
		Args:         cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !interactive {
				return cmd.Help()
			}
			expr := ""
			filePath := ""
			if interactive {
				if len(args) > 0 {
					filePath = args[0]
				}
			} else {
				expr = args[0]
				if len(args) > 1 {
					filePath = args[1]
				}
			}
			return dispatch(cmd, expr, filePath, isInPlace, interactive, outputFmt, rawStrings)
		},
	}

	cmd.Flags().BoolVarP(&isInPlace, "in-place", "i", false, "write result back to the original file")
	cmd.Flags().BoolVarP(&interactive, "interactive", "I", false, "launch interactive query mode (TUI)")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "ini", "output format: ini or json")
	cmd.Flags().BoolVar(&rawStrings, "raw-strings", false, "disable JSON type coercion")
	// --profile and --ignore-case are accepted but not yet wired (post-MVP).
	cmd.Flags().String("profile", "generic", "dialect profile override")
	cmd.Flags().Bool("ignore-case", false, "case-insensitive key matching")

	return cmd
}

// Execute executes the root command.
func Execute() error {
	rootCmd := NewRootCommand()
	rootCmd.AddCommand(newVersionCommand())
	return rootCmd.Execute()
}

// SetVersionInfo sets the version and revision information.
func SetVersionInfo(v, r string) {
	version = v
	revision = r
}

// newVersionCommand creates the version subcommand.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of iq",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "iq version %s, revision %s\n", version, revision)
		},
	}
}

// dispatch routes a parsed request to the correct pipeline stage.
func dispatch(cmd *cobra.Command, expr, filePath string, isInPlace, interactive bool, outputFmt string, rawStrings bool) error {
	if interactive && isInPlace {
		return fmt.Errorf("--interactive and --in-place cannot be used together")
	}
	if interactive {
		if filePath == "" {
			return fmt.Errorf("--interactive requires a file argument")
		}
		f, err := parser.Parse(filePath, dialect.ProfileGeneric)
		if err != nil {
			return fmt.Errorf("parsing file for interactive mode: %w", err)
		}
		chosen, err := tui.Run(f, filePath)
		if err != nil {
			return fmt.Errorf("running interactive query: %w", err)
		}
		if chosen != "" {
			fmt.Fprintln(cmd.OutOrStdout(), chosen)
		}
		return nil
	}

	f, err := parser.Parse(filePath, dialect.ProfileGeneric)
	if err != nil {
		return err
	}

	// Mutation path: --in-place + assignment or deletion expression.
	if isInPlace && isMutationExpr(expr) {
		targets, err := parseMutationTargets(expr)
		if err != nil {
			return err
		}
		if err := mutation.Apply(f, targets); err != nil {
			return err
		}
		if filePath == "" || filePath == "-" {
			// In-place on stdin is a no-op for stdout; write INI to stdout instead.
			return serializer.WriteINI(f, cmd.OutOrStdout())
		}
		return serializer.WriteInPlace(f, filePath)
	}

	// JSON conversion path.
	if outputFmt == "json" {
		return serializer.WriteJSON(f, cmd.OutOrStdout(), rawStrings)
	}

	// Query path.
	results, err := query.Execute(f, expr)
	if err != nil {
		return err
	}
	for _, v := range results {
		s, fmtErr := query.FormatValue(v)
		if fmtErr != nil {
			return fmtErr
		}
		printResult(cmd.OutOrStdout(), v, s)
	}
	return nil
}

// isMutationExpr reports whether expr contains an assignment or deletion.
func isMutationExpr(expr string) bool {
	for _, part := range strings.Split(expr, "|") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "del(") {
			return true
		}
		// Assignment: contains " = " with spaces to avoid matching jq operators like "==".
		if strings.Contains(part, " = ") {
			return true
		}
	}
	return false
}

// parseMutationTargets converts a mutation expression into a slice of Targets.
// Handles pipe-separated expressions: ".a.b = x | del(.c.d)".
func parseMutationTargets(expr string) ([]mutation.Target, error) {
	var targets []mutation.Target
	for _, part := range strings.Split(expr, "|") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var t mutation.Target
		var err error
		if strings.HasPrefix(part, "del(") {
			t, err = parseDeletion(part)
		} else {
			t, err = parseAssignment(part)
		}
		if err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// parseAssignment parses ".section.key = value" into a Target.
func parseAssignment(expr string) (mutation.Target, error) {
	idx := strings.Index(expr, " = ")
	if idx < 0 {
		return mutation.Target{}, fmt.Errorf("invalid assignment expression: %q", expr)
	}
	lhs := strings.TrimSpace(expr[:idx])
	rhs := strings.TrimSpace(expr[idx+3:])

	section, key, err := parseDotPath(lhs)
	if err != nil {
		return mutation.Target{}, err
	}

	val := resolveValue(rhs)
	return mutation.Target{Section: section, Key: key, NewVal: val}, nil
}

// parseDeletion parses "del(.section.key)" or "del(.section)" into a Target.
func parseDeletion(expr string) (mutation.Target, error) {
	inner := strings.TrimPrefix(expr, "del(")
	inner = strings.TrimSuffix(inner, ")")
	inner = strings.TrimSpace(inner)

	section, key, err := parseDotPath(inner)
	if err != nil {
		return mutation.Target{}, err
	}
	return mutation.Target{Section: section, Key: key, NewVal: nil}, nil
}

// parseDotPath splits a dot-notation path like ".section.key" into (section, key).
// Bracket notation like .["section"]["key"] is not supported in MVP.
func parseDotPath(path string) (section, key string, err error) {
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return "", "", fmt.Errorf("empty path")
	}
	parts := strings.SplitN(path, ".", 2)
	section = parts[0]
	if len(parts) == 2 {
		key = parts[1]
	}
	return section, key, nil
}

// resolveValue resolves a right-hand side value, handling quoted strings and strenv().
func resolveValue(rhs string) string {
	// strenv(VAR) → read from environment.
	if strings.HasPrefix(rhs, "strenv(") && strings.HasSuffix(rhs, ")") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(rhs, "strenv("), ")")
		return os.Getenv(envVar)
	}
	// Quoted string → unquote.
	if len(rhs) >= 2 && rhs[0] == '"' && rhs[len(rhs)-1] == '"' {
		return rhs[1 : len(rhs)-1]
	}
	return rhs
}
