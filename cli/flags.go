package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagData struct {
	Token     string
	Repos     []string
	Authors   []string
	CachePath string
	// FullFetch bool todo
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.StringVarP(&flags.Token, "token", "t", "", "github oauth token (GITHUB_TOKEN)")
	pflags.StringSliceVarP(&flags.Repos, "repos", "r", nil, "repos to fetch data for in the format owner/repo. ie 'katbyte/tctest,katbyte/terrafmt'")
	pflags.StringSliceVarP(&flags.Authors, "authors", "a", nil, "only sync prs by these authors. ie 'katbyte,author2,author3'")
	pflags.StringVarP(&flags.CachePath, "cache", "c", "", "path to sqllite3 db to use as cache")
	// pflags.BoolVarP(&flags.FullFetch, "full", "f", false, "do a full fetch and not abort")

	// binding map for viper/pflag -> env
	m := map[string]string{
		"token":   "GITHUB_TOKEN",
		"repos":   "GITHUB_REPOS",
		"authors": "GITHUB_AUTHORS",
		"cache":   "CACHE_DB_FILE",
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

	// for some reason we don't get a proper array back from viper for authors so fix it liek this for now TODO FIX
	authors := viper.GetStringSlice("authors")
	if len(authors) != 0 {
		authors = strings.Split(authors[0], ",")
	}

	// for some reason we don't get a proper array back from viper for authors so fix it liek this for now TODO FIX
	repos := viper.GetStringSlice("repos")
	if len(repos) != 0 {
		repos = strings.Split(repos[0], ",")
	}

	// there has to be an easier way....
	return FlagData{
		Token:     viper.GetString("token"),
		Repos:     repos,
		Authors:   authors,
		CachePath: viper.GetString("cache"),
	}
}
