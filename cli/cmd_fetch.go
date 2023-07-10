package cli

import (
	"fmt"

	"github.com/google/go-github/v45/github"
	c "github.com/gookit/color" // nolint: misspell
	"github.com/katbyte/gogo-repo-stats/lib/cache"
	"github.com/katbyte/gogo-repo-stats/lib/gh"
	"github.com/spf13/cobra"
)

func CmdFetch(_ *cobra.Command, _ []string) error {
	f := GetFlags()

	// open cache
	cache, err := cache.Open(f.CachePath)
	if err != nil {
		return fmt.Errorf("opening cache %s: %w", f.CachePath, err)
	}
	defer cache.DB.Close()

	full := false // todo full get everything mode

	for _, repo := range f.Repos {
		r, err := gh.NewRepo(repo, f.Token)
		if err != nil {
			return fmt.Errorf("creating repo %s: %w", repo, err)
		}

		// for each PR, check if cached, if not insert && update
		count := 0
		c.Printf("Retrieving all prs for <white>%s</>/<cyan>%s</>...\n", r.Owner, r.Name)
		err = r.ListAllPullRequests("all", func(prs []*github.PullRequest, resp *github.Response) error {
			for i, p := range prs {
				count++

				if p == nil {
					c.Printf(" <red>prs[%d] was nil</>, skipping", i)
					continue
				}

				n := p.GetNumber()
				if n == 0 {
					c.Printf(" <red>prs[%d].Number was nil</>, skipping", i)
					continue
				}

				// check cache
				cpr, err := cache.GetPR(repo, n)
				if err == nil {
					// if cached && closed (in cache) we have all relevant data
					if cpr != nil && cpr.State != "open" {
						// but check events, if zero we likely should get all events again
						cevents, err := cache.GetEventsForPR(repo, n)
						if err != nil {
							return fmt.Errorf("failed to get events from cache %s/%s/%d: %w", r.Owner, r.Name, n, err)
						}

						c.Printf(" pr <cyan>#%d</> <darkGray>(%d @ %s)</>: %s\n", n, count, p.GetCreatedAt().Format("2006-01-02"), p.GetTitle())
						c.Printf("   CACHED! with <green>%d</> events\n", len(cevents))

						if len(cevents) != 0 && !full {
							continue
						}
					}
				}

				if p.GetState() == "open" {
					c.Printf(" pr <yellow>#%d</> <darkGray>(%d @ %s)</>: %s\n", n, count, p.GetCreatedAt().Format("2006-01-02"), p.GetTitle())
				} else {
					c.Printf(" pr <green>#%d</> <darkGray>(%d @ %s)</>: %s\n", n, count, p.GetCreatedAt().Format("2006-01-02"), p.GetTitle())
				}

				gh, ctx := r.NewClient()
				pr, _, err := gh.PullRequests.Get(ctx, r.Owner, r.Name, n)
				if err != nil {
					return fmt.Errorf("failed to get pr from GH %s/%s/%d: %w", r.Owner, r.Name, n, err)
				}

				err = cache.UpsertRepoPRFromGH(repo, pr)
				if err != nil {
					return fmt.Errorf("cache upsert failed: %w", err)
				}

				// get and store events
				events, err := r.GetAllIssueEvents(*pr.Number)
				if err != nil {
					return fmt.Errorf("failed to get events from GH %s/%s/%d: %w", r.Owner, r.Name, n, err)
				}

				c.Printf("   <darkGray>events:</> ")
				for _, t := range *events {
					c.Printf("%s, ", t.GetEvent())

					err = cache.UpsertEvent(repo, n, &t)
					if err != nil {
						return fmt.Errorf("cache upsert failed: %w", err)
					}
				}
				c.Printf("\n")

				// now that we have PR and events in the cache, we can calculate stats:
				daysOpen, daysWaiting, daysToFirst, err := cache.ComputeAndUpdatePRStats(repo, pr.GetNumber())
				if err != nil {
					return fmt.Errorf("falied to compute and update stats: %w", err)
				}
				c.Printf("   <darkGray>days</> open: <green>%.2f</> waiting: <green>%.2f</> first: <green>%.2f</> \n", *daysOpen, *daysWaiting, *daysToFirst)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to get all prs for %s/%s: %w", r.Owner, r.Name, err)
		}
	}
	return nil
}
