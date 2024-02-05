package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	c "github.com/gookit/color" // nolint:misspell
	"github.com/katbyte/gogo-repo-stats/lib/cache"
	"github.com/katbyte/gogo-repo-stats/lib/gh"
	"github.com/spf13/cobra"
)

func CmdGraphs(_ *cobra.Command, args []string) error {
	var err error
	f := GetFlags()

	// todo add to flags
	outPath := "graphs"

	// ensure path exists
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		err := os.MkdirAll(outPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create path: %w", err)
		}
	}

	// default to past year
	from := time.Now().AddDate(-1, 0, 0)
	from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
	to := time.Now()

	aragc := len(args)
	if aragc > 1 {
		from, err = time.Parse("2006-01", args[0])
		if err != nil {
			return fmt.Errorf("failed to parse time %s : %w", args[0], err)
		}

		if aragc == 2 {
			to, err = time.Parse("2006-01", args[1])
			if err != nil {
				return fmt.Errorf("failed to parse time %s : %w", args[1], err)
			}
		}
	}

	// open cache
	cache, err := cache.Open(f.CachePath)
	if err != nil {
		return fmt.Errorf("opening cache %s: %w", f.CachePath, err)
	}
	defer cache.DB.Close()

	c.Printf("Generating graphs for PRs from <white>%s</> to <white>%s</>...\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
	c.Printf("  for repos: <cyan>%s</>\n", strings.Join(f.Repos, "</>, <cyan>"))
	if len(f.Authors) > 0 {
		c.Printf("  for authors: <green>%s</>\n", strings.Join(f.Authors, "</>, <green>"))
	}

	for _, repo := range f.Repos {
		repoPath := outPath + "/" + gh.RepoShortName(repo)

		c.Printf("  <cyan>%s</> -> %s:\n", repo, repoPath)

		// ensure path exists
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			err := os.MkdirAll(repoPath, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to create repo path: %w", err)
			}
		}

		// todo collect these in an array and then do each one in a loop
		if err = GraphsRepoOpenedPRsDaily(cache, repoPath, from, to, []string{repo}); err != nil {
			return fmt.Errorf("failed to generate daily pr graphs path: %w", err)
		}
		if err = GraphRepoTotalPRsDaily(cache, repoPath, from, to, []string{repo}); err != nil {
			return fmt.Errorf("failed to generate daily total pr graphs path: %w", err)
		}
		if err = GraphRepoOpenPRsDaily(cache, repoPath, from, to, []string{repo}); err != nil {
			return fmt.Errorf("failed to generate daily open pr graphs path: %w", err)
		}
		/*if err = GraphRepoOpenPRsByAuthorsDaily(cache, repoPath, from, to, []string{repo}); err != nil {
			return fmt.Errorf("failed to generate daily open pr by author graphs path: %w", err)
		}*/

		/*if err = GraphRepoWeeklyPrStats(repo, cache, repoPath, from, to); err != nil {
			return fmt.Errorf("failed to generate daily pr graphs path: %w", err)
		}*/

		// todo daily "how long prs waited"?

		if err = GraphRepoOpenPRsDailyByType(cache, repoPath, from, to, []string{repo}); err != nil {
			return fmt.Errorf("failed to generate daily open issues by type graphs path: %w", err)
		}
	}

	c.Printf("  <magenta>MultiRepo graphs</>...\n")
	if err = GraphMultiRepoTotalPRsDaily(cache, outPath, from, to, f.Repos); err != nil {
		return fmt.Errorf("failed to generate daily pr graphs path: %w", err)
	}
	if err = GraphMultiRepoOpenPRsDaily(cache, outPath, from, to, f.Repos); err != nil {
		return fmt.Errorf("failed to generate daily pr graphs path: %w", err)
	}

	/*
		if err = GraphRepoDailyTotalPRs(cache, outPath, from, to, nil); err != nil {
			return fmt.Errorf("failed to generate daily total pr graphs path: %w", err)
		}
		if err = GraphRepoDailyOpenPRs(cache, outPath, from, to, nil); err != nil {
			return fmt.Errorf("failed to generate daily open pr graphs path: %w", err)
		}*/

	return nil
}
