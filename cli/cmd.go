package cli

import (
	"fmt"

	"github.com/katbyte/gogo-repo-stats/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/mattn/go-sqlite3"
	//nolint:misspell
)

func ValidateParams(params []string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, p := range params {
			if viper.GetString(p) == "" {
				return fmt.Errorf(p + " parameter can't be empty")
			}
		}

		return nil
	}
}

func Make(cmdName string) (*cobra.Command, error) {
	// todo should this be a no-op to avoid accidentally triggering broken builds on malformed commands ?
	root := &cobra.Command{
		Use:           cmdName + " [command]",
		Short:         cmdName + "is a small utility to TODO",
		Long:          `TODO`,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "org", "repo", "cache"}),
		RunE: func(cmd *cobra.Command, args []string) error {

			// f := GetFlags()
			// r := gh.NewRepo(f.Owner, f.Repo, f.Token)

			// what should default be?

			// fetch
			// calculate
			// stats

			// ??

			return nil
		},
	}

	root.AddCommand(&cobra.Command{
		Use:           "fetch",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"token", "org", "repo", "cache"}),
		RunE:          CmdFetch,
	})

	root.AddCommand(&cobra.Command{
		Use:           "report [YYYY-MM] [YYYY-MM]",
		Short:         cmdName + " calculates a report for a given month range. defaults to last month till now. single date is then to now. 2 dates is range",
		Args:          cobra.MaximumNArgs(2),
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"cache"}),
		RunE:          CmdReport,
	})

	root.AddCommand(&cobra.Command{
		Use:           "graphs",
		Args:          cobra.MaximumNArgs(2),
		SilenceErrors: true,
		RunE:          CmdGraphs,
	})

	// todo emoji stats/counter

	root.AddCommand(&cobra.Command{
		Use:           "version",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmdName + " v" + version.Version + "-" + version.GitCommit)
		},
	})

	if err := configureFlags(root); err != nil {
		return nil, fmt.Errorf("unable to configure flags: %w", err)
	}

	return root, nil
}
