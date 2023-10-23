package gh

import (
	"fmt"
	"sort"

	"github.com/google/go-github/v45/github"
	"github.com/katbyte/gogo-repo-stats/lib/clog"
)

func (r Repo) ListAllIssueEvents(number int, cb func([]*github.Timeline, *github.Response) error) error {
	client, ctx := r.NewClient()

	opts := &github.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		clog.Log.Debugf("Listing all events for %s/%s/%d (Page %d)...", r.Owner, r.Name, number, opts.Page)
		events, resp, err := client.Issues.ListIssueTimeline(ctx, r.Owner, r.Name, number, opts)
		if err != nil {

			return fmt.Errorf("unable to list events for %s/%s/%d (Page %d): %w", r.Owner, r.Name, number, opts.Page, err)

		}

		if err = cb(events, resp); err != nil {
			return fmt.Errorf("callback failed for %s/%s/%d (Page %d): %w", r.Owner, r.Name, number, opts.Page, err)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil
}

func (r Repo) GetAllIssueEvents(number int) (*[]github.Timeline, error) {
	var allEvents []github.Timeline

	err := r.ListAllIssueEvents(number, func(events []*github.Timeline, resp *github.Response) error {
		for i, e := range events {
			if e == nil {
				clog.Log.Debugf("events[%d] was nil, skipping", i)
				continue
			}

			if e.CreatedAt == nil && e.SubmittedAt == nil {
				clog.Log.Debugf("events[%d] has no date, skipping", i)
				continue
			}

			allEvents = append(allEvents, *e)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get all prs for %s/%s: %w", r.Owner, r.Name, err)
	}

	// sort ascending?
	sort.Slice(allEvents, func(a, b int) bool {
		dateA := allEvents[a].CreatedAt
		if dateA == nil {
			dateA = allEvents[a].SubmittedAt
		}

		dateB := allEvents[b].CreatedAt
		if dateB == nil {
			dateB = allEvents[b].SubmittedAt
		}

		return dateA.After(*dateB)
	})

	return &allEvents, nil
}
