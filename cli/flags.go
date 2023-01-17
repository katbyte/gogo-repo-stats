package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagData struct {
	Token         string
	Org           string
	Owner         string
	Repo          string
	ProjectNumber int
	Authors       []string
	CachePath     string
	// FullFetch bool todo
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.StringVarP(&flags.Token, "token", "t", "", "github oauth token (GITHUB_TOKEN)")
	pflags.StringVarP(&flags.Org, "org", "o", "", "github organization (GITHUB_ORG)") // nolint: misspell
	pflags.StringVarP(&flags.Owner, "owner", "", "", "github repo owner, defaults to org (GITHUB_OWNER)")
	pflags.StringVarP(&flags.Repo, "repo", "r", "", "github repo name (GITHUB_REPO)")
	pflags.IntVarP(&flags.ProjectNumber, "project-number", "p", 0, "github project number (GITHUB_PROJECT_NUMBER)")
	pflags.StringSliceVarP(&flags.Authors, "authors", "a", nil, "only sync prs by these authors. ie 'katbyte,author2,author3'")
	pflags.StringVarP(&flags.CachePath, "cache", "c", "", "path to sqllite3 db to use as cache")
	// pflags.BoolVarP(&flags.FullFetch, "full", "f", false, "do a full fetch and not abort")

	// binding map for viper/pflag -> env
	m := map[string]string{
		"token":          "GITHUB_TOKEN",
		"org":            "GITHUB_ORG",
		"owner":          "GITHUB_OWNER",
		"repo":           "GITHUB_REPO",
		"project-number": "GITHUB_PROJECT_NUMBER",
		"authors":        "GITHUB_AUTHORS",
		"cache":          "CACHE_DB_FILE",
	}

	for name, env := range m {
		if err := viper.BindPFlag(name, pflags.Lookup(name)); err != nil {
			return fmt.Errorf("error binding '%s' flag: %w", name, err)
		}

		if env != "" {
			if err := viper.BindEnv(name, env); err != nil {
				return fmt.Errorf("error binding '%s' to env '%s' : %w", name, env, err)
			}
		}
	}

	return nil
}

func GetFlags() FlagData {
	owner := viper.GetString("owner")
	if owner == "" {
		owner = viper.GetString("org")
	}

	// for some reason we don't get a proper array back from viper for authors so fix it
	authors := viper.GetStringSlice("authors")
	if len(authors) != 0 {
		authors = strings.Split(authors[0], ",")
	}

	// there has to be an easier way....
	return FlagData{
		Token:         viper.GetString("token"),
		Org:           viper.GetString("org"),
		Owner:         owner,
		Repo:          viper.GetString("repo"),
		ProjectNumber: viper.GetInt("project-number"),
		Authors:       authors,
		CachePath:     viper.GetString("cache"),
	}
}
