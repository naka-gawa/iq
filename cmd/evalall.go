package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"iq/internal/dialect"
	"iq/internal/merge"
	"iq/internal/parser"
	"iq/internal/query"
	"iq/internal/serializer"
)

// newEvalAllCommand builds the `eval-all` subcommand, which deep-merges two or
// more INI files under a configurable conflict policy.
func newEvalAllCommand() *cobra.Command {
	var (
		mergeOverwrite bool
		mergeAppend    bool
		mergeStrict    bool
		outputFmt      string
		rawStrings     bool
	)

	cmd := &cobra.Command{
		Use:   "eval-all [flags] FILE FILE...",
		Short: "Merge multiple INI files into one",
		Long: `eval-all deep-merges two or more INI files in order, section by section.

Keys in later files overwrite their counterparts in earlier files; keys that
exist only in an earlier file are preserved. The conflict policy is selectable:

  --merge-overwrite  later files win (default)
  --merge-append     union conflicting values into an array
  --merge-strict     error when files disagree on a key

Merging synthesizes a new document, so comments from the source files are not
carried over.`,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			policy, err := resolvePolicy(mergeOverwrite, mergeAppend, mergeStrict)
			if err != nil {
				return err
			}
			return runEvalAll(cmd, args, policy, outputFmt, rawStrings)
		},
	}

	cmd.Flags().BoolVar(&mergeOverwrite, "merge-overwrite", false, "later files win on conflict (default)")
	cmd.Flags().BoolVar(&mergeAppend, "merge-append", false, "union conflicting values into an array")
	cmd.Flags().BoolVar(&mergeStrict, "merge-strict", false, "error when files disagree on a key")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "ini", "output format: ini or json")
	cmd.Flags().BoolVar(&rawStrings, "raw-strings", false, "disable JSON type coercion")
	cmd.Flags().String("profile", "generic", "dialect profile: generic, systemd, gitconfig")
	cmd.Flags().Bool("ignore-case", false, "case-insensitive key matching")

	return cmd
}

// resolvePolicy maps the mutually exclusive merge flags to a merge.Policy.
func resolvePolicy(overwrite, appendVals, strict bool) (merge.Policy, error) {
	count := 0
	for _, b := range []bool{overwrite, appendVals, strict} {
		if b {
			count++
		}
	}
	if count > 1 {
		return 0, fmt.Errorf("--merge-overwrite, --merge-append, and --merge-strict are mutually exclusive")
	}
	switch {
	case appendVals:
		return merge.PolicyAppend, nil
	case strict:
		return merge.PolicyStrict, nil
	default:
		return merge.PolicyOverwrite, nil
	}
}

// runEvalAll parses every input file into a normalized map, merges them, and
// writes the result as INI or JSON.
func runEvalAll(cmd *cobra.Command, files []string, policy merge.Policy, outputFmt string, rawStrings bool) error {
	docs := make([]map[string]any, 0, len(files))
	for _, path := range files {
		prof, err := resolveProfile(cmd, path)
		if err != nil {
			return err
		}
		opts := prof.LoadOptions()
		if ignoreCase, _ := cmd.Flags().GetBool("ignore-case"); ignoreCase {
			opts.Insensitive = true
		}

		f, err := parser.ParseWithOptions(path, opts)
		if err != nil {
			return err
		}
		docs = append(docs, dialect.TransformMap(prof, query.ToMap(f)))
	}

	merged, err := merge.Merge(docs, policy)
	if err != nil {
		return err
	}

	if outputFmt == "json" {
		return serializer.WriteMergedJSON(merged, cmd.OutOrStdout(), rawStrings)
	}
	return serializer.WriteMergedINI(merged, cmd.OutOrStdout())
}
